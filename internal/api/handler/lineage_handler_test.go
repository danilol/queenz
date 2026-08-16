package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"queenx/internal/lineage/domain"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockLineageRepo struct {
	mock.Mock
}

func (m *mockLineageRepo) GetQueenByID(ctx context.Context, id string) (*domain.Queen, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Queen), args.Error(1)
}

func (m *mockLineageRepo) FindAestheticSiblings(ctx context.Context, queenID string, limit int) ([]*domain.SiblingQueryResult, error) {
	args := m.Called(ctx, queenID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.SiblingQueryResult), args.Error(1)
}

func TestLineageHandler_FindAestheticSiblings(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?limit=5", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("q1")

	repo := new(mockLineageRepo)
	repo.On("GetQueenByID", mock.Anything, "q1").Return(&domain.Queen{ID: "q1", DragName: "Gigi"}, nil)
	repo.On("FindAestheticSiblings", mock.Anything, "q1", 5).Return([]*domain.SiblingQueryResult{
		{Queen: &domain.Queen{ID: "q2", DragName: "Symone"}, Score: 10},
	}, nil)

	h := NewLineageHandler(repo)

	err := h.FindAestheticSiblings(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Symone")
	assert.Contains(t, rec.Body.String(), `"score":10`)
}
