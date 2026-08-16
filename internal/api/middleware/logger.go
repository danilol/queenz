package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// Logger returns an Echo middleware that logs HTTP requests using the standard slog package.
func Logger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()
			latency := time.Since(start)

			logger.Info("http request",
				slog.String("method", req.Method),
				slog.String("uri", req.RequestURI),
				slog.Int("status", res.Status),
				slog.String("ip", c.RealIP()),
				slog.Duration("latency", latency),
			)

			return nil
		}
	}
}
