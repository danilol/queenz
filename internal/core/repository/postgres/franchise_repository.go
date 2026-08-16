package postgres

import (
	"context"
	"errors"
	"fmt"

	"queenx/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type franchiseRepository struct {
	db DB
}

// NewFranchiseRepository creates a new PostgreSQL implementation of domain.FranchiseRepository.
func NewFranchiseRepository(db DB) domain.FranchiseRepository {
	return &franchiseRepository{db: db}
}

func (r *franchiseRepository) Create(ctx context.Context, f *domain.Franchise) error {
	query := `
		INSERT INTO franchises (id, name, country, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query, f.ID, f.Name, f.Country, f.CreatedAt, f.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return fmt.Errorf("creating franchise: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("creating franchise: %w", err)
	}
	return nil
}

func (r *franchiseRepository) GetByID(ctx context.Context, id string) (*domain.Franchise, error) {
	query := `
		SELECT id, name, country, created_at, updated_at
		FROM franchises
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	var f domain.Franchise
	err := row.Scan(&f.ID, &f.Name, &f.Country, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting franchise by id: %w", err)
	}
	return &f, nil
}

func (r *franchiseRepository) Update(ctx context.Context, f *domain.Franchise) error {
	query := `
		UPDATE franchises
		SET name = $1, country = $2, updated_at = $3
		WHERE id = $4
	`
	cmdTag, err := r.db.Exec(ctx, query, f.Name, f.Country, f.UpdatedAt, f.ID)
	if err != nil {
		return fmt.Errorf("updating franchise: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *franchiseRepository) Delete(ctx context.Context, id string) error {
	query := `
		DELETE FROM franchises
		WHERE id = $1
	`
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting franchise: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *franchiseRepository) List(ctx context.Context) ([]*domain.Franchise, error) {
	query := `
		SELECT id, name, country, created_at, updated_at
		FROM franchises
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing franchises: %w", err)
	}
	defer rows.Close()

	var franchises []*domain.Franchise
	for rows.Next() {
		var f domain.Franchise
		if err := rows.Scan(&f.ID, &f.Name, &f.Country, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning franchise row: %w", err)
		}
		franchises = append(franchises, &f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating franchise rows: %w", err)
	}

	return franchises, nil
}
