package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestEnsureSchema(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		// Expect 6 table/index creation executions
		for i := 0; i < 6; i++ {
			mock.ExpectExec("CREATE (TABLE|INDEX) IF NOT EXISTS").WillReturnResult(pgxmock.NewResult("CREATE", 0))
		}

		err := EnsureSchema(ctx, mock)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %s", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS").WillReturnError(errors.New("db error"))

		err := EnsureSchema(ctx, mock)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
