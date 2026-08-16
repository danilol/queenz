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

func TestFranchiseRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewFranchiseRepository(mock)
	ctx := context.Background()
	now := time.Now()

	f := &domain.Franchise{
		ID:        "f-1",
		Name:      "RuPaul's Drag Race",
		Country:   "USA",
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO franchises").
			WithArgs(f.ID, f.Name, f.Country, f.CreatedAt, f.UpdatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(ctx, f)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO franchises").
			WithArgs(f.ID, f.Name, f.Country, f.CreatedAt, f.UpdatedAt).
			WillReturnError(errors.New("db error"))

		err := repo.Create(ctx, f)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("already exists", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO franchises").
			WithArgs(f.ID, f.Name, f.Country, f.CreatedAt, f.UpdatedAt).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err := repo.Create(ctx, f)
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Errorf("expected ErrAlreadyExists, got %v", err)
		}
	})
}

func TestFranchiseRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewFranchiseRepository(mock)
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		columns := []string{"id", "name", "country", "created_at", "updated_at"}
		mock.ExpectQuery("SELECT id, name, country, created_at, updated_at FROM franchises").
			WithArgs("f-1").
			WillReturnRows(pgxmock.NewRows(columns).AddRow("f-1", "RPDR", "USA", now, now))

		f, err := repo.GetByID(ctx, "f-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.ID != "f-1" || f.Name != "RPDR" {
			t.Errorf("unexpected result: %+v", f)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, name, country, created_at, updated_at FROM franchises").
			WithArgs("f-1").
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetByID(ctx, "f-1")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, name, country, created_at, updated_at FROM franchises").
			WithArgs("f-1").
			WillReturnError(errors.New("db error"))

		_, err := repo.GetByID(ctx, "f-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestFranchiseRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewFranchiseRepository(mock)
	ctx := context.Background()
	now := time.Now()

	f := &domain.Franchise{
		ID:        "f-1",
		Name:      "RPDR All Stars",
		Country:   "USA",
		UpdatedAt: now,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("UPDATE franchises").
			WithArgs(f.Name, f.Country, f.UpdatedAt, f.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(ctx, f)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("UPDATE franchises").
			WithArgs(f.Name, f.Country, f.UpdatedAt, f.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(ctx, f)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("UPDATE franchises").
			WithArgs(f.Name, f.Country, f.UpdatedAt, f.ID).
			WillReturnError(errors.New("db error"))

		err := repo.Update(ctx, f)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestFranchiseRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewFranchiseRepository(mock)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM franchises").
			WithArgs("f-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(ctx, "f-1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM franchises").
			WithArgs("f-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(ctx, "f-1")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM franchises").
			WithArgs("f-1").
			WillReturnError(errors.New("db error"))

		err := repo.Delete(ctx, "f-1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestFranchiseRepository_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	repo := NewFranchiseRepository(mock)
	ctx := context.Background()
	now := time.Now()

	columns := []string{"id", "name", "country", "created_at", "updated_at"}

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, name, country, created_at, updated_at FROM franchises").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("f-1", "RPDR", "USA", now, now).
				AddRow("f-2", "DR España", "Spain", now, now))

		list, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("expected 2 franchises, got %d", len(list))
		}
	})

	t.Run("query error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, name, country, created_at, updated_at FROM franchises").
			WillReturnError(errors.New("db error"))

		_, err := repo.List(ctx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("scan error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, name, country, created_at, updated_at FROM franchises").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("f-1", "RPDR", "USA", 123, now)) // int 123 cannot be scanned into time.Time

		_, err := repo.List(ctx)
		if err == nil {
			t.Error("expected scan error, got nil")
		}
	})

	t.Run("rows error", func(t *testing.T) {
		rows := pgxmock.NewRows(columns).
			AddRow("f-1", "RPDR", "USA", now, now).
			RowError(0, errors.New("row iteration error"))

		mock.ExpectQuery("SELECT id, name, country, created_at, updated_at FROM franchises").
			WillReturnRows(rows)

		_, err := repo.List(ctx)
		if err == nil {
			t.Error("expected row iteration error, got nil")
		}
	})
}
