package kensho

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/google/generative-ai-go/genai"
)

// mockGenerativeModel is a mock implementation of the GenerativeModel interface.
type mockGenerativeModel struct {
	GenerateContentFunc func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error)
}

func (m *mockGenerativeModel) GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, parts...)
	}
	return nil, errors.New("GenerateContentFunc not implemented")
}

func TestNewClient(t *testing.T) {
	t.Run("should return error when api key is not set", func(t *testing.T) {
		_, err := NewClient(context.Background(), "")
		if err == nil {
			t.Error("expected error, but got nil")
		}
	})

	t.Run("should create new client successfully with embedded config", func(t *testing.T) {
		// This test depends on the real genai.NewClient, but since we can't easily mock it
		// without a complex interface, we'll just ensure no error is returned.
		// A more advanced setup might use dependency injection for the genai client itself.
		if os.Getenv("GEMINI_API_KEY") == "" {
			t.Skip("GEMINI_API_KEY not set, skipping integration-like test")
		}
		client, err := NewClient(context.Background(), os.Getenv("GEMINI_API_KEY"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if client == nil {
			t.Error("expected client to be non-nil")
		} else {
			client.Close()
		}
	})

	t.Run("should create new client successfully with custom config path", func(t *testing.T) {
		configContent := `
documents:
  test_doc:
    prompt: "test prompt"
    image_parts: ["front"]
`
		configFile, err := os.CreateTemp("", "config-*.yml")
		if err != nil {
			t.Fatalf("Failed to create temp config file: %v", err)
		}
		defer os.Remove(configFile.Name())
		if _, err := configFile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Failed to write to temp config file: %v", err)
		}
		configFile.Close()

		if os.Getenv("GEMINI_API_KEY") == "" {
			t.Skip("GEMINI_API_KEY not set, skipping integration-like test")
		}
		client, err := NewClientWithConfigPath(context.Background(), os.Getenv("GEMINI_API_KEY"), configFile.Name())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if client == nil {
			t.Error("expected client to be non-nil")
		} else {
			client.Close()
		}
	})
}

func TestPreprocessImage(t *testing.T) {
	// Create a simple 10x10 black PNG image for testing
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	for i := range img.Pix {
		img.Pix[i] = 0 // Black
	}
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}
	originalData := buf.Bytes()

	t.Run("should preprocess PNG image without error", func(t *testing.T) {
		processedData, err := PreprocessImage(originalData, "image/png")
		if err != nil {
			t.Errorf("unexpected error during preprocessing: %v", err)
		}

		// Check that the data has been modified
		if bytes.Equal(originalData, processedData) {
			t.Error("image data was not modified by preprocessing")
		}

		// Check that the output is still a valid image
		_, _, err = image.Decode(bytes.NewReader(processedData))
		if err != nil {
			t.Errorf("processed data is not a valid image: %v", err)
		}
	})

	t.Run("should handle non-image data gracefully", func(t *testing.T) {
		nonImageData := []byte("this is not an image")
		processedData, err := PreprocessImage(nonImageData, "text/plain")

		if err != nil {
			t.Errorf("expected nil error for non-image data, but got %v", err)
		}
		if !bytes.Equal(nonImageData, processedData) {
			t.Error("non-image data should be returned unmodified")
		}
	})
}

func TestExtractText(t *testing.T) {
	mockModel := &mockGenerativeModel{}
	config := &Config{
		Documents: map[string]Document{
			"test_doc": {
				Prompt: "Extract data from this document.",
				JSONStructure: map[string]string{
					"name": "name of the person",
					"age":  "age of the person",
				},
				ImageParts: []string{"front"},
			},
		},
	}
	// We manually create the client and inject the mock model.
	// The genaiClient can be nil because it's not used by ExtractText, only the generativeModel is.
	client := &Client{
		generativeModel: mockModel,
		config:          config,
	}
	mockFileParts := map[string]FilePart{
		"front": {Content: []byte("fake image data"), MimeType: "image/png"},
	}

	t.Run("should extract text successfully with supported mime type", func(t *testing.T) {
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			// Basic validation of the prompt structure
			if len(parts) != 3 { // Prompt, Label, Blob
				t.Errorf("expected 3 parts, got %d", len(parts))
			}
			blob, ok := parts[2].(genai.Blob)
			if !ok {
				t.Error("expected a genai.Blob part")
			}
			if blob.MIMEType != "image/png" {
				t.Errorf("expected mime type image/png, got %s", blob.MIMEType)
			}

			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []genai.Part{
								genai.Text("{\"name\":\"John Doe\",\"age\":\"30\"}"),
							},
						},
					},
				},
			}, nil
		}

		result, err := client.ExtractText(context.Background(), mockFileParts, "test_doc")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := "{\"name\":\"John Doe\",\"age\":\"30\"}"
		if result != expected {
			t.Errorf("expected result %q, but got %q", expected, result)
		}
	})

	t.Run("should extract text successfully with PDF mime type", func(t *testing.T) {
		pdfParts := map[string]FilePart{
			"front": {Content: []byte("fake pdf data"), MimeType: "application/pdf"},
		}
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			blob, ok := parts[2].(genai.Blob)
			if !ok {
				t.Error("expected a genai.Blob part")
			}
			if blob.MIMEType != "application/pdf" {
				t.Errorf("expected mime type application/pdf, got %s", blob.MIMEType)
			}
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []genai.Part{genai.Text("{}")}}}},
			}, nil
		}

		_, err := client.ExtractText(context.Background(), pdfParts, "test_doc")
		if err != nil {
			t.Errorf("unexpected error for PDF: %v", err)
		}
	})

	t.Run("should return error when mime type is not supported", func(t *testing.T) {
		unsupportedParts := map[string]FilePart{
			"front": {Content: []byte("fake data"), MimeType: "application/zip"},
		}
		_, err := client.ExtractText(context.Background(), unsupportedParts, "test_doc")
		if !errors.Is(err, ErrUnsupportedMimeType) {
			t.Errorf("expected error %v, but got %v", ErrUnsupportedMimeType, err)
		}
	})

	t.Run("should return error when doc type is not supported", func(t *testing.T) {
		_, err := client.ExtractText(context.Background(), mockFileParts, "unsupported_doc")
		if !errors.Is(err, ErrUnsupportedDocumentType) {
			t.Errorf("expected error %v, but got %v", ErrUnsupportedDocumentType, err)
		}
	})

	t.Run("should return error when api returns error", func(t *testing.T) {
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return nil, errors.New("api error")
		}

		_, err := client.ExtractText(context.Background(), mockFileParts, "test_doc")
		if err == nil {
			t.Error("expected error, but got nil")
		}
	})

	t.Run("should return error when api returns no candidates", func(t *testing.T) {
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{},
			}, nil
		}

		_, err := client.ExtractText(context.Background(), mockFileParts, "test_doc")
		if err == nil {
			t.Error("expected error, but got nil")
		}
	})

	t.Run("should return error when api returns unexpected format", func(t *testing.T) {
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []genai.Part{
								genai.ImageData("png", []byte("fake image data")),
							},
						},
					},
				},
			}, nil
		}

		_, err := client.ExtractText(context.Background(), mockFileParts, "test_doc")
		if err == nil {
			t.Error("expected error, but got nil")
		}
	})
}
