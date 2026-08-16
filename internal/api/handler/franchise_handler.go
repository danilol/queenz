package handler

import (
	"context"
	"errors"
	"net/http"

	"queenx/internal/core/domain"

	"github.com/labstack/echo/v4"
)

type franchiseReader interface {
	GetByID(ctx context.Context, id string) (*domain.Franchise, error)
	List(ctx context.Context) ([]*domain.Franchise, error)
}

type seasonReader interface {
	ListByFranchiseID(ctx context.Context, franchiseID string) ([]*domain.Season, error)
}

// FranchiseHandler handles HTTP requests related to Franchises and their Seasons.
type FranchiseHandler struct {
	fRepo franchiseReader
	sRepo seasonReader
}

// NewFranchiseHandler creates a new FranchiseHandler.
func NewFranchiseHandler(fRepo franchiseReader, sRepo seasonReader) *FranchiseHandler {
	return &FranchiseHandler{
		fRepo: fRepo,
		sRepo: sRepo,
	}
}

// ListFranchises returns a list of all franchises.
func (h *FranchiseHandler) ListFranchises(c echo.Context) error {
	franchises, err := h.fRepo.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list franchises")
	}
	if franchises == nil {
		franchises = []*domain.Franchise{}
	}
	return c.JSON(http.StatusOK, franchises)
}

// GetFranchise returns details for a specific franchise.
func (h *FranchiseHandler) GetFranchise(c echo.Context) error {
	id := c.Param("id")
	f, err := h.fRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "franchise not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get franchise")
	}
	return c.JSON(http.StatusOK, f)
}

// ListSeasons returns all seasons for a specific franchise.
func (h *FranchiseHandler) ListSeasons(c echo.Context) error {
	id := c.Param("id")

	// Check if franchise exists
	_, err := h.fRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "franchise not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get franchise")
	}

	seasons, err := h.sRepo.ListByFranchiseID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list seasons")
	}
	if seasons == nil {
		seasons = []*domain.Season{}
	}
	return c.JSON(http.StatusOK, seasons)
}
