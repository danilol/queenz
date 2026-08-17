package handler

import (
	"context"
	"encoding/json"
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

// writeSSEEvent safely JSON-encodes payloads to escape special characters/newlines and writes SSE events.
func writeSSEEvent(w io.Writer, f http.Flusher, eventType, data string) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(encoded))
	if err != nil {
		return err
	}
	if f != nil {
		f.Flush()
	}
	return nil
}

// sseChunkWriter wraps an io.Writer and formats raw text chunks into SSE "chunk" events.
type sseChunkWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

func (sw *sseChunkWriter) Write(p []byte) (int, error) {
	err := writeSSEEvent(sw.w, sw.f, "chunk", string(p))
	if err != nil {
		return 0, err
	}
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
		_ = writeSSEEvent(w, flusher, "error", err.Error())
		return nil // End of stream
	}

	// 5. Send final event containing the generated Fal.ai high-glamour image URL
	_ = writeSSEEvent(w, flusher, "image", imageURL)

	return nil
}
