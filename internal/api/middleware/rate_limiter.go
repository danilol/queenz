package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

const (
	cleanupInterval = 1 * time.Minute
	visitorExpire   = 3 * time.Minute
)

type clientVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter returns an Echo middleware that performs IP-based rate limiting.
func RateLimiter(ctx context.Context, r rate.Limit, b int) echo.MiddlewareFunc {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*clientVisitor)
	)

	// Background routine to cleanup old visitors
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				for ip, v := range visitors {
					if time.Since(v.lastSeen) > visitorExpire {
						delete(visitors, ip)
					}
				}
				mu.Unlock()
			}
		}
	}()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			mu.Lock()
			v, exists := visitors[ip]
			if !exists {
				limiter := rate.NewLimiter(r, b)
				v = &clientVisitor{limiter: limiter, lastSeen: time.Now()}
				visitors[ip] = v
			} else {
				v.lastSeen = time.Now()
			}

			if !v.limiter.Allow() {
				mu.Unlock()
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			mu.Unlock()

			return next(c)
		}
	}
}
