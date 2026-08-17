package handler_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"queenx/internal/api/handler"
	personadomain "queenx/internal/persona/domain"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPersonaGenerator mocks the PersonaGenerator interface.
type MockPersonaGenerator struct {
	mock.Mock
}

func (m *MockPersonaGenerator) GeneratePersona(ctx context.Context, req *personadomain.GeneratePersonaRequest, sseWriter io.Writer) (*personadomain.PersonaResult, string, error) {
	args := m.Called(ctx, req, sseWriter)
	if sseWriter != nil && args.String(2) != "" {
		_, _ = sseWriter.Write([]byte(args.String(2)))
	}
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(3)
	}
	return args.Get(0).(*personadomain.PersonaResult), args.String(1), args.Error(3)
}

func TestPersonaHandler_GeneratePersona_Success(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/persona/generate?traits=camp,comedy", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockGen := new(MockPersonaGenerator)
	result := &personadomain.PersonaResult{
		DragName: "Lulu Lemon",
	}
	mockGen.On("GeneratePersona", mock.Anything, &personadomain.GeneratePersonaRequest{Traits: []string{"camp", "comedy"}}, mock.Anything).
		Return(result, "https://fal.media/lulu.png", "lulu-chunk", nil)

	h := handler.NewPersonaHandler(mockGen)
	err := h.GeneratePersona(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	body := rec.Body.String()
	assert.Contains(t, body, "event: chunk\ndata: \"lulu-chunk\"\n\n")
	assert.Contains(t, body, "event: image\ndata: \"https://fal.media/lulu.png\"\n\n")

	mockGen.AssertExpectations(t)
}

func TestPersonaHandler_GeneratePersona_MissingTraits(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/persona/generate", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := handler.NewPersonaHandler(nil)
	err := h.GeneratePersona(c)

	assert.Error(t, err)
	echoErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, echoErr.Code)
	assert.Contains(t, echoErr.Message, "traits parameter is required")
}

func TestPersonaHandler_GeneratePersona_ServiceError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/persona/generate?traits=fashion", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockGen := new(MockPersonaGenerator)
	mockGen.On("GeneratePersona", mock.Anything, &personadomain.GeneratePersonaRequest{Traits: []string{"fashion"}}, mock.Anything).
		Return(nil, "", "", errors.New("upstream failure"))

	h := handler.NewPersonaHandler(mockGen)
	err := h.GeneratePersona(c)

	assert.NoError(t, err)
	body := rec.Body.String()
	assert.Contains(t, body, "event: error\ndata: \"upstream failure\"\n\n")
}
