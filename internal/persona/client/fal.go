package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	personadomain "queenx/internal/persona/domain"

	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

type FalClient struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
	limiter    *rate.Limiter
	breaker    *gobreaker.CircuitBreaker
}

func NewFalClient(apiKey, apiURL string) *FalClient {
	if apiURL == "" {
		apiURL = "https://queue.fal.run/fal-ai/flux/schnell"
	}
	// Limit to 2 requests per second to avoid rate limits
	limiter := rate.NewLimiter(rate.Limit(2.0), 4)

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "FalAI_API",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	return &FalClient{
		apiKey:     apiKey,
		apiURL:     apiURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    limiter,
		breaker:    breaker,
	}
}

type FalRequest struct {
	Prompt    string `json:"prompt"`
	ImageSize string `json:"image_size,omitempty"`
}

type FalImage struct {
	URL string `json:"url"`
}

type FalResponse struct {
	Images []FalImage `json:"images"`
}

type falBreakerResult struct {
	statusCode int
	body       []byte
}

type falError struct {
	result *falBreakerResult
}

func (e *falError) Error() string {
	return fmt.Sprintf("fal.ai returned retryable error status: %d", e.result.statusCode)
}

// GenerateImage sends a request to the Fal.ai API with retry logic and circuit breaking.
func (f *FalClient) GenerateImage(ctx context.Context, prompt string) (string, error) {
	if f.apiKey == "" {
		return "", fmt.Errorf("fal.ai API key is not configured")
	}

	payload := FalRequest{
		Prompt:    prompt,
		ImageSize: "square_hd",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fal.ai payload: %w", err)
	}

	const maxRetries = 3
	baseDelay := 500 * time.Millisecond
	maxDelay := 3 * time.Second

	var imageURL string

	// 2. Execute POST with Retries & Circuit Breaking
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Respect context cancellation before each attempt
		if err := ctx.Err(); err != nil {
			return "", err
		}

		// 1. Rate Limiting check inside the attempt immediately before breaker.Execute
		if err := f.limiter.Wait(ctx); err != nil {
			return "", fmt.Errorf("fal.ai rate limit wait: %w", personadomain.ErrRateLimitExceeded)
		}

		respVal, err := f.breaker.Execute(func() (any, error) {
			req, err := http.NewRequestWithContext(ctx, "POST", f.apiURL, bytes.NewReader(payloadBytes))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Key "+f.apiKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := f.httpClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer func() { _ = resp.Body.Close() }()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			// If it's a 5xx retryable status, return typed error to trigger failure tracking inside the circuit breaker
			result := &falBreakerResult{statusCode: resp.StatusCode, body: bodyBytes}
			if resp.StatusCode >= 500 {
				return nil, &falError{result: result}
			}

			return result, nil
		})

		if err != nil {
			if errors.Is(err, gobreaker.ErrOpenState) {
				return "", personadomain.ErrCircuitBreakerOpen
			}

			var falErr *falError
			if errors.As(err, &falErr) {
				// Retryable 5xx error carrying response body
				timer := time.NewTimer(f.calculateDelay(attempt, baseDelay, maxDelay))
				select {
				case <-ctx.Done():
					timer.Stop()
					return "", ctx.Err()
				case <-timer.C:
				}
				continue
			}

			// Connection or read issue
			timer := time.NewTimer(f.calculateDelay(attempt, baseDelay, maxDelay))
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", ctx.Err()
			case <-timer.C:
			}
			continue
		}

		result := respVal.(*falBreakerResult)

		if result.statusCode != http.StatusOK {
			// This is a non-5xx error (e.g. 4xx). We fail immediately without retry!
			return "", fmt.Errorf("fal.ai returned error status: %d", result.statusCode)
		}

		var falResp FalResponse
		if err := json.Unmarshal(result.body, &falResp); err != nil {
			return "", fmt.Errorf("failed to decode fal.ai response: %w", err)
		}

		if len(falResp.Images) == 0 || falResp.Images[0].URL == "" {
			return "", fmt.Errorf("fal.ai response did not contain image URL")
		}

		imageURL = falResp.Images[0].URL
		break
	}

	if imageURL == "" {
		return "", personadomain.ErrImageGenerationFailed
	}

	return imageURL, nil
}

func (f *FalClient) calculateDelay(attempt int, base, maxDelay time.Duration) time.Duration {
	delay := base * time.Duration(1<<uint(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}
	// Add jitter (up to 20% of delay)
	jitter := time.Duration(rand.Int63n(int64(delay / 5))) //nolint:gosec // Not cryptographically critical
	return delay + jitter
}
