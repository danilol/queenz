package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"queenx/internal/core/domain"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockFranchiseRepo struct {
	mock.Mock
}

func (m *mockFranchiseRepo) GetByID(ctx context.Context, id string) (*domain.Franchise, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Franchise), args.Error(1)
}

func (m *mockFranchiseRepo) List(ctx context.Context) ([]*domain.Franchise, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Franchise), args.Error(1)
}

type mockSeasonRepo struct {
	mock.Mock
}

func (m *mockSeasonRepo) ListByFranchiseID(ctx context.Context, id string) ([]*domain.Season, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Season), args.Error(1)
}

func TestFranchiseHandler_ListFranchises(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/franchises", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	fRepo := new(mockFranchiseRepo)
	fRepo.On("List", mock.Anything).Return([]*domain.Franchise{{ID: "f1", Name: "RPDR"}}, nil)
	sRepo := new(mockSeasonRepo)
	h := NewFranchiseHandler(fRepo, sRepo)

	err := h.ListFranchises(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "RPDR")
}

func TestFranchiseHandler_GetFranchise(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("f1")

	fRepo := new(mockFranchiseRepo)
	fRepo.On("GetByID", mock.Anything, "f1").Return(&domain.Franchise{ID: "f1", Name: "RPDR"}, nil)
	sRepo := new(mockSeasonRepo)
	h := NewFranchiseHandler(fRepo, sRepo)

	err := h.GetFranchise(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "RPDR")
}

func TestFranchiseHandler_ListSeasons(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("f1")

	fRepo := new(mockFranchiseRepo)
	fRepo.On("GetByID", mock.Anything, "f1").Return(&domain.Franchise{ID: "f1", Name: "RPDR"}, nil)
	sRepo := new(mockSeasonRepo)
	sRepo.On("ListByFranchiseID", mock.Anything, "f1").Return([]*domain.Season{{ID: "s1", Name: "Season 1"}}, nil)
	h := NewFranchiseHandler(fRepo, sRepo)

	err := h.ListSeasons(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Season 1")
}
