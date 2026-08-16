package ingestion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"
)

const (
	defaultPollInterval  = 2 * time.Second
	defaultLeaseDuration = 5 * time.Minute
	progressChanSize     = 10
	backoffMultiplier    = 10
)

// Worker represents the background worker process responsible for polling
// the ingestion job queue, executing jobs, updating progress, and handling errors.
type Worker struct {
	repo          JobRepository
	pollInterval  time.Duration
	leaseDuration time.Duration
	executeFn     func(ctx context.Context, job *Job, progress chan<- string) error
}

// NewWorker initializes a new background ingestion worker.
func NewWorker(repo JobRepository, executeFn func(ctx context.Context, job *Job, progress chan<- string) error) *Worker {
	return &Worker{
		repo:          repo,
		pollInterval:  defaultPollInterval,
		leaseDuration: defaultLeaseDuration,
		executeFn:     executeFn,
	}
}

// Start begins the background polling loop. It blocks until the context is canceled.
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.pollAndExecute(ctx)
		}
	}
}

func (w *Worker) pollAndExecute(ctx context.Context) {
	job, err := w.repo.ClaimNextJob(ctx, w.leaseDuration)
	if err != nil {
		if errors.Is(err, ErrNoJobsAvailable) {
			return // No jobs available, just return and wait for next tick
		}
		slog.Error("failed to claim next job", "error", err)
		return
	}

	progressCh := make(chan string, progressChanSize)
	progressDoneCh := make(chan struct{})
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Goroutine to periodically update job progress in the database
	go func() {
		defer close(progressDoneCh)
		for {
			select {
			case <-execCtx.Done():
				return
			case p, ok := <-progressCh:
				if !ok {
					return
				}
				job.Progress = p
				job.UpdatedAt = time.Now()
				// Extend lease while working
				lockedUntil := time.Now().Add(w.leaseDuration)
				job.LockedUntil = &lockedUntil

				if updateErr := w.repo.Update(execCtx, job); updateErr != nil {
					slog.Error("failed to update job progress", "job_id", job.ID, "error", updateErr)
				}
			}
		}
	}()

	err = w.executeFn(execCtx, job, progressCh)

	cancel()         // ensure progress updater exits
	<-progressDoneCh // wait for progress updater to finish

	if err != nil {
		w.handleFailure(ctx, job, err)
		return
	}

	w.handleSuccess(ctx, job)
}

func (w *Worker) handleSuccess(ctx context.Context, job *Job) {
	job.Status = StatusCompleted
	job.Progress = "Ingestion completed successfully."
	job.UpdatedAt = time.Now()
	job.LockedUntil = nil
	if updateErr := w.repo.Update(ctx, job); updateErr != nil {
		slog.Error("failed to update job success status", "job_id", job.ID, "error", updateErr)
	}
}

func (w *Worker) handleFailure(ctx context.Context, job *Job, err error) {
	job.Retries++
	errMsg := err.Error()
	job.ErrorMsg = &errMsg
	job.UpdatedAt = time.Now()

	if job.Retries >= job.MaxRetries {
		job.Status = StatusFailed
		job.Progress = "Job failed after maximum retries."
		job.LockedUntil = nil
	} else {
		job.Status = StatusPending
		job.Progress = fmt.Sprintf("Job failed, retrying (%d/%d)...", job.Retries, job.MaxRetries)
		// Exponential backoff
		backoffSeconds := math.Pow(2, float64(job.Retries)) * backoffMultiplier
		lockedUntil := time.Now().Add(time.Duration(backoffSeconds) * time.Second)
		job.LockedUntil = &lockedUntil
	}
	if updateErr := w.repo.Update(ctx, job); updateErr != nil {
		slog.Error("failed to update job failure status", "job_id", job.ID, "error", updateErr)
	}
}
