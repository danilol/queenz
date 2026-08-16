package domain

import (
	"context"
)

// FranchiseRepository defines the database abstraction and data-store operations
// for the Franchise domain model in the relational repository layer.
type FranchiseRepository interface {
	// Create persists a new Franchise record in the relational store.
	// Returns ErrAlreadyExists if a Franchise with the same ID or unique name/country combo already exists.
	Create(ctx context.Context, f *Franchise) error

	// GetByID retrieves a single Franchise by its unique ID.
	// Returns ErrNotFound if the Franchise cannot be located.
	GetByID(ctx context.Context, id string) (*Franchise, error)

	// Update modifies the fields of an existing Franchise.
	// Returns ErrNotFound if the record does not exist.
	Update(ctx context.Context, f *Franchise) error

	// Delete removes a Franchise and its cascade-eligible records from the store.
	// Returns ErrNotFound if the record does not exist.
	Delete(ctx context.Context, id string) error

	// List retrieves all recorded Franchises from the store.
	// Returns an empty slice and no error if no records exist.
	List(ctx context.Context) ([]*Franchise, error)
}

// SeasonRepository defines the database abstraction and data-store operations
// for the Season domain model in the relational repository layer.
type SeasonRepository interface {
	// Create persists a new Season record in the relational store.
	// Returns ErrAlreadyExists if a Season with the same ID already exists.
	Create(ctx context.Context, s *Season) error

	// GetByID retrieves a single Season by its unique ID.
	// Returns ErrNotFound if the Season cannot be located.
	GetByID(ctx context.Context, id string) (*Season, error)

	// Update modifies the fields of an existing Season.
	// Returns ErrNotFound if the record does not exist.
	Update(ctx context.Context, s *Season) error

	// Delete removes a Season and its dependent sub-entities from the store.
	// Returns ErrNotFound if the record does not exist.
	Delete(ctx context.Context, id string) error

	// ListByFranchiseID retrieves all Seasons belonging to a specific Franchise ID.
	// Returns an empty slice and no error if no seasons are found.
	ListByFranchiseID(ctx context.Context, franchiseID string) ([]*Season, error)
}
