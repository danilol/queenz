package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"queenx/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestEpisodeRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewEpisodeRepository(mock)
	ctx := context.Background()
	now := time.Now()

	e := &domain.Episode{
		ID:        "e-1",
		SeasonID:  "s-1",
		Title:     "Snatch Game",
		Number:    5,
		AirDate:   now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO episodes").
			WithArgs(e.ID, e.SeasonID, e.Title, e.Number, e.AirDate, e.CreatedAt, e.UpdatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(ctx, e)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO episodes").
			WithArgs(e.ID, e.SeasonID, e.Title, e.Number, e.AirDate, e.CreatedAt, e.UpdatedAt).
			WillReturnError(errors.New("db error"))

		err := repo.Create(ctx, e)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("already exists", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO episodes").
			WithArgs(e.ID, e.SeasonID, e.Title, e.Number, e.AirDate, e.CreatedAt, e.UpdatedAt).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err := repo.Create(ctx, e)
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Errorf("expected ErrAlreadyExists, got %v", err)
		}
	})
}

func TestEpisodeRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewEpisodeRepository(mock)
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		columns := []string{"id", "season_id", "title", "number", "air_date", "created_at", "updated_at"}
		mock.ExpectQuery("SELECT id, season_id, title, number, air_date, created_at, updated_at FROM episodes").
			WithArgs("e-1").
			WillReturnRows(pgxmock.NewRows(columns).AddRow("e-1", "s-1", "Snatch Game", 5, now, now, now))

		e, err := repo.GetByID(ctx, "e-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.ID != "e-1" || e.Title != "Snatch Game" {
			t.Errorf("unexpected result: %+v", e)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, season_id, title, number, air_date, created_at, updated_at FROM episodes").
			WithArgs("e-1").
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetByID(ctx, "e-1")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, season_id, title, number, air_date, created_at, updated_at FROM episodes").
			WithArgs("e-1").
			WillReturnError(errors.New("db error"))

		_, err := repo.GetByID(ctx, "e-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestEpisodeRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewEpisodeRepository(mock)
	ctx := context.Background()
	now := time.Now()

	e := &domain.Episode{
		ID:        "e-1",
		Title:     "Snatch Game Edit",
		Number:    5,
		AirDate:   now,
		UpdatedAt: now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("UPDATE episodes").
			WithArgs(e.Title, e.Number, e.AirDate, e.UpdatedAt, e.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(ctx, e)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("UPDATE episodes").
			WithArgs(e.Title, e.Number, e.AirDate, e.UpdatedAt, e.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(ctx, e)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("UPDATE episodes").
			WithArgs(e.Title, e.Number, e.AirDate, e.UpdatedAt, e.ID).
			WillReturnError(errors.New("db error"))

		err := repo.Update(ctx, e)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestEpisodeRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewEpisodeRepository(mock)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM episodes").
			WithArgs("e-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(ctx, "e-1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM episodes").
			WithArgs("e-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(ctx, "e-1")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM episodes").
			WithArgs("e-1").
			WillReturnError(errors.New("db error"))

		err := repo.Delete(ctx, "e-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestEpisodeRepository_ListBySeasonID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewEpisodeRepository(mock)
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		columns := []string{"id", "season_id", "title", "number", "air_date", "created_at", "updated_at"}
		mock.ExpectQuery("SELECT id, season_id, title, number, air_date, created_at, updated_at FROM episodes").
			WithArgs("s-1").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("e-1", "s-1", "Ep 1", 1, now, now, now).
				AddRow("e-2", "s-1", "Ep 2", 2, now, now, now))

		episodes, err := repo.ListBySeasonID(ctx, "s-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(episodes) != 2 {
			t.Errorf("expected 2 episodes, got %d", len(episodes))
		}
		if episodes[0].ID != "e-1" || episodes[1].ID != "e-2" {
			t.Errorf("unexpected elements in list: %+v", episodes)
		}
	})

	t.Run("error query", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, season_id, title, number, air_date, created_at, updated_at FROM episodes").
			WithArgs("s-1").
			WillReturnError(errors.New("db error"))

		_, err := repo.ListBySeasonID(ctx, "s-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("error row scan", func(t *testing.T) {
		columns := []string{"id", "season_id", "title"} // Invalid columns
		mock.ExpectQuery("SELECT id, season_id, title, number, air_date, created_at, updated_at FROM episodes").
			WithArgs("s-1").
			WillReturnRows(pgxmock.NewRows(columns).AddRow("e-1", "s-1", "Title"))

		_, err := repo.ListBySeasonID(ctx, "s-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("error row iteration", func(t *testing.T) {
		columns := []string{"id", "season_id", "title", "number", "air_date", "created_at", "updated_at"}
		rows := pgxmock.NewRows(columns).AddRow("e-1", "s-1", "Ep 1", 1, now, now, now).RowError(0, errors.New("iteration error"))
		mock.ExpectQuery("SELECT id, season_id, title, number, air_date, created_at, updated_at FROM episodes").
			WithArgs("s-1").
			WillReturnRows(rows)

		_, err := repo.ListBySeasonID(ctx, "s-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
