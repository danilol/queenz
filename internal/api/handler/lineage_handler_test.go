package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"queenx/internal/lineage/domain"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLineageRepo struct {
	q   *domain.Queen
	s   *domain.SiblingQueryResult
	err error
}

func (m *mockLineageRepo) EnsureConstraints(ctx context.Context) error            { return nil }
func (m *mockLineageRepo) CreateQueen(ctx context.Context, q *domain.Queen) error { return nil }
func (m *mockLineageRepo) GetQueenByID(ctx context.Context, id string) (*domain.Queen, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.q, nil
}
func (m *mockLineageRepo) CreateHouse(ctx context.Context, h *domain.House) error { return nil }
func (m *mockLineageRepo) GetHouseByID(ctx context.Context, id string) (*domain.House, error) {
	return nil, nil
}
func (m *mockLineageRepo) CreateSeason(ctx context.Context, s *domain.Season) error { return nil }
func (m *mockLineageRepo) GetSeasonByID(ctx context.Context, id string) (*domain.Season, error) {
	return nil, nil
}
func (m *mockLineageRepo) AddDragMother(ctx context.Context, motherID, daughterID string) error {
	return nil
}
func (m *mockLineageRepo) AddSister(ctx context.Context, queenID1, queenID2 string) error { return nil }
func (m *mockLineageRepo) AddHouseMember(ctx context.Context, queenID, houseID string) error {
	return nil
}
func (m *mockLineageRepo) AddParticipation(ctx context.Context, queenID, seasonID, placement string, wins int) error {
	return nil
}
func (m *mockLineageRepo) AddLipSync(ctx context.Context, queenID1, queenID2, song, episodeID, winnerID string) error {
	return nil
}
func (m *mockLineageRepo) FindAestheticSiblings(ctx context.Context, queenID string, limit int) ([]*domain.SiblingQueryResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.s != nil {
		return []*domain.SiblingQueryResult{m.s}, nil
	}
	return nil, nil
}

func TestLineageHandler_FindAestheticSiblings(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?limit=5", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("q1")

	q := &domain.Queen{ID: "q1", DragName: "Gigi"}
	sib := &domain.SiblingQueryResult{Queen: &domain.Queen{ID: "q2", DragName: "Symone"}, Score: 10}
	repo := &mockLineageRepo{q: q, s: sib}
	h := NewLineageHandler(repo)

	err := h.FindAestheticSiblings(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Symone")
	assert.Contains(t, rec.Body.String(), `"score":10`)
}
