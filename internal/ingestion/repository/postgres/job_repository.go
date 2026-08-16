package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"queenx/internal/ingestion"

	"github.com/jackc/pgx/v5"
)

type jobRepository struct {
	db DB
}

// NewJobRepository creates a new PostgreSQL implementation of ingestion.JobRepository.
func NewJobRepository(db DB) ingestion.JobRepository {
	return &jobRepository{db: db}
}

func (r *jobRepository) Create(ctx context.Context, job *ingestion.Job) error {
	query := `
		INSERT INTO ingestion_jobs (id, status, progress, error_msg, retries, max_retries, locked_until, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query, job.ID, job.Status, job.Progress, job.ErrorMsg, job.Retries, job.MaxRetries, job.LockedUntil, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return fmt.Errorf("creating job: %w", err)
	}
	return nil
}

func (r *jobRepository) GetByID(ctx context.Context, id string) (*ingestion.Job, error) {
	query := `
		SELECT id, status, progress, error_msg, retries, max_retries, locked_until, created_at, updated_at
		FROM ingestion_jobs
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	var j ingestion.Job
	err := row.Scan(&j.ID, &j.Status, &j.Progress, &j.ErrorMsg, &j.Retries, &j.MaxRetries, &j.LockedUntil, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ingestion.ErrJobNotFound
		}
		return nil, fmt.Errorf("getting job by id: %w", err)
	}
	return &j, nil
}

func (r *jobRepository) Update(ctx context.Context, job *ingestion.Job) error {
	query := `
		UPDATE ingestion_jobs
		SET status = $1, progress = $2, error_msg = $3, retries = $4, locked_until = $5, updated_at = $6
		WHERE id = $7
	`
	cmdTag, err := r.db.Exec(ctx, query, job.Status, job.Progress, job.ErrorMsg, job.Retries, job.LockedUntil, job.UpdatedAt, job.ID)
	if err != nil {
		return fmt.Errorf("updating job: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ingestion.ErrJobNotFound
	}
	return nil
}

func (r *jobRepository) ClaimNextJob(ctx context.Context, leaseDuration time.Duration) (*ingestion.Job, error) {
	query := `
		UPDATE ingestion_jobs
		SET status = $1,
		    locked_until = NOW() + $2::INTERVAL,
		    updated_at = NOW()
		WHERE id = (
			SELECT id
			FROM ingestion_jobs
			WHERE (status = $3)
			   OR (status = $1 AND locked_until < NOW())
			   OR (status = $4 AND retries < max_retries AND locked_until < NOW())
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, status, progress, error_msg, retries, max_retries, locked_until, created_at, updated_at;
	`
	interval := fmt.Sprintf("%f seconds", leaseDuration.Seconds())
	row := r.db.QueryRow(ctx, query, ingestion.StatusRunning, interval, ingestion.StatusPending, ingestion.StatusFailed)

	var j ingestion.Job
	err := row.Scan(&j.ID, &j.Status, &j.Progress, &j.ErrorMsg, &j.Retries, &j.MaxRetries, &j.LockedUntil, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ingestion.ErrNoJobsAvailable
		}
		return nil, fmt.Errorf("claiming next job: %w", err)
	}
	return &j, nil
}
