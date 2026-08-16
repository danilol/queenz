package handler

import (
	"fmt"
	"net/http"
	"time"

	"queenx/internal/ingestion"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// JobHandler handles HTTP requests related to ingestion jobs.
type JobHandler struct {
	repo ingestion.JobRepository
}

// NewJobHandler creates a new JobHandler.
func NewJobHandler(repo ingestion.JobRepository) *JobHandler {
	return &JobHandler{repo: repo}
}

// CreateJob enqueues a new background ingestion job and returns its ID.
func (h *JobHandler) CreateJob(c echo.Context) error {
	jobID := uuid.New().String()
	now := time.Now()

	job := &ingestion.Job{
		ID:         jobID,
		Status:     ingestion.StatusPending,
		Progress:   "Job created, waiting for worker...",
		Retries:    0,
		MaxRetries: 3,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := h.repo.Create(c.Request().Context(), job); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create job")
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"id": jobID,
	})
}

// GetJob returns the current status and progress of a specific job.
func (h *JobHandler) GetJob(c echo.Context) error {
	jobID := c.Param("id")
	job, err := h.repo.GetByID(c.Request().Context(), jobID)
	if err != nil {
		if errorsIs(err, ingestion.ErrJobNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "job not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get job")
	}

	return c.JSON(http.StatusOK, job)
}

// GetJobProgressSSE streams the progress of a job via Server-Sent Events.
func (h *JobHandler) GetJobProgressSSE(c echo.Context) error {
	w := c.Response().Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	jobID := c.Param("id")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Initial fetch to make sure the job exists before streaming
	_, err := h.repo.GetByID(c.Request().Context(), jobID)
	if err != nil {
		if errorsIs(err, ingestion.ErrJobNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "job not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get job")
	}

	for {
		select {
		case <-c.Request().Context().Done():
			// Client disconnected
			return nil
		case <-ticker.C:
			job, err := h.repo.GetByID(c.Request().Context(), jobID)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(w, "data: {\"status\":\"%s\",\"progress\":\"%s\"}\n\n", job.Status, job.Progress)
			c.Response().Flush()

			if job.Status == ingestion.StatusCompleted || job.Status == ingestion.StatusFailed {
				return nil
			}
		}
	}
}

// Helper to avoid directly importing "errors" if not strictly needed
func errorsIs(err, target error) bool {
	return err != nil && err.Error() == target.Error()
}
