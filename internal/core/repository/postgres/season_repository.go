package postgres

import (
	"context"
	"errors"
	"fmt"

	"queenx/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type seasonRepository struct {
	db DB
}

// NewSeasonRepository creates a new PostgreSQL implementation of domain.SeasonRepository.
func NewSeasonRepository(db DB) domain.SeasonRepository {
	return &seasonRepository{db: db}
}

func (r *seasonRepository) Create(ctx context.Context, s *domain.Season) error {
	query := `
		INSERT INTO seasons (id, franchise_id, name, number, air_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query, s.ID, s.FranchiseID, s.Name, s.Number, s.AirDate, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("creating season: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("creating season: %w", err)
	}
	return nil
}

func (r *seasonRepository) GetByID(ctx context.Context, id string) (*domain.Season, error) {
	query := `
		SELECT id, franchise_id, name, number, air_date, created_at, updated_at
		FROM seasons
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	var s domain.Season
	err := row.Scan(&s.ID, &s.FranchiseID, &s.Name, &s.Number, &s.AirDate, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting season by id: %w", err)
	}
	return &s, nil
}

func (r *seasonRepository) Update(ctx context.Context, s *domain.Season) error {
	query := `
		UPDATE seasons
		SET franchise_id = $1, name = $2, number = $3, air_date = $4, updated_at = $5
		WHERE id = $6
	`
	cmdTag, err := r.db.Exec(ctx, query, s.FranchiseID, s.Name, s.Number, s.AirDate, s.UpdatedAt, s.ID)
	if err != nil {
		return fmt.Errorf("updating season: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *seasonRepository) Delete(ctx context.Context, id string) error {
	query := `
		DELETE FROM seasons
		WHERE id = $1
	`
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting season: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

//nolint:dupl // similar list method for child relation
func (r *seasonRepository) ListByFranchiseID(ctx context.Context, franchiseID string) ([]*domain.Season, error) {
	query := `
		SELECT id, franchise_id, name, number, air_date, created_at, updated_at
		FROM seasons
		WHERE franchise_id = $1
		ORDER BY number ASC
	`
	rows, err := r.db.Query(ctx, query, franchiseID)
	if err != nil {
		return nil, fmt.Errorf("listing seasons by franchise id: %w", err)
	}
	defer rows.Close()

	var seasons []*domain.Season
	for rows.Next() {
		var s domain.Season
		if err := rows.Scan(&s.ID, &s.FranchiseID, &s.Name, &s.Number, &s.AirDate, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning season row: %w", err)
		}
		seasons = append(seasons, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating season rows: %w", err)
	}

	return seasons, nil
}
