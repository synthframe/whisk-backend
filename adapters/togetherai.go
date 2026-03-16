package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// Vision: analyze image and return text description
func (t *TogetherAI) AnalyzeImage(imageBase64 string, slotType string) (string, error) {
	promptText := slotPrompt(slotType)

	payload := map[string]interface{}{
		"model": "meta-llama/Llama-3.2-11B-Vision-Instruct-Turbo",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": "data:image/jpeg;base64," + imageBase64,
						},
					},
					{
						"type": "text",
						"text": promptText,
					},
				},
			},
		},
		"max_tokens": 200,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.together.xyz/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+t.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("together.ai vision error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from vision API")
	}
	return result.Choices[0].Message.Content, nil
}

// GenerateImage: generate image via Cloudflare Workers AI (free, no credits required)
func (t *TogetherAI) GenerateImage(prompt string) ([]byte, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"prompt": prompt,
		"width":  1024,
		"height": 1024,
	})
	if err != nil {
		return nil, err
	}

	resp, err := t.HTTPClient.Post(
		"https://whisk-image-gen.teamxquare867.workers.dev",
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
