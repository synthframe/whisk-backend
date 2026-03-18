package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type TogetherAI struct {
	APIKey     string
	HTTPClient *http.Client
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

// RefinePrompts uses Together AI LLM to refine image prompts based on user feedback
func (t *TogetherAI) RefinePrompts(subject, scene, style, feedback string) (string, string, string, error) {
	if t.APIKey == "" {
		return subject, scene, style, nil
	}

	systemMsg := `You are an AI image generation prompt refinement assistant. Given current image prompts and user feedback, output ONLY a JSON object with keys "subject_prompt", "scene_prompt", "style_prompt". Keep each value concise (under 50 words). Output valid JSON only, no explanation.`
	userMsg := fmt.Sprintf("Subject: %s\nScene: %s\nStyle: %s\n\nUser feedback: %s", subject, scene, style, feedback)

	payload, err := json.Marshal(map[string]interface{}{
		"model": "meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo",
		"messages": []map[string]string{
			{"role": "system", "content": systemMsg},
			{"role": "user", "content": userMsg},
		},
		"max_tokens":  300,
		"temperature": 0.4,
	})
	if err != nil {
		return subject, scene, style, err
	}

	req, err := http.NewRequest("POST", "https://api.together.xyz/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return subject, scene, style, err
	}
	req.Header.Set("Authorization", "Bearer "+t.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		return subject, scene, style, err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Choices) == 0 {
		return subject, scene, style, nil
	}

	content := result.Choices[0].Message.Content
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return subject, scene, style, nil
	}

	var prompts struct {
		Subject string `json:"subject_prompt"`
		Scene   string `json:"scene_prompt"`
		Style   string `json:"style_prompt"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &prompts); err != nil {
		return subject, scene, style, nil
	}

	if prompts.Subject == "" {
		prompts.Subject = subject
	}
	if prompts.Scene == "" {
		prompts.Scene = scene
	}
	if prompts.Style == "" {
		prompts.Style = style
	}
	return prompts.Subject, prompts.Scene, prompts.Style, nil
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
