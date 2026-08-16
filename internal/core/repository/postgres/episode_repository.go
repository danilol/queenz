package postgres

import (
	"context"
	"errors"
	"fmt"

	"queenx/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type episodeRepository struct {
	db DB
}

// NewEpisodeRepository creates a new PostgreSQL implementation of domain.EpisodeRepository.
func NewEpisodeRepository(db DB) domain.EpisodeRepository {
	return &episodeRepository{db: db}
}

func (r *episodeRepository) Create(ctx context.Context, e *domain.Episode) error {
	query := `
		INSERT INTO episodes (id, season_id, title, number, air_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query, e.ID, e.SeasonID, e.Title, e.Number, e.AirDate, e.CreatedAt, e.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("creating episode: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("creating episode: %w", err)
	}
	return nil
}

func (r *episodeRepository) GetByID(ctx context.Context, id string) (*domain.Episode, error) {
	query := `
		SELECT id, season_id, title, number, air_date, created_at, updated_at
		FROM episodes
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	var e domain.Episode
	err := row.Scan(&e.ID, &e.SeasonID, &e.Title, &e.Number, &e.AirDate, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting episode by id: %w", err)
	}
	return &e, nil
}

func (r *episodeRepository) Update(ctx context.Context, e *domain.Episode) error {
	query := `
		UPDATE episodes
		SET title = $1, number = $2, air_date = $3, updated_at = $4
		WHERE id = $5
	`
	cmdTag, err := r.db.Exec(ctx, query, e.Title, e.Number, e.AirDate, e.UpdatedAt, e.ID)
	if err != nil {
		return fmt.Errorf("updating episode: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *episodeRepository) Delete(ctx context.Context, id string) error {
	query := `
		DELETE FROM episodes
		WHERE id = $1
	`
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting episode: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

//nolint:dupl // similar list method for child relation
func (r *episodeRepository) ListBySeasonID(ctx context.Context, seasonID string) ([]*domain.Episode, error) {
	query := `
		SELECT id, season_id, title, number, air_date, created_at, updated_at
		FROM episodes
		WHERE season_id = $1
		ORDER BY number ASC
	`
	rows, err := r.db.Query(ctx, query, seasonID)
	if err != nil {
		return nil, fmt.Errorf("listing episodes: %w", err)
	}
	defer rows.Close()

	var episodes []*domain.Episode
	for rows.Next() {
		var e domain.Episode
		if err := rows.Scan(&e.ID, &e.SeasonID, &e.Title, &e.Number, &e.AirDate, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning episode row: %w", err)
		}
		episodes = append(episodes, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating episode rows: %w", err)
	}

	return episodes, nil
}
