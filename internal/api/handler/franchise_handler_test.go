package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"queenx/internal/core/domain"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFranchiseRepo struct {
	f   *domain.Franchise
	err error
}

func (m *mockFranchiseRepo) Create(ctx context.Context, f *domain.Franchise) error { return nil }
func (m *mockFranchiseRepo) GetByID(ctx context.Context, id string) (*domain.Franchise, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.f, nil
}
func (m *mockFranchiseRepo) Update(ctx context.Context, f *domain.Franchise) error { return nil }
func (m *mockFranchiseRepo) Delete(ctx context.Context, id string) error           { return nil }
func (m *mockFranchiseRepo) List(ctx context.Context) ([]*domain.Franchise, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.f != nil {
		return []*domain.Franchise{m.f}, nil
	}
	return nil, nil
}

type mockSeasonRepo struct {
	s   *domain.Season
	err error
}

func (m *mockSeasonRepo) Create(ctx context.Context, s *domain.Season) error { return nil }
func (m *mockSeasonRepo) GetByID(ctx context.Context, id string) (*domain.Season, error) {
	return nil, nil
}
func (m *mockSeasonRepo) Update(ctx context.Context, s *domain.Season) error { return nil }
func (m *mockSeasonRepo) Delete(ctx context.Context, id string) error        { return nil }
func (m *mockSeasonRepo) ListByFranchiseID(ctx context.Context, id string) ([]*domain.Season, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.s != nil {
		return []*domain.Season{m.s}, nil
	}
	return nil, nil
}

func TestFranchiseHandler_ListFranchises(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/franchises", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	fRepo := &mockFranchiseRepo{f: &domain.Franchise{ID: "f1", Name: "RPDR"}}
	sRepo := &mockSeasonRepo{}
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

	fRepo := &mockFranchiseRepo{f: &domain.Franchise{ID: "f1", Name: "RPDR"}}
	sRepo := &mockSeasonRepo{}
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

	fRepo := &mockFranchiseRepo{f: &domain.Franchise{ID: "f1", Name: "RPDR"}}
	sRepo := &mockSeasonRepo{s: &domain.Season{ID: "s1", Name: "Season 1"}}
	h := NewFranchiseHandler(fRepo, sRepo)

	err := h.ListSeasons(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Season 1")
}
