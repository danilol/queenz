package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockJobRepository implements JobRepository for testing.
type mockJobRepository struct {
	jobs            map[string]*Job
	claimError      error
	updateError     error
	claimCallCount  int
	updateCallCount int
}

func (m *mockJobRepository) Create(ctx context.Context, job *Job) error {
	m.jobs[job.ID] = job
	return nil
}

func (m *mockJobRepository) GetByID(ctx context.Context, id string) (*Job, error) {
	if job, ok := m.jobs[id]; ok {
		return job, nil
	}
	return nil, ErrJobNotFound
}

func (m *mockJobRepository) Update(ctx context.Context, job *Job) error {
	m.updateCallCount++
	if m.updateError != nil {
		return m.updateError
	}
	m.jobs[job.ID] = job
	return nil
}

func (m *mockJobRepository) ClaimNextJob(ctx context.Context, leaseDuration time.Duration) (*Job, error) {
	m.claimCallCount++
	if m.claimError != nil {
		return nil, m.claimError
	}
	for _, job := range m.jobs {
		if job.Status == StatusPending {
			job.Status = StatusRunning
			return job, nil
		}
	}
	return nil, ErrNoJobsAvailable
}

func TestWorker_Success(t *testing.T) {
	repo := &mockJobRepository{
		jobs: map[string]*Job{
			"job-1": {
				ID:         "job-1",
				Status:     StatusPending,
				MaxRetries: 3,
			},
		},
	}

	execFn := func(ctx context.Context, job *Job, progress chan<- string) error {
		progress <- "Processing..."
		time.Sleep(10 * time.Millisecond) // Give time for progress update
		return nil
	}

	worker := NewWorker(repo, execFn)
	worker.pollAndExecute(context.Background())

	job, _ := repo.GetByID(context.Background(), "job-1")
	assert.Equal(t, StatusCompleted, job.Status)
	assert.Equal(t, "Ingestion completed successfully.", job.Progress)
}

func TestWorker_FailureAndRetry(t *testing.T) {
	repo := &mockJobRepository{
		jobs: map[string]*Job{
			"job-1": {
				ID:         "job-1",
				Status:     StatusPending,
				MaxRetries: 3,
				Retries:    0,
			},
		},
	}

	execFn := func(ctx context.Context, job *Job, progress chan<- string) error {
		return errors.New("temporary network error")
	}

	worker := NewWorker(repo, execFn)
	worker.pollAndExecute(context.Background())

	job, _ := repo.GetByID(context.Background(), "job-1")
	assert.Equal(t, StatusFailed, job.Status)
	assert.Equal(t, 1, job.Retries)
	require.NotNil(t, job.ErrorMsg)
	assert.Equal(t, "temporary network error", *job.ErrorMsg)
	assert.Contains(t, job.Progress, "retrying (1/3)")
	assert.NotNil(t, job.LockedUntil)
	assert.True(t, job.LockedUntil.After(time.Now()))
}

func TestWorker_MaxRetriesExhausted(t *testing.T) {
	repo := &mockJobRepository{
		jobs: map[string]*Job{
			"job-1": {
				ID:         "job-1",
				Status:     StatusPending,
				MaxRetries: 3,
				Retries:    2, // Next failure will push it to 3
			},
		},
	}

	execFn := func(ctx context.Context, job *Job, progress chan<- string) error {
		return errors.New("fatal error")
	}

	worker := NewWorker(repo, execFn)
	worker.pollAndExecute(context.Background())

	job, _ := repo.GetByID(context.Background(), "job-1")
	assert.Equal(t, StatusFailed, job.Status)
	assert.Equal(t, 3, job.Retries)
	assert.Equal(t, "Job failed after maximum retries.", job.Progress)
	assert.Nil(t, job.LockedUntil)
}
