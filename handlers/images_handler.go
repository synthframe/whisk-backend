package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func UserImagesHandler(dbPool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if dbPool == nil {
			c.JSON(http.StatusOK, []interface{}{})
			return
		}

		userID := c.GetString("user_id")
		rows, err := dbPool.Query(c.Request.Context(),
			`SELECT id, storage_key, subject_prompt, scene_prompt, style_prompt, style_preset, width, height, created_at
			 FROM generated_images
			 WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`,
			userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch images"})
			return
		}
		defer rows.Close()

		type ImageItem struct {
			ID            string    `json:"id"`
			URL           string    `json:"url"`
			SubjectPrompt string    `json:"subject_prompt"`
			ScenePrompt   string    `json:"scene_prompt"`
			StylePrompt   string    `json:"style_prompt"`
			StylePreset   string    `json:"style_preset"`
			Width         int       `json:"width"`
			Height        int       `json:"height"`
			CreatedAt     time.Time `json:"created_at"`
		}

		var images []ImageItem
		for rows.Next() {
			var item ImageItem
			var key string
			if err := rows.Scan(&item.ID, &key, &item.SubjectPrompt, &item.ScenePrompt, &item.StylePrompt, &item.StylePreset, &item.Width, &item.Height, &item.CreatedAt); err != nil {
				continue
			}
			item.URL = "/outputs/" + key
			images = append(images, item)
		}
		if images == nil {
			images = []ImageItem{}
		}
		c.JSON(http.StatusOK, images)
	}
}
