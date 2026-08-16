package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"queenx/internal/api"
	"queenx/internal/api/handler"
	"queenx/internal/core/domain"
	core_pg "queenx/internal/core/repository/postgres"
	"queenx/internal/ingestion"
	ingestion_pg "queenx/internal/ingestion/repository/postgres"
	lineage_neo4j "queenx/internal/lineage/repository/neo4j"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server startup failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- 1. PostgreSQL Connection ---
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres_password@localhost:5432/queenx_dev?sslmode=disable" //nolint:gosec // Local development credentials
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("unable to connect to postgres: %w", err)
	}
	defer pool.Close()

	if pingErr := pool.Ping(ctx); pingErr != nil {
		return fmt.Errorf("postgres ping failed: %w", pingErr)
	}
	logger.Info("Connected to PostgreSQL")

	// Ensure Schema
	if schemaErr := core_pg.EnsureSchema(ctx, pool); schemaErr != nil {
		return fmt.Errorf("failed to bootstrap postgres schema: %w", schemaErr)
	}

	// --- 2. Neo4j Connection ---
	neo4jURL := os.Getenv("NEO4J_URI")
	if neo4jURL == "" {
		neo4jURL = "bolt://localhost:7687"
	}
	neo4jUser := os.Getenv("NEO4J_USER")
	if neo4jUser == "" {
		neo4jUser = "neo4j"
	}
	neo4jPass := os.Getenv("NEO4J_PASSWORD")
	if neo4jPass == "" {
		neo4jPass = "neo4j_password"
	}
	driver, err := neo4j.NewDriverWithContext(neo4jURL, neo4j.BasicAuth(neo4jUser, neo4jPass, ""))
	if err != nil {
		return fmt.Errorf("unable to connect to neo4j: %w", err)
	}
	defer func() { _ = driver.Close(ctx) }()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		// Log warning instead of failing completely, since Neo4j might be optional for some early testing
		logger.Warn("neo4j connectivity check failed", slog.Any("error", err))
	} else {
		logger.Info("Connected to Neo4j")
	}

	// --- 3. Repositories Initialization ---
	franchiseRepo := core_pg.NewFranchiseRepository(pool)
	seasonRepo := core_pg.NewSeasonRepository(pool)
	lineageRepo := lineage_neo4j.NewRepository(driver)
	jobRepo := ingestion_pg.NewJobRepository(pool)

	// Ensure constraints on Neo4j
	if err := lineageRepo.EnsureConstraints(ctx); err != nil {
		logger.Warn("failed to ensure neo4j constraints", slog.Any("error", err))
	}

	// --- 4. Background Ingestion Worker ---
	scraper := ingestion.NewScraper("https://rupaulsdragrace.fandom.com")

	executeJob := func(jobCtx context.Context, job *ingestion.Job, progress chan<- string) error {
		progress <- "Connecting to Wiki..."

		franchises, err := scraper.ScrapeFranchises(jobCtx)
		if err != nil {
			return fmt.Errorf("scraping franchises: %w", err)
		}
		progress <- fmt.Sprintf("Scraped %d franchises. Synchronizing...", len(franchises))

		for i, f := range franchises {
			progress <- fmt.Sprintf("Saving franchise %d/%d: %s", i+1, len(franchises), f.Name)
			domainF := &domain.Franchise{
				ID:        uuid.New().String(), // Generate a new ID or extract from URL if possible
				Name:      f.Name,
				Country:   f.Country,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err := franchiseRepo.Create(jobCtx, domainF)
			if err != nil && !errors.Is(err, domain.ErrAlreadyExists) {
				return fmt.Errorf("saving franchise %s: %w", f.Name, err)
			}
		}

		progress <- "Done."
		return nil
	}

	worker := ingestion.NewWorker(jobRepo, executeJob)
	workerDone := make(chan struct{})
	go func() {
		logger.Info("Starting background ingestion worker")
		worker.Start(ctx)
		logger.Info("Background ingestion worker stopped")
		close(workerDone)
	}()

	// --- 5. HTTP Handlers & Server Setup ---
	franchiseH := handler.NewFranchiseHandler(franchiseRepo, seasonRepo)
	lineageH := handler.NewLineageHandler(lineageRepo)
	jobH := handler.NewJobHandler(jobRepo)

	server := api.NewServer(ctx, franchiseH, lineageH, jobH, logger)

	// --- 6. Graceful Shutdown ---
	serverErrChan := make(chan error, 1)
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		if err := server.Start(ctx, ":"+port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrChan <- fmt.Errorf("server start failed: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrChan:
		return err
	case <-quit:
		logger.Info("Received shutdown signal")
	}

	cancel() // Cancel context for background worker and server

	const shutdownTimeout = 10 * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Wait for worker to finish or timeout
	select {
	case <-workerDone:
		logger.Info("Worker shut down successfully")
	case <-shutdownCtx.Done():
		logger.Warn("Worker shutdown timed out")
	}

	logger.Info("Server exited gracefully")
	return nil
}
