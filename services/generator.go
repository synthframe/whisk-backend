package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"synthframe-api/adapters"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var presetSuffixes = map[string]string{
	"photorealistic": "ultra detailed, 8k, photorealistic, DSLR",
	"cinematic":      "cinematic lighting, movie still, anamorphic lens",
	"anime":          "anime style, studio ghibli, detailed illustration",
	"oil_painting":   "oil on canvas, impressionist brushwork",
	"watercolor":     "watercolor painting, soft edges, translucent",
	"pixel_art":      "pixel art, 16-bit, retro game style",
	"sketched":       "pencil sketch, hand drawn, monochrome",
	"pixar_3d":       "Pixar 3D animation style, vibrant colors, soft lighting, subsurface scattering, high detail, cinematic",
}

type GeneratorService struct {
	adapter *adapters.TogetherAI
	storage *adapters.StorageAdapter
	db      *pgxpool.Pool
	// legacy local storage fallback
	localStorage *Storage
}

func NewGeneratorService(adapter *adapters.TogetherAI, storage *adapters.StorageAdapter, db *pgxpool.Pool) *GeneratorService {
	return &GeneratorService{adapter: adapter, storage: storage, db: db}
}

func (g *GeneratorService) BuildPrompt(subject, scene, style, preset string) string {
	parts := []string{}
	if subject != "" {
		parts = append(parts, subject)
	}
	if scene != "" {
		parts = append(parts, scene)
	}
	if style != "" {
		parts = append(parts, style)
	}
	master := strings.Join(parts, ", ")
	if suffix, ok := presetSuffixes[preset]; ok {
		master += ", " + suffix
	}
	return master
}

func (g *GeneratorService) Generate(subject, scene, style, preset string, width, height int) (string, error) {
	return g.GenerateWithUser(context.Background(), subject, scene, style, preset, width, height, "")
}

func (g *GeneratorService) GenerateWithUser(ctx context.Context, subject, scene, style, preset string, width, height int, userID string) (string, error) {
	prompt := g.BuildPrompt(subject, scene, style, preset)
	if width <= 0 {
		width = 1024
	}
	if height <= 0 {
		height = 1024
	}
	imgBytes, err := g.adapter.GenerateImage(prompt, width, height)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("gen_%s.jpg", uuid.New().String()[:8])

	if g.storage != nil {
		if err := g.storage.Upload(ctx, key, imgBytes, "image/jpeg"); err != nil {
			log.Printf("WARNING: S3 upload failed: %v — falling back to local storage", err)
			// fallback to local storage
			return g.saveLocal(imgBytes, key)
		}
		// Record in DB if user_id is available
		if userID != "" {
			if err := g.SaveImageRecord(ctx, userID, key, subject, scene, style, preset, width, height); err != nil {
				log.Printf("WARNING: failed to save image record: %v", err)
			}
		}
		return key, nil
	}

	return g.saveLocal(imgBytes, key)
}

func (g *GeneratorService) saveLocal(imgBytes []byte, key string) (string, error) {
	if g.localStorage == nil {
		var err error
		g.localStorage, err = NewStorage("./outputs")
		if err != nil {
			return "", fmt.Errorf("local storage init failed: %w", err)
		}
	}
	filename, err := g.localStorage.SaveImage(imgBytes, "gen")
	if err != nil {
		return "", err
	}
	return filename, nil
}

func (g *GeneratorService) SaveImageRecord(ctx context.Context, userID, key, subject, scene, style, preset string, width, height int) error {
	if g.db == nil {
		return nil
	}
	_, err := g.db.Exec(ctx,
		`INSERT INTO generated_images (user_id, storage_key, subject_prompt, scene_prompt, style_prompt, style_preset, width, height)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, key, subject, scene, style, preset, width, height,
	)
	return err
}
