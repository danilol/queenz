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

func TestSeasonRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewSeasonRepository(mock)
	ctx := context.Background()
	now := time.Now()

	s := &domain.Season{
		ID:          "s-1",
		FranchiseID: "f-1",
		Name:        "Season 1",
		Number:      1,
		AirDate:     now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO seasons").
			WithArgs(s.ID, s.FranchiseID, s.Name, s.Number, s.AirDate, s.CreatedAt, s.UpdatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(ctx, s)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO seasons").
			WithArgs(s.ID, s.FranchiseID, s.Name, s.Number, s.AirDate, s.CreatedAt, s.UpdatedAt).
			WillReturnError(errors.New("db error"))

		err := repo.Create(ctx, s)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("already exists", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO seasons").
			WithArgs(s.ID, s.FranchiseID, s.Name, s.Number, s.AirDate, s.CreatedAt, s.UpdatedAt).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err := repo.Create(ctx, s)
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Errorf("expected ErrAlreadyExists, got %v", err)
		}
	})
}

func TestSeasonRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewSeasonRepository(mock)
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		columns := []string{"id", "franchise_id", "name", "number", "air_date", "created_at", "updated_at"}
		mock.ExpectQuery("SELECT id, franchise_id, name, number, air_date, created_at, updated_at FROM seasons").
			WithArgs("s-1").
			WillReturnRows(pgxmock.NewRows(columns).AddRow("s-1", "f-1", "Season 1", 1, now, now, now))

		s, err := repo.GetByID(ctx, "s-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != "s-1" || s.Name != "Season 1" {
			t.Errorf("unexpected result: %+v", s)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, franchise_id, name, number, air_date, created_at, updated_at FROM seasons").
			WithArgs("s-1").
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetByID(ctx, "s-1")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, franchise_id, name, number, air_date, created_at, updated_at FROM seasons").
			WithArgs("s-1").
			WillReturnError(errors.New("db error"))

		_, err := repo.GetByID(ctx, "s-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestSeasonRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewSeasonRepository(mock)
	ctx := context.Background()
	now := time.Now()

	s := &domain.Season{
		ID:          "s-1",
		FranchiseID: "f-1",
		Name:        "Season 1 Updated",
		Number:      1,
		AirDate:     now,
		UpdatedAt:   now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("UPDATE seasons").
			WithArgs(s.FranchiseID, s.Name, s.Number, s.AirDate, s.UpdatedAt, s.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(ctx, s)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("UPDATE seasons").
			WithArgs(s.FranchiseID, s.Name, s.Number, s.AirDate, s.UpdatedAt, s.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(ctx, s)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("UPDATE seasons").
			WithArgs(s.FranchiseID, s.Name, s.Number, s.AirDate, s.UpdatedAt, s.ID).
			WillReturnError(errors.New("db error"))

		err := repo.Update(ctx, s)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestSeasonRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewSeasonRepository(mock)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM seasons").
			WithArgs("s-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(ctx, "s-1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM seasons").
			WithArgs("s-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(ctx, "s-1")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM seasons").
			WithArgs("s-1").
			WillReturnError(errors.New("db error"))

		err := repo.Delete(ctx, "s-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestSeasonRepository_ListByFranchiseID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewSeasonRepository(mock)
	ctx := context.Background()
	now := time.Now()

	columns := []string{"id", "franchise_id", "name", "number", "air_date", "created_at", "updated_at"}

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, franchise_id, name, number, air_date, created_at, updated_at FROM seasons").
			WithArgs("f-1").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("s-1", "f-1", "Season 1", 1, now, now, now).
				AddRow("s-2", "f-1", "Season 2", 2, now, now, now))

		list, err := repo.ListByFranchiseID(ctx, "f-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("expected 2 seasons, got %d", len(list))
		}
	})

	t.Run("query error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, franchise_id, name, number, air_date, created_at, updated_at FROM seasons").
			WithArgs("f-1").
			WillReturnError(errors.New("db error"))

		_, err := repo.ListByFranchiseID(ctx, "f-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("scan error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, franchise_id, name, number, air_date, created_at, updated_at FROM seasons").
			WithArgs("f-1").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("s-1", "f-1", "Season 1", 1, 123, now, now)) // int 123 cannot be scanned into time.Time

		_, err := repo.ListByFranchiseID(ctx, "f-1")
		if err == nil {
			t.Error("expected scan error, got nil")
		}
	})

	t.Run("rows error", func(t *testing.T) {
		rows := pgxmock.NewRows(columns).
			AddRow("s-1", "f-1", "Season 1", 1, now, now, now).
			RowError(0, errors.New("row iteration error"))

		mock.ExpectQuery("SELECT id, franchise_id, name, number, air_date, created_at, updated_at FROM seasons").
			WithArgs("f-1").
			WillReturnRows(rows)

		_, err := repo.ListByFranchiseID(ctx, "f-1")
		if err == nil {
			t.Error("expected row iteration error, got nil")
		}
	})
}
