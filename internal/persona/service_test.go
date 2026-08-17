package persona_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	lineagedomain "queenx/internal/lineage/domain"
	"queenx/internal/persona"
	personadomain "queenx/internal/persona/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLineageReader mocks the LineageReader interface.
type MockLineageReader struct {
	mock.Mock
}

func (m *MockLineageReader) FindQueensByClassifications(ctx context.Context, classifications []string, limit int) ([]*lineagedomain.Queen, error) {
	args := m.Called(ctx, classifications, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*lineagedomain.Queen), args.Error(1)
}

// MockGeminiClient mocks the GeminiClient interface.
type MockGeminiClient struct {
	mock.Mock
}

func (m *MockGeminiClient) StreamGenerate(ctx context.Context, systemPrompt, prompt string, writer io.Writer) (string, error) {
	args := m.Called(ctx, systemPrompt, prompt, writer)
	if writer != nil && args.String(1) != "" {
		_, _ = writer.Write([]byte(args.String(1)))
	}
	return args.String(0), args.Error(2)
}

// MockFalClient mocks the FalClient interface.
type MockFalClient struct {
	mock.Mock
}

func (m *MockFalClient) GenerateImage(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func TestPersonaService_GeneratePersona_Success(t *testing.T) {
	ctx := context.Background()
	mockLineage := new(MockLineageReader)
	mockGemini := new(MockGeminiClient)
	mockFal := new(MockFalClient)

	// Setup similar queens
	contextQueens := []*lineagedomain.Queen{
		{
			ID:              "queen-1",
			DragName:        "Gigi Goode",
			Classifications: []string{"fashion", "comedy"},
		},
	}
	mockLineage.On("FindQueensByClassifications", ctx, []string{"fashion"}, 3).Return(contextQueens, nil)

	// Setup Gemini output (valid JSON matching schema)
	geminiOutput := `{"dragName": "Fashionista", "bio": "Iconic", "stats": {"glamour": 10, "comedy": 7, "dance": 8, "camp": 6}, "imageGenerationPrompt": "portrait of Fashionista"}`
	mockGemini.On("StreamGenerate", ctx, mock.Anything, mock.Anything, mock.Anything).Return(geminiOutput, geminiOutput, nil)

	// Setup Fal.ai output
	mockFal.On("GenerateImage", ctx, "portrait of Fashionista").Return("https://fal.media/image.png", nil)

	svc := persona.NewPersonaService(mockLineage, mockGemini, mockFal)

	var buf bytes.Buffer
	req := &personadomain.GeneratePersonaRequest{Traits: []string{"fashion"}}

	res, imageURL, err := svc.GeneratePersona(ctx, req, &buf)
	assert.NoError(t, err)
	assert.Equal(t, "https://fal.media/image.png", imageURL)
	assert.Equal(t, "Fashionista", res.DragName)
	assert.Equal(t, "Iconic", res.Bio)
	assert.Equal(t, 10, res.Stats.Glamour)
	assert.Equal(t, geminiOutput, buf.String()) // Verify stream got written to

	mockLineage.AssertExpectations(t)
	mockGemini.AssertExpectations(t)
	mockFal.AssertExpectations(t)
}

func TestPersonaService_GeneratePersona_InvalidTraits(t *testing.T) {
	svc := persona.NewPersonaService(nil, nil, nil)
	req := &personadomain.GeneratePersonaRequest{Traits: []string{}}
	_, _, err := svc.GeneratePersona(context.Background(), req, nil)
	assert.ErrorIs(t, err, personadomain.ErrInvalidTraits)
}

func TestPersonaService_GeneratePersona_LineageErrorFallback(t *testing.T) {
	ctx := context.Background()
	mockLineage := new(MockLineageReader)
	mockGemini := new(MockGeminiClient)
	mockFal := new(MockFalClient)

	// Lineage fails, but the service should still continue! (Resilient Fallback)
	mockLineage.On("FindQueensByClassifications", ctx, []string{"camp"}, 3).Return(nil, errors.New("neo4j down"))

	geminiOutput := `{"dragName": "Campy", "bio": "Hilarious", "stats": {"glamour": 6, "comedy": 10, "dance": 7, "camp": 10}, "imageGenerationPrompt": "portrait of Campy"}`
	mockGemini.On("StreamGenerate", ctx, mock.Anything, mock.Anything, mock.Anything).Return(geminiOutput, geminiOutput, nil)
	mockFal.On("GenerateImage", ctx, "portrait of Campy").Return("https://fal.media/image2.png", nil)

	svc := persona.NewPersonaService(mockLineage, mockGemini, mockFal)

	req := &personadomain.GeneratePersonaRequest{Traits: []string{"camp"}}
	res, imageURL, err := svc.GeneratePersona(ctx, req, io.Discard)

	assert.NoError(t, err)
	assert.Equal(t, "Campy", res.DragName)
	assert.Equal(t, "https://fal.media/image2.png", imageURL)

	mockLineage.AssertExpectations(t)
}

func TestPersonaService_GeneratePersona_ValidationFailures(t *testing.T) {
	tests := []struct {
		name         string
		geminiOutput string
	}{
		{
			name:         "empty object",
			geminiOutput: `{}`,
		},
		{
			name:         "null object",
			geminiOutput: `null`,
		},
		{
			name:         "missing imageGenerationPrompt",
			geminiOutput: `{"dragName": "Fashionista", "bio": "Iconic", "stats": {"glamour": 10, "comedy": 7, "dance": 8, "camp": 6}}`,
		},
		{
			name:         "out-of-range stat too high",
			geminiOutput: `{"dragName": "Fashionista", "bio": "Iconic", "stats": {"glamour": 11, "comedy": 7, "dance": 8, "camp": 6}, "imageGenerationPrompt": "portrait"}`,
		},
		{
			name:         "out-of-range stat too low",
			geminiOutput: `{"dragName": "Fashionista", "bio": "Iconic", "stats": {"glamour": 0, "comedy": 7, "dance": 8, "camp": 6}, "imageGenerationPrompt": "portrait"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockLineage := new(MockLineageReader)
			mockGemini := new(MockGeminiClient)
			mockFal := new(MockFalClient)

			mockLineage.On("FindQueensByClassifications", ctx, []string{"fashion"}, 3).Return(nil, nil)
			mockGemini.On("StreamGenerate", ctx, mock.Anything, mock.Anything, mock.Anything).Return(tt.geminiOutput, tt.geminiOutput, nil)

			svc := persona.NewPersonaService(mockLineage, mockGemini, mockFal)
			req := &personadomain.GeneratePersonaRequest{Traits: []string{"fashion"}}

			_, _, err := svc.GeneratePersona(ctx, req, io.Discard)
			assert.Error(t, err)

			mockFal.AssertNotCalled(t, "GenerateImage", mock.Anything)
		})
	}
}
