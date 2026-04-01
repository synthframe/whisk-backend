package main

import (
	"log"
	"net/http"
	"synthframe-api/adapters"
	"synthframe-api/config"
	"synthframe-api/db"
	"synthframe-api/handlers"
	"synthframe-api/middleware"
	"synthframe-api/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if cfg.TogetherAPIKey == "" {
		log.Println("WARNING: TOGETHER_API_KEY is not set")
	}

	// Connect to PostgreSQL
	dbPool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Printf("WARNING: DB connection failed: %v — running without DB", err)
		dbPool = nil
	} else {
		if err := db.Migrate(dbPool); err != nil {
			log.Printf("WARNING: Migration failed: %v", err)
		}
	}

	// Init S3 storage
	storageAdapter := adapters.NewStorageAdapter(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket)

	togetherAdapter := adapters.NewTogetherAI(cfg.TogetherAPIKey)
	visionSvc := services.NewVisionService(togetherAdapter)
	generatorSvc := services.NewGeneratorService(togetherAdapter, storageAdapter, dbPool)
	batchSvc := services.NewBatchService(generatorSvc)
	authSvc := services.NewAuthService(dbPool, cfg.JWTSecret)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// Proxy images from SeaweedFS S3
	r.GET("/outputs/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		data, err := storageAdapter.Download(c.Request.Context(), filename)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		contentType := "image/jpeg"
		if len(filename) >= 4 && filename[len(filename)-4:] == ".png" {
			contentType = "image/png"
		}
		c.Data(http.StatusOK, contentType, data)
	})

	api := r.Group("/api")
	{
		api.GET("/health", handlers.HealthHandler(cfg))
		api.POST("/auth/register", handlers.RegisterHandler(authSvc))
		api.POST("/auth/login", handlers.LoginHandler(authSvc))

		protected := api.Group("/")
		protected.Use(middleware.Auth(authSvc))
		{
			protected.GET("/auth/me", handlers.MeHandler(authSvc))
			protected.POST("/analyze", handlers.AnalyzeHandler(visionSvc))
			protected.POST("/generate", handlers.GenerateHandler(generatorSvc))
			protected.GET("/generate/:id", handlers.GenerateStatusHandler())
			protected.POST("/batch", handlers.BatchCreateHandler(batchSvc))
			protected.GET("/batch/:id", handlers.BatchStatusHandler())
			protected.GET("/batch/:id/stream", handlers.BatchStreamHandler())
			protected.GET("/images", handlers.UserImagesHandler(dbPool))
			protected.POST("/refine", handlers.RefineHandler(togetherAdapter, generatorSvc, storageAdapter))
		}
	}

	log.Printf("Server starting on %s", cfg.ServerPort)
	if err := r.Run(cfg.ServerPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
