package ocr

import (
	"context"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Client holds the genai client.
type Client struct {
	genaiClient *genai.GenerativeModel
}

// NewClient creates a new client for the Gemini API.
func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	model := client.GenerativeModel("gemini-1.5-pro")
	return &Client{genaiClient: model}, nil
}

// Close closes the underlying genai client.
func (c *Client) Close(client *genai.Client) {
	if err := client.Close(); err != nil {
		log.Printf("failed to close genai client: %v", err)
	}
}

// ExtractText sends an image to the Gemini API and asks it to extract information.
func (c *Client) ExtractText(ctx context.Context, imageData []byte, mimeType string) (string, error) {
	prompt := []genai.Part{
		genai.Text(`
Please analyze the provided image, which could be a Japanese driver's license (運転免許証) or a My Number Card (マイナンバーカード).
Extract the following fields and return the result in a single, minified JSON object.
If a field is not present, use an empty string "" as its value.

- name (氏名)
- address (住所)
- birth_date (生年月日)
- issue_date (交付日)
- expiry_date (有効期限)
- card_number (免許の番号 or マイナンバー)
- gender (性別) - only for My Number Card. For driver's licenses, this field can be omitted or empty.

Example for a driver's license:
{"name":"見本太郎","address":"東京都千代田区霞が関2-1-1","birth_date":"昭和60年1月1日","issue_date":"平成25年4月1日","expiry_date":"平成30年2月1日","card_number":"第123456789012号"}

Example for a My Number Card:
{"name":"見本太郎","address":"東京都新宿区西新宿2-8-1","birth_date":"1985年1月1日","issue_date":"2016年1月1日","expiry_date":"2026年1月1日","card_number":"123456789012","gender":"男"}
`),
		genai.ImageData(mimeType, imageData),
	}

	resp, err := c.genaiClient.GenerateContent(ctx, prompt...)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	// Assuming the first part of the response is the JSON text
	if jsonText, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		return string(jsonText), nil
	}

	return "", fmt.Errorf("unexpected response format from API")
}
