package ocr

import (
	"context"
	"errors"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/y-mitsuyoshi/kensho/internal/configs"
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
		_, err := NewClient(context.Background(), "", &configs.Config{})
		if err == nil {
			t.Error("expected error, but got nil")
		}
	})
}

func TestExtractText(t *testing.T) {
	mockModel := &mockGenerativeModel{}
	config := &configs.Config{
		Documents: map[string]configs.Document{
			"test_doc": {
				Prompt: "Extract data from this document.",
				JSONStructure: map[string]string{
					"name": "name of the person",
					"age":  "age of the person",
				},
			},
		},
	}
	client := &Client{
		genaiClient: mockModel,
		config:      config,
	}

	t.Run("should extract text successfully", func(t *testing.T) {
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
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

		result, err := client.ExtractText(context.Background(), []byte("fake image data"), "image/png", "test_doc")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := "{\"name\":\"John Doe\",\"age\":\"30\"}"
		if result != expected {
			t.Errorf("expected result %q, but got %q", expected, result)
		}
	})

	t.Run("should return error when doc type is not supported", func(t *testing.T) {
		_, err := client.ExtractText(context.Background(), []byte("fake image data"), "image/png", "unsupported_doc")
		if !errors.Is(err, ErrUnsupportedDocumentType) {
			t.Errorf("expected error %v, but got %v", ErrUnsupportedDocumentType, err)
		}
	})

	t.Run("should return error when api returns error", func(t *testing.T) {
		mockModel.GenerateContentFunc = func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return nil, errors.New("api error")
		}

		_, err := client.ExtractText(context.Background(), []byte("fake image data"), "image/png", "test_doc")
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

		_, err := client.ExtractText(context.Background(), []byte("fake image data"), "image/png", "test_doc")
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
								genai.ImageData("image/png", []byte("fake image data")),
							},
						},
					},
				},
			}, nil
		}

		_, err := client.ExtractText(context.Background(), []byte("fake image data"), "image/png", "test_doc")
		if err == nil {
			t.Error("expected error, but got nil")
		}
	})
}
