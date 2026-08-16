package ingestion

import "errors"

var (
	// ErrJobNotFound is returned when a requested job does not exist in the database.
	ErrJobNotFound = errors.New("job not found")

	// ErrNoJobsAvailable is returned by ClaimNextJob when the queue is currently empty.
	ErrNoJobsAvailable = errors.New("no jobs available to claim")
)
