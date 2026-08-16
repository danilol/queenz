package ingestion

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockJobRepository implements JobRepository for testing.
type mockJobRepository struct {
	mu              sync.Mutex
	jobs            map[string]*Job
	claimError      error
	updateError     error
	claimCallCount  int
	updateCallCount int
}

func (m *mockJobRepository) Create(ctx context.Context, job *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *mockJobRepository) GetByID(ctx context.Context, id string) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, ok := m.jobs[id]; ok {
		// Return a copy so callers don't mutate state directly
		j := *job
		return &j, nil
	}
	return nil, ErrJobNotFound
}

func (m *mockJobRepository) Update(ctx context.Context, job *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCallCount++
	if m.updateError != nil {
		return m.updateError
	}
	// Copy to store
	j := *job
	m.jobs[job.ID] = &j
	return nil
}

func (m *mockJobRepository) ClaimNextJob(ctx context.Context, leaseDuration time.Duration) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.claimCallCount++
	if m.claimError != nil {
		return nil, m.claimError
	}
	for _, job := range m.jobs {
		if job.Status == StatusPending {
			job.Status = StatusRunning
			// Return a copy
			j := *job
			return &j, nil
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
		// Do not sleep. Rely on progress context cancellation sync.
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
	assert.Equal(t, StatusPending, job.Status) // Retrying sets it back to pending
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

func TestWorker_UpdateErrorLogging(t *testing.T) {
	repo := &mockJobRepository{
		jobs: map[string]*Job{
			"job-1": {
				ID:         "job-1",
				Status:     StatusPending,
				MaxRetries: 3,
			},
		},
		updateError: errors.New("simulated db update error"),
	}

	execFn := func(ctx context.Context, job *Job, progress chan<- string) error {
		return nil
	}

	worker := NewWorker(repo, execFn)
	worker.pollAndExecute(context.Background())

	// The job should remain untouched in repo since update fails
	// It was picked up by ClaimNextJob, setting status to StatusRunning, but update fails
	job, _ := repo.GetByID(context.Background(), "job-1")
	assert.Equal(t, StatusRunning, job.Status) // Was running locally and db never updated to success
	repo.mu.Lock()
	assert.GreaterOrEqual(t, repo.updateCallCount, 1)
	repo.mu.Unlock()
}
