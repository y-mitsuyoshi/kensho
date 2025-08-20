package kensho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// ErrUnsupportedDocumentType is returned when the document type is not supported.
var ErrUnsupportedDocumentType = errors.New("unsupported document type")

// ErrUnsupportedMimeType is returned when the MIME type of a file is not supported.
var ErrUnsupportedMimeType = errors.New("unsupported MIME type")

// ErrRequestBodyTooLarge is returned when the request body is larger than the allowed limit.
var ErrRequestBodyTooLarge = errors.New("request body too large")

// ErrMissingField is returned when a required field is missing from the request.
var ErrMissingField = errors.New("missing required field")

var supportedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

// GenerativeModel is an interface that abstracts the genai.GenerativeModel.
type GenerativeModel interface {
	GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error)
}

// Client holds the genai client and configuration.
type Client struct {
	genaiClient     *genai.Client
	generativeModel GenerativeModel
	config          *Config
}

// NewClient creates a new client for the Gemini API using the default embedded configuration.
func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	config, err := loadDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}
	return NewClientWithConfig(ctx, apiKey, *config)
}

// NewClientWithConfigPath creates a new client for the Gemini API using a configuration file from the specified path.
func NewClientWithConfigPath(ctx context.Context, apiKey string, configPath string) (*Client, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from path %s: %w", configPath, err)
	}
	return NewClientWithConfig(ctx, apiKey, *config)
}

// NewClientWithConfig creates a new client for the Gemini API with a provided configuration struct.
func NewClientWithConfig(ctx context.Context, apiKey string, config Config) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	model := client.GenerativeModel("gemini-2.5-pro")
	return &Client{
		genaiClient:     client,
		generativeModel: model,
		config:          &config,
	}, nil
}

// Close closes the underlying genai client.
func (c *Client) Close() {
	if err := c.genaiClient.Close(); err != nil {
		log.Printf("failed to close genai client: %v", err)
	}
}

// FilePart represents a file part with its content and MIME type.
type FilePart struct {
	Content  []byte
	MimeType string
}

// ParseRequest parses a multipart HTTP request to extract the document type and file parts.
// It enforces a request body size limit of 100MB.
func ParseRequest(r *http.Request) (string, map[string]FilePart, error) {
	if r.Method != http.MethodPost {
		return "", nil, fmt.Errorf("invalid request method: %s", r.Method)
	}

	// Limit request body to 100MB to avoid OOM
	r.Body = http.MaxBytesReader(nil, r.Body, 100<<20)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		if err == http.ErrBodyReadAfterClose || err.Error() == "http: request body too large" {
			return "", nil, ErrRequestBodyTooLarge
		}
		return "", nil, fmt.Errorf("could not parse multipart form: %w", err)
	}

	docType := r.FormValue("document_type")
	if docType == "" {
		return "", nil, fmt.Errorf("%w: document_type", ErrMissingField)
	}

	fileParts := make(map[string]FilePart)
	for name, headers := range r.MultipartForm.File {
		if !strings.HasPrefix(name, "image_") {
			continue
		}
		if len(headers) == 0 {
			continue
		}
		header := headers[0]

		file, err := header.Open()
		if err != nil {
			return "", nil, fmt.Errorf("could not open file part %s: %w", name, err)
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return "", nil, fmt.Errorf("could not read file part %s: %w", name, err)
		}

		if len(content) > 0 {
			key := strings.TrimPrefix(name, "image_")
			fileParts[key] = FilePart{
				Content:  content,
				MimeType: header.Header.Get("Content-Type"),
			}
		}
	}

	if len(fileParts) == 0 {
		return "", nil, fmt.Errorf("%w: at least one image is required", ErrMissingField)
	}

	return docType, fileParts, nil
}

// Extract sends one or more files to the Gemini API, asks it to extract information,
// and returns the result as a map.
func (c *Client) Extract(ctx context.Context, fileParts map[string]FilePart, docType string) (map[string]interface{}, error) {
	doc, ok := c.config.Documents[docType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDocumentType, docType)
	}

	prompt := []genai.Part{
		genai.Text(doc.Prompt),
	}

	for _, partName := range doc.ImageParts {
		part, ok := fileParts[partName]
		if !ok {
			// This allows for optional parts, like the back of a driver's license
			continue
		}

		mimeType := cleanMimeType(part.MimeType)
		if !supportedMimeTypes[mimeType] {
			detectedMimeType := http.DetectContentType(part.Content)
			if !supportedMimeTypes[detectedMimeType] {
				return nil, fmt.Errorf("%w: %s", ErrUnsupportedMimeType, mimeType)
			}
			mimeType = detectedMimeType
		}

		processedContent, err := c.preprocessContent(part.Content, mimeType)
		if err != nil {
			log.Printf("could not preprocess image part %s: %v, using original", partName, err)
			processedContent = part.Content
		}

		prompt = append(prompt, genai.Text(fmt.Sprintf("\nFile part: %s", partName)))
		prompt = append(prompt, genai.Blob{MIMEType: mimeType, Data: processedContent})
	}

	resp, err := c.generativeModel.GenerateContent(ctx, prompt...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content generated")
	}

	jsonText, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response format from API")
	}

	cleaned := sanitizeJSONResponse(string(jsonText))
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON from response: %w (raw response: %s)", err, cleaned)
	}

	return data, nil
}

func (c *Client) preprocessContent(content []byte, mimeType string) ([]byte, error) {
	if strings.Contains(mimeType, "pdf") {
		return content, nil // PDF preprocessing is not implemented
	}
	return PreprocessImage(content, mimeType)
}

func cleanMimeType(mimeType string) string {
	mimeType = strings.TrimSpace(mimeType)
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	if strings.Count(mimeType, "image/") > 1 {
		if last := strings.LastIndex(mimeType, "image/"); last != -1 {
			mimeType = mimeType[last:]
		}
	}
	return mimeType
}

// sanitizeJSONResponse attempts to extract a JSON object/array from a string
// that may contain Markdown code fences or surrounding text. It returns the
// inner JSON string if found, otherwise returns the trimmed original.
func sanitizeJSONResponse(s string) string {
	s = strings.TrimSpace(s)

	// If the response contains triple-backtick fences like ```json ... ```
	// remove them by extracting the first JSON-like substring.
	// Find the first brace or bracket and the last matching closing char.
	start := strings.IndexAny(s, "{[")
	end := strings.LastIndexAny(s, "}]")
	if start != -1 && end != -1 && end >= start {
		return s[start : end+1]
	}

	// Fallback: strip backticks and trim whitespace.
	s = strings.Trim(s, "`\n \t\r")
	return s
}
