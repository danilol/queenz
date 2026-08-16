package domain

import "errors"

var (
	// ErrNotFound is returned when a resource is not found in the database.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when attempting to create a resource with an existing unique identifier.
	ErrAlreadyExists = errors.New("resource already exists")
)
