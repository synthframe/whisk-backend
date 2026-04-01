package adapters

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type TogetherAI struct {
	APIKey     string
	HTTPClient *http.Client
}

type RefineContextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewTogetherAI(apiKey string) *TogetherAI {
	return &TogetherAI{
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

// Vision: analyze image via Cloudflare Workers AI (LLaVA 1.5)
func (t *TogetherAI) AnalyzeImage(imageBase64 string, slotType string) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"image_base64": imageBase64,
		"slot_type":    slotType,
	})
	if err != nil {
		return "", err
	}

	resp, err := t.HTTPClient.Post(
		"https://whisk-image-gen.gimchan29.workers.dev/analyze",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", fmt.Errorf("cloudflare worker vision request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("cloudflare worker vision error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Prompt == "" {
		return "", fmt.Errorf("empty prompt from vision API")
	}
	return result.Prompt, nil
}

// GenerateImage: generate image via Cloudflare Workers AI (free, no credits required)
func (t *TogetherAI) GenerateImage(prompt string, width, height int) ([]byte, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"prompt": prompt,
		"width":  width,
		"height": height,
	})
	if err != nil {
		return nil, err
	}

	resp, err := t.HTTPClient.Post(
		"https://whisk-image-gen.gimchan29.workers.dev",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("cloudflare worker request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloudflare worker error %d: %s", resp.StatusCode, string(respBody))
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image bytes: %w", err)
	}
	return imgBytes, nil
}

// RefinePrompts calls Cloudflare Workers AI (Llama 3.1 8B) to refine image prompts based on user feedback
func (t *TogetherAI) RefinePrompts(subject, scene, style, feedback string, history []RefineContextMessage) (string, string, string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"subject_prompt": subject,
		"scene_prompt":   scene,
		"style_prompt":   style,
		"feedback":       feedback,
		"history":        history,
	})
	if err != nil {
		return subject, scene, style, err
	}

	resp, err := t.HTTPClient.Post(
		"https://whisk-image-gen.gimchan29.workers.dev/refine-prompt",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return subject, scene, style, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return subject, scene, style, nil
	}

	var result struct {
		SubjectPrompt string `json:"subject_prompt"`
		ScenePrompt   string `json:"scene_prompt"`
		StylePrompt   string `json:"style_prompt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return subject, scene, style, nil
	}

	if result.SubjectPrompt == "" {
		result.SubjectPrompt = subject
	}
	if result.ScenePrompt == "" {
		result.ScenePrompt = scene
	}
	if result.StylePrompt == "" {
		result.StylePrompt = style
	}
	return result.SubjectPrompt, result.ScenePrompt, result.StylePrompt, nil
}

// Img2Img calls Cloudflare Workers AI SD 1.5 img2img to modify an existing image
func (t *TogetherAI) Img2Img(imageBytes []byte, prompt string, strength float64) ([]byte, error) {
	b64 := base64.StdEncoding.EncodeToString(imageBytes)
	payload, err := json.Marshal(map[string]interface{}{
		"image_base64": b64,
		"prompt":       prompt,
		"strength":     strength,
	})
	if err != nil {
		return nil, err
	}

	resp, err := t.HTTPClient.Post(
		"https://whisk-image-gen.gimchan29.workers.dev/img2img",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("img2img worker request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("img2img worker error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func slotPrompt(slotType string) string {
	switch slotType {
	case "subject":
		return "Describe only the main subject/character. Be concise, comma-separated."
	case "scene":
		return "Describe only the setting/background environment. Be concise."
	case "style":
		return "Describe only the artistic style, color palette, and lighting. Be concise."
	default:
		return "Describe the image concisely."
	}
}
