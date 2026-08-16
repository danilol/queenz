package ingestion

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrJobNotFound is returned when a requested job does not exist in the database.
	ErrJobNotFound = errors.New("job not found")

	// ErrNoJobsAvailable is returned by ClaimNextJob when the queue is currently empty.
	ErrNoJobsAvailable = errors.New("no jobs available to claim")
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// Job represents a background task execution within the Postgres job queue.
type Job struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	Progress    string     `json:"progress"`
	ErrorMsg    *string    `json:"error_msg,omitempty"`
	Retries     int        `json:"retries"`
	MaxRetries  int        `json:"max_retries"`
	LockedUntil *time.Time `json:"locked_until,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// JobRepository defines the persistence operations for managing the task queue.
type JobRepository interface {
	// Create persists a new Job into the queue in a 'pending' state.
	Create(ctx context.Context, job *Job) error

	// GetByID retrieves a Job's current state and progress.
	GetByID(ctx context.Context, id string) (*Job, error)

	// Update updates an existing job's status, progress, and metadata.
	Update(ctx context.Context, job *Job) error

	// ClaimNextJob atomically finds and locks the next available job for execution
	// using the 'FOR UPDATE SKIP LOCKED' pattern. It sets the job status to 'running'
	// and extends the 'locked_until' lease by the provided lease duration.
	// Returns ErrNoJobsAvailable if there are no jobs to process.
	ClaimNextJob(ctx context.Context, leaseDuration time.Duration) (*Job, error)
}
