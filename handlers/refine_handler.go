package handlers

import (
	"net/http"
	"whisk-clone/adapters"
	"whisk-clone/models"
	"whisk-clone/services"

	"github.com/gin-gonic/gin"
)

func RefineHandler(adapter *adapters.TogetherAI, generator *services.GeneratorService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.RefineRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Refine prompts using LLM
		newSubject, newScene, newStyle, err := adapter.RefinePrompts(
			req.SubjectPrompt, req.ScenePrompt, req.StylePrompt, req.Feedback,
		)
		if err != nil {
			// fallback to original prompts on LLM error
			newSubject = req.SubjectPrompt
			newScene = req.ScenePrompt
			newStyle = req.StylePrompt
		}

		width := req.Width
		if width <= 0 {
			width = 1024
		}
		height := req.Height
		if height <= 0 {
			height = 1024
		}

		userID := c.GetString("user_id")
		filename, err := generator.GenerateWithUser(c.Request.Context(), newSubject, newScene, newStyle, req.StylePreset, width, height, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, models.RefineResponse{
			ImageURL:      "/outputs/" + filename,
			SubjectPrompt: newSubject,
			ScenePrompt:   newScene,
			StylePrompt:   newStyle,
		})
	}
}
