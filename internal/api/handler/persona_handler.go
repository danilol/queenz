package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	personadomain "queenx/internal/persona/domain"

	"github.com/labstack/echo/v4"
)

type PersonaGenerator interface {
	GeneratePersona(ctx context.Context, req *personadomain.GeneratePersonaRequest, sseWriter io.Writer) (*personadomain.PersonaResult, string, error)
}

type PersonaHandler struct {
	service PersonaGenerator
}

func NewPersonaHandler(service PersonaGenerator) *PersonaHandler {
	return &PersonaHandler{service: service}
}

// sseChunkWriter wraps an io.Writer and formats raw text chunks into SSE "chunk" events.
type sseChunkWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

func (sw *sseChunkWriter) Write(p []byte) (int, error) {
	_, err := fmt.Fprintf(sw.w, "event: chunk\ndata: %s\n\n", string(p))
	if err != nil {
		return 0, err
	}
	sw.f.Flush()
	return len(p), nil
}

// GeneratePersona handles the GET /api/v1/persona/generate SSE endpoint.
func (h *PersonaHandler) GeneratePersona(c echo.Context) error {
	w := c.Response().Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "Streaming unsupported by container")
	}

	// 1. Extract traits from query parameter: e.g. ?traits=camp,comedy
	traitsParam := c.QueryParam("traits")
	if traitsParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "traits parameter is required")
	}

	traits := strings.Split(traitsParam, ",")
	var cleanedTraits []string
	for _, t := range traits {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			cleanedTraits = append(cleanedTraits, trimmed)
		}
	}

	if len(cleanedTraits) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "at least one non-empty trait is required")
	}

	// 2. Set SSE response headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Prevent Nginx buffering
	c.Response().WriteHeader(http.StatusOK)

	// Flush headers to client
	flusher.Flush()

	req := &personadomain.GeneratePersonaRequest{Traits: cleanedTraits}
	ctx := c.Request().Context()

	// 3. Create sseChunkWriter to pipe Gemini chunks to SSE client in real-time
	chunkWriter := &sseChunkWriter{w: w, f: flusher}

	// 4. Invoke the service
	_, imageURL, err := h.service.GeneratePersona(ctx, req, chunkWriter)
	if err != nil {
		// Send error event if we failed
		_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return nil // End of stream
	}

	// 5. Send final event containing the generated Fal.ai high-glamour image URL
	_, _ = fmt.Fprintf(w, "event: image\ndata: %s\n\n", imageURL)
	flusher.Flush()

	return nil
}
