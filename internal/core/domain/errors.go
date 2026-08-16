// Package domain defines the core business entities, interfaces, and shared rules
// for the QueenX application, establishing consistent, technology-agnostic boundaries.
package domain

import "errors"

var (
	// ErrNotFound represents a sentinel error returned by repository and service layers
	// when a requested entity cannot be found in the database or graph store.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists represents a sentinel error returned during entity creation
	// when a uniqueness constraint is violated (e.g., duplicated UUIDs, stage names, or keys).
	ErrAlreadyExists = errors.New("resource already exists")
)
