package ingestion

import (
	"context"
	"time"
)

// JobStatus represents the state of a background job.
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

// Job represents a background task execution within the Postgres job queue.
type Job struct {
	ID          string     `json:"id"`
	Status      JobStatus  `json:"status"`
	Progress    string     `json:"progress"`
	ErrorMsg    *string    `json:"errorMsg,omitempty"`
	Retries     int        `json:"retries"`
	MaxRetries  int        `json:"maxRetries"`
	LockedUntil *time.Time `json:"lockedUntil,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
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
