package domain

import (
	"context"
)

// FranchiseRepository defines the data store operations for Franchises.
type FranchiseRepository interface {
	Create(ctx context.Context, f *Franchise) error
	GetByID(ctx context.Context, id string) (*Franchise, error)
	Update(ctx context.Context, f *Franchise) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Franchise, error)
}

// SeasonRepository defines the data store operations for Seasons.
type SeasonRepository interface {
	Create(ctx context.Context, s *Season) error
	GetByID(ctx context.Context, id string) (*Season, error)
	Update(ctx context.Context, s *Season) error
	Delete(ctx context.Context, id string) error
	ListByFranchiseID(ctx context.Context, franchiseID string) ([]*Season, error)
}
