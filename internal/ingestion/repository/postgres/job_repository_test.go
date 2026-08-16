package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"queenx/internal/ingestion"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestJobRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewJobRepository(mock)
	ctx := context.Background()
	now := time.Now()
	var lockedUntil *time.Time
	var errorMsg *string

	job := &ingestion.Job{
		ID:          "job-1",
		Status:      ingestion.StatusPending,
		Progress:    "Starting...",
		ErrorMsg:    errorMsg,
		Retries:     0,
		MaxRetries:  3,
		LockedUntil: lockedUntil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO ingestion_jobs").
			WithArgs(job.ID, job.Status, job.Progress, job.ErrorMsg, job.Retries, job.MaxRetries, job.LockedUntil, job.CreatedAt, job.UpdatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(ctx, job)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO ingestion_jobs").
			WithArgs(job.ID, job.Status, job.Progress, job.ErrorMsg, job.Retries, job.MaxRetries, job.LockedUntil, job.CreatedAt, job.UpdatedAt).
			WillReturnError(errors.New("db error"))

		err := repo.Create(ctx, job)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestJobRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewJobRepository(mock)
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		columns := []string{"id", "status", "progress", "error_msg", "retries", "max_retries", "locked_until", "created_at", "updated_at"}
		mock.ExpectQuery("SELECT id, status, progress, error_msg, retries, max_retries, locked_until, created_at, updated_at FROM ingestion_jobs").
			WithArgs("job-1").
			WillReturnRows(pgxmock.NewRows(columns).AddRow("job-1", ingestion.StatusRunning, "Scraping...", nil, 1, 3, nil, now, now))

		job, err := repo.GetByID(ctx, "job-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job.ID != "job-1" || job.Status != ingestion.StatusRunning {
			t.Errorf("unexpected result: %+v", job)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, status, progress, error_msg, retries, max_retries, locked_until, created_at, updated_at FROM ingestion_jobs").
			WithArgs("job-1").
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetByID(ctx, "job-1")
		if !errors.Is(err, ingestion.ErrJobNotFound) {
			t.Errorf("expected ErrJobNotFound, got: %v", err)
		}
	})
}

func TestJobRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewJobRepository(mock)
	ctx := context.Background()
	now := time.Now()

	job := &ingestion.Job{
		ID:        "job-1",
		Status:    ingestion.StatusCompleted,
		Progress:  "Done",
		Retries:   0,
		UpdatedAt: now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("UPDATE ingestion_jobs").
			WithArgs(job.Status, job.Progress, job.ErrorMsg, job.Retries, job.LockedUntil, job.UpdatedAt, job.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(ctx, job)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("UPDATE ingestion_jobs").
			WithArgs(job.Status, job.Progress, job.ErrorMsg, job.Retries, job.LockedUntil, job.UpdatedAt, job.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(ctx, job)
		if !errors.Is(err, ingestion.ErrJobNotFound) {
			t.Errorf("expected ErrJobNotFound, got %v", err)
		}
	})
}

func TestJobRepository_ClaimNextJob(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewJobRepository(mock)
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		columns := []string{"id", "status", "progress", "error_msg", "retries", "max_retries", "locked_until", "created_at", "updated_at"}
		mock.ExpectQuery("UPDATE ingestion_jobs SET").
			WithArgs(ingestion.StatusRunning, "60.000000 seconds", ingestion.StatusPending, ingestion.StatusFailed).
			WillReturnRows(pgxmock.NewRows(columns).AddRow("job-1", ingestion.StatusRunning, "Starting...", nil, 0, 3, &now, now, now))

		job, err := repo.ClaimNextJob(ctx, 60*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job.ID != "job-1" || job.Status != ingestion.StatusRunning {
			t.Errorf("unexpected result: %+v", job)
		}
	})

	t.Run("no jobs available", func(t *testing.T) {
		mock.ExpectQuery("UPDATE ingestion_jobs SET").
			WithArgs(ingestion.StatusRunning, "60.000000 seconds", ingestion.StatusPending, ingestion.StatusFailed).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.ClaimNextJob(ctx, 60*time.Second)
		if !errors.Is(err, ingestion.ErrNoJobsAvailable) {
			t.Errorf("expected ErrNoJobsAvailable, got: %v", err)
		}
	})
}
