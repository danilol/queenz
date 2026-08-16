package api

import (
	"context"
	"log/slog"

	"queenx/internal/api/handler"
	"queenx/internal/api/middleware"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

const (
	rateLimitRPS   = 5
	rateLimitBurst = 10
)

// Server encapsulates the Echo framework and its dependencies.
type Server struct {
	echo       *echo.Echo
	franchiseH *handler.FranchiseHandler
	lineageH   *handler.LineageHandler
	jobH       *handler.JobHandler
	logger     *slog.Logger
}

// NewServer initializes a new Echo Server with its dependencies.
func NewServer(
	ctx context.Context,
	franchiseH *handler.FranchiseHandler,
	lineageH *handler.LineageHandler,
	jobH *handler.JobHandler,
	logger *slog.Logger,
) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	s := &Server{
		echo:       e,
		franchiseH: franchiseH,
		lineageH:   lineageH,
		jobH:       jobH,
		logger:     logger,
	}

	s.setupMiddlewares(ctx)
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddlewares(ctx context.Context) {
	s.echo.Use(echoMiddleware.Recover())
	s.echo.Use(middleware.Logger(s.logger))
	s.echo.Use(middleware.RateLimiter(ctx, rate.Limit(rateLimitRPS), rateLimitBurst))
}

func (s *Server) setupRoutes() {
	v1 := s.echo.Group("/api/v1")

	// Franchise routes
	v1.GET("/franchises", s.franchiseH.ListFranchises)
	v1.GET("/franchises/:id", s.franchiseH.GetFranchise)
	v1.GET("/franchises/:id/seasons", s.franchiseH.ListSeasons)

	// Lineage routes
	v1.GET("/queens/:id/lineage", s.lineageH.FindAestheticSiblings)

	// Job routes
	v1.POST("/jobs/ingest", s.jobH.CreateJob)
	v1.GET("/jobs/:id", s.jobH.GetJob)
	v1.GET("/jobs/:id/progress", s.jobH.GetJobProgressSSE)
}

// Start runs the HTTP server.
func (s *Server) Start(ctx context.Context, address string) error {
	s.logger.Info("Starting API server...", slog.String("address", address))

	// Start server in background to allow shutdown
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.echo.Start(address)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("Shutting down API server gracefully...")
		return s.echo.Shutdown(context.Background())
	}
}
