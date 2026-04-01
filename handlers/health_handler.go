package handlers

import (
	"net/http"
	"synthframe-api/config"
	"synthframe-api/models"

	"github.com/gin-gonic/gin"
)

func HealthHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, models.HealthResponse{
			Status: "ok",
			Model:  "black-forest-labs/FLUX.1-schnell-Free",
			APIKey: cfg.TogetherAPIKey != "",
		})
	}
}
