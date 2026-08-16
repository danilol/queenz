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

	s.setupMiddlewares()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddlewares() {
	s.echo.Use(echoMiddleware.Recover())
	s.echo.Use(middleware.Logger(s.logger))
	s.echo.Use(middleware.RateLimiter(rate.Limit(5), 10)) // 5 req/s burst 10
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
func (s *Server) Start(address string) error {
	s.logger.Info("Starting API server...", slog.String("address", address))
	return s.echo.Start(address)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down API server gracefully...")
	return s.echo.Shutdown(ctx)
}
