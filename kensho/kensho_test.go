package kensho

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"reflect"
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

func TestExtract(t *testing.T) {
	mockModel := &mockGenerativeModel{}
	config := &Config{
		Documents: map[string]Document{
			"test_doc": {
				Prompt: "Extract data from this document.",
				JSONStructure: map[string]string{
					"name":        "name of the person",
					"age":         "age of the person",
					"card_number": "card number",
				},
				ImageParts: []string{"front"},
			},
		},
	}
	client := &Client{
		generativeModel: mockModel,
		config:          config,
	}
	mockFileParts := map[string]FilePart{
		"front": {Content: []byte("fake image data"), MimeType: "image/png"},
	}

	t.Run("should extract data successfully", func(t *testing.T) {
		mockResponse := `{"name":{"value":"John Doe","confidence_score":0.9},"age":{"value":30,"confidence_score":0.95}}`
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []genai.Part{genai.Text(mockResponse)},
						},
					},
				},
			}, nil
		}

		result, err := client.Extract(context.Background(), mockFileParts, "test_doc", false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expectedData := map[string]Field{
			"name": {Value: "John Doe", ConfidenceScore: 0.9},
			"age":  {Value: float64(30), ConfidenceScore: 0.95},
		}
		if !reflect.DeepEqual(result.ExtractedData, expectedData) {
			t.Errorf("expected result %v, but got %v", expectedData, result.ExtractedData)
		}
	})

	t.Run("should return error for invalid JSON response", func(t *testing.T) {
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []genai.Part{genai.Text("not a valid json")},
						},
					},
				},
			}, nil
		}

		_, err := client.Extract(context.Background(), mockFileParts, "test_doc", false)
		if err == nil {
			t.Error("expected an error for invalid JSON, but got nil")
		}
	})

	t.Run("should mask card number when masking is true", func(t *testing.T) {
		mockResponse := `{"card_number":{"value":"123456789012","confidence_score":0.99}}`
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []genai.Part{genai.Text(mockResponse)}}}},
			}, nil
		}

		result, err := client.Extract(context.Background(), mockFileParts, "test_doc", true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		maskedValue := result.ExtractedData["card_number"].Value
		expectedMaskedValue := "************9012"
		if maskedValue != expectedMaskedValue {
			t.Errorf("expected card number to be masked as %s, but got %s", expectedMaskedValue, maskedValue)
		}
	})

	t.Run("should parse forgery warning correctly", func(t *testing.T) {
		mockResponse := `{"name":{"value":"John Doe","confidence_score":0.9}, "forgery_warning": {"has_signs_of_forgery": true, "reason": "font mismatch"}}`
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []genai.Part{genai.Text(mockResponse)}}}},
			}, nil
		}

		result, err := client.Extract(context.Background(), mockFileParts, "test_doc", false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result.ForgeryWarning == nil {
			t.Fatal("expected forgery warning, but got nil")
		}
		if !result.ForgeryWarning.HasSignsOfForgery {
			t.Error("expected HasSignsOfForgery to be true")
		}
		if result.ForgeryWarning.Reason != "font mismatch" {
			t.Errorf("expected forgery reason 'font mismatch', but got '%s'", result.ForgeryWarning.Reason)
		}
		if _, ok := result.ExtractedData["forgery_warning"]; ok {
			t.Error("forgery_warning should not be in extracted data")
		}
	})

	t.Run("should extract data successfully with PDF mime type", func(t *testing.T) {
		pdfParts := map[string]FilePart{
			"front": {Content: []byte("fake pdf data"), MimeType: "application/pdf"},
		}
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []genai.Part{genai.Text("{}")}}}},
			}, nil
		}

		_, err := client.Extract(context.Background(), pdfParts, "test_doc", false)
		if err != nil {
			t.Errorf("unexpected error for PDF: %v", err)
		}
	})

	t.Run("should return error when mime type is not supported", func(t *testing.T) {
		unsupportedParts := map[string]FilePart{
			"front": {Content: []byte("fake data"), MimeType: "application/zip"},
		}
		_, err := client.Extract(context.Background(), unsupportedParts, "test_doc", false)
		if !errors.Is(err, ErrUnsupportedMimeType) {
			t.Errorf("expected error %v, but got %v", ErrUnsupportedMimeType, err)
		}
	})

	t.Run("should return error when doc type is not supported", func(t *testing.T) {
		_, err := client.Extract(context.Background(), mockFileParts, "unsupported_doc", false)
		if !errors.Is(err, ErrUnsupportedDocumentType) {
			t.Errorf("expected error %v, but got %v", ErrUnsupportedDocumentType, err)
		}
	})

	t.Run("should return error when api returns error", func(t *testing.T) {
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return nil, errors.New("api error")
		}

		_, err := client.Extract(context.Background(), mockFileParts, "test_doc", false)
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

		_, err := client.Extract(context.Background(), mockFileParts, "test_doc", false)
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

		_, err := client.Extract(context.Background(), mockFileParts, "test_doc", false)
		if err == nil {
			t.Error("expected error, but got nil")
		}
	})
}
