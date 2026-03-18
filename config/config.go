package config

import "os"

type Config struct {
	TogetherAPIKey string
	OutputDir      string
	ServerPort     string
	JWTSecret      string
	DatabaseURL    string
	S3Endpoint     string
	S3AccessKey    string
	S3SecretKey    string
	S3Bucket       string
}

func Load() *Config {
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "./outputs"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = ":8080"
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "whisk-secret-key-change-in-production"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres@whisk-db:5432/whisk_db?sslmode=disable"
	}
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "http://whisk-storage:8333"
	}
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "uploads"
	}
	return &Config{
		TogetherAPIKey: os.Getenv("TOGETHER_API_KEY"),
		OutputDir:      outputDir,
		ServerPort:     port,
		JWTSecret:      secret,
		DatabaseURL:    dbURL,
		S3Endpoint:     s3Endpoint,
		S3AccessKey:    os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:    os.Getenv("S3_SECRET_KEY"),
		S3Bucket:       s3Bucket,
	}
}
