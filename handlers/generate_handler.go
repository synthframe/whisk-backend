package handlers

import (
	"context"
	"net/http"
	"sync"
	"whisk-clone/models"
	"whisk-clone/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type generateJob struct {
	Status   string
	ImageURL string
	Error    string
}

var (
	generateJobs sync.Map
)

func GenerateHandler(generator *services.GeneratorService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.GenerateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id := uuid.New().String()
		userID := c.GetString("user_id")
		generateJobs.Store(id, &generateJob{Status: "processing"})

		go func() {
			key, err := generator.GenerateWithUser(context.Background(), req.SubjectPrompt, req.ScenePrompt, req.StylePrompt, req.StylePreset, req.Width, req.Height, userID)
			if err != nil {
				generateJobs.Store(id, &generateJob{Status: "failed", Error: err.Error()})
				return
			}
			generateJobs.Store(id, &generateJob{Status: "completed", ImageURL: "/outputs/" + key})
		}()

		c.JSON(http.StatusAccepted, models.GenerateResponse{
			ID:     id,
			Status: "processing",
		})
	}
}

func GenerateStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		v, ok := generateJobs.Load(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		job := v.(*generateJob)
		resp := models.GenerateResponse{
			ID:       id,
			Status:   job.Status,
			ImageURL: job.ImageURL,
			Error:    job.Error,
		}
		c.JSON(http.StatusOK, resp)
	}
}
