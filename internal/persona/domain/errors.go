package domain

import "errors"

var (
	ErrInvalidTraits         = errors.New("at least one trait is required")
	ErrGenerationFailed      = errors.New("failed to generate drag queen persona")
	ErrImageGenerationFailed = errors.New("failed to generate drag queen portrait")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded, please try again later")
	ErrCircuitBreakerOpen    = errors.New("service is temporarily unavailable due to upstream issues")
)
