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
			`SELECT id, storage_key, created_at FROM generated_images
			 WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`,
			userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch images"})
			return
		}
		defer rows.Close()

		type ImageItem struct {
			ID        string    `json:"id"`
			URL       string    `json:"url"`
			CreatedAt time.Time `json:"created_at"`
		}

		var images []ImageItem
		for rows.Next() {
			var item ImageItem
			var key string
			if err := rows.Scan(&item.ID, &key, &item.CreatedAt); err != nil {
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
