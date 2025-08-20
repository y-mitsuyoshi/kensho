package kensho

import (
	"context"
	"errors"
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
	// Create a dummy config file for testing
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

	t.Run("should return error when api key is not set", func(t *testing.T) {
		_, err := NewClient(context.Background(), "", configFile.Name())
		if err == nil {
			t.Error("expected error, but got nil")
		}
	})

	t.Run("should create new client successfully", func(t *testing.T) {
		_, err := NewClient(context.Background(), "fake-api-key", configFile.Name())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
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
	client := &Client{
		genaiClient: mockModel,
		config:      config,
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
