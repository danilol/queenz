package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"queenx/internal/lineage/domain"

	"github.com/labstack/echo/v4"
)

const (
	defaultSiblingLimit = 10
	maxSiblingLimit     = 50
)

type lineageReader interface {
	GetQueenByID(ctx context.Context, id string) (*domain.Queen, error)
	FindAestheticSiblings(ctx context.Context, queenID string, limit int) ([]*domain.SiblingQueryResult, error)
}

// LineageHandler handles HTTP requests related to the Neo4j Social Graph context.
type LineageHandler struct {
	repo lineageReader
}

// NewLineageHandler creates a new LineageHandler.
func NewLineageHandler(repo lineageReader) *LineageHandler {
	return &LineageHandler{repo: repo}
}

// FindAestheticSiblings returns the top siblings for a given queen.
func (h *LineageHandler) FindAestheticSiblings(c echo.Context) error {
	queenID := c.Param("id")

	limitStr := c.QueryParam("limit")
	limit := defaultSiblingLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= maxSiblingLimit {
			limit = l
		}
	}

	// Verify queen exists in graph
	_, err := h.repo.GetQueenByID(c.Request().Context(), queenID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "queen not found in graph")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to query graph")
	}

	siblings, err := h.repo.FindAestheticSiblings(c.Request().Context(), queenID, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to find siblings")
	}

	if siblings == nil {
		siblings = []*domain.SiblingQueryResult{}
	}

	return c.JSON(http.StatusOK, siblings)
}
