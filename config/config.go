package config

import "os"

type Config struct {
	TogetherAPIKey string
	OutputDir      string
	ServerPort     string
	JWTSecret      string
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
	return &Config{
		TogetherAPIKey: os.Getenv("TOGETHER_API_KEY"),
		OutputDir:      outputDir,
		ServerPort:     port,
		JWTSecret:      secret,
	}
}
