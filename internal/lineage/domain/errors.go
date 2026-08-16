package domain

import "errors"

var (
	// ErrNotFound represents a sentinel error returned when a requested node (Queen, House, Season)
	// cannot be located in the graph database.
	ErrNotFound = errors.New("graph entity not found")

	// ErrAlreadyExists is returned when attempting to create a node that already exists
	// under a uniqueness constraint.
	ErrAlreadyExists = errors.New("graph entity already exists")

	// ErrInvalidRelationship is returned when attempting to create a relationship
	// with invalid or missing parameters (e.g. self-referencing relationship).
	ErrInvalidRelationship = errors.New("invalid graph relationship")
)
