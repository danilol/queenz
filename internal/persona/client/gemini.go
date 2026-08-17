package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	personadomain "queenx/internal/persona/domain"

	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	limiter    *rate.Limiter
	breaker    *gobreaker.CircuitBreaker
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	// Limit to 2 requests per second to avoid free-tier rate limits
	limiter := rate.NewLimiter(rate.Limit(2.0), 4)

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "GeminiAPI",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	return &GeminiClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		limiter:    limiter,
		breaker:    breaker,
	}
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiSystemInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiGenConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

type GeminiRequest struct {
	Contents          []GeminiContent          `json:"contents"`
	SystemInstruction *GeminiSystemInstruction `json:"systemInstruction,omitempty"`
	GenerationConfig  *GeminiGenConfig         `json:"generationConfig,omitempty"`
}

type GeminiResponsePart struct {
	Text string `json:"text"`
}

type GeminiResponseContent struct {
	Parts []GeminiResponsePart `json:"parts"`
}

type GeminiResponseCandidate struct {
	Content GeminiResponseContent `json:"content"`
}

type GeminiStreamResponse struct {
	Candidates []GeminiResponseCandidate `json:"candidates"`
}

func extractTextFromSSELine(line string) (string, error) {
	if !strings.HasPrefix(line, "data:") {
		return "", nil
	}
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if data == "" {
		return "", nil
	}

	var res GeminiStreamResponse
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		return "", err
	}

	var textBuilder strings.Builder
	for _, c := range res.Candidates {
		for _, p := range c.Content.Parts {
			textBuilder.WriteString(p.Text)
		}
	}
	return textBuilder.String(), nil
}

// StreamGenerate calls Gemini stream API and writes raw text chunks to the writer.
// It returns the fully accumulated text generated.
func (g *GeminiClient) StreamGenerate(ctx context.Context, systemPrompt, prompt string, writer io.Writer) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("gemini API key is not configured")
	}

	// 1. Rate Limiting check
	if err := g.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("gemini rate limit wait: %w", personadomain.ErrRateLimitExceeded)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse", g.model)

	reqPayload := GeminiRequest{
		Contents: []GeminiContent{
			{
				Role:  "user",
				Parts: []GeminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: &GeminiGenConfig{
			ResponseMimeType: "application/json",
		},
	}
	if systemPrompt != "" {
		reqPayload.SystemInstruction = &GeminiSystemInstruction{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gemini payload: %w", err)
	}

	// 2. Execute POST handshake within Circuit Breaker
	respVal, err := g.breaker.Execute(func() (any, error) {
		req, reqErr := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", g.apiKey)

		resp, respErr := g.httpClient.Do(req)
		if respErr != nil {
			return nil, respErr
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("gemini returned unexpected status code: %d", resp.StatusCode)
		}
		return resp, nil
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			return "", personadomain.ErrCircuitBreakerOpen
		}
		return "", fmt.Errorf("gemini request failed: %w", err)
	}

	resp := respVal.(*http.Response)
	defer func() { _ = resp.Body.Close() }()

	// 3. Stream reader loop (outside of the circuit breaker handshake block)
	reader := bufio.NewReader(resp.Body)
	var accumulatedText strings.Builder

	for {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return accumulatedText.String(), err
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return accumulatedText.String(), fmt.Errorf("error reading gemini stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		textChunk, err := extractTextFromSSELine(line)
		if err != nil {
			// Skip malformed SSE lines or log them, but don't crash
			continue
		}

		if textChunk != "" {
			accumulatedText.WriteString(textChunk)
			_, _ = writer.Write([]byte(textChunk))
		}
	}

	return accumulatedText.String(), nil
}
