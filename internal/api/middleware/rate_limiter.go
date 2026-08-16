package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type clientVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter returns an Echo middleware that performs IP-based rate limiting.
func RateLimiter(r rate.Limit, b int) echo.MiddlewareFunc {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*clientVisitor)
	)

	// Background routine to cleanup old visitors
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			mu.Lock()
			v, exists := visitors[ip]
			if !exists {
				limiter := rate.NewLimiter(r, b)
				visitors[ip] = &clientVisitor{limiter: limiter, lastSeen: time.Now()}
				mu.Unlock()
				return next(c)
			}
			v.lastSeen = time.Now()
			if !v.limiter.Allow() {
				mu.Unlock()
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			mu.Unlock()

			return next(c)
		}
	}
}
