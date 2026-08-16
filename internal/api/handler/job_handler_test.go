package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"queenx/internal/ingestion"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockJobRepo struct {
	mock.Mock
}

func (m *mockJobRepo) Create(ctx context.Context, job *ingestion.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *mockJobRepo) GetByID(ctx context.Context, id string) (*ingestion.Job, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ingestion.Job), args.Error(1)
}

func TestJobHandler_CreateJob(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/ingest", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := new(mockJobRepo)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)
	h := &JobHandler{repo: repo}

	err := h.CreateJob(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":`)
}

func TestJobHandler_GetJob(t *testing.T) {
	e := echo.New()

	repo := new(mockJobRepo)
	h := &JobHandler{repo: repo}

	// Found
	repo.On("GetByID", mock.Anything, "job-1").Return(&ingestion.Job{
		ID:       "job-1",
		Status:   ingestion.StatusRunning,
		Progress: "Scraping season 1",
	}, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-1", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("job-1")

	err := h.GetJob(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Scraping season 1")

	// Not Found
	repo.On("GetByID", mock.Anything, "missing").Return(nil, ingestion.ErrJobNotFound).Once()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing", http.NoBody)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("missing")

	err = h.GetJob(c)
	require.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestJobHandler_GetJobProgressSSE(t *testing.T) {
	e := echo.New()

	repo := new(mockJobRepo)
	h := &JobHandler{repo: repo}

	// First call succeeds and returns running
	repo.On("GetByID", mock.Anything, "job-1").Return(&ingestion.Job{
		ID:       "job-1",
		Status:   ingestion.StatusRunning,
		Progress: "Streaming",
	}, nil).Once()

	// Subsequent calls return completed
	repo.On("GetByID", mock.Anything, "job-1").Return(&ingestion.Job{
		ID:       "job-1",
		Status:   ingestion.StatusCompleted,
		Progress: "Streaming",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("job-1")

	err := h.GetJobProgressSSE(c)
	require.NoError(t, err)

	body := rec.Body.String()
	assert.True(t, strings.Contains(body, `data: {"status":"completed","progress":"Streaming"}`))
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
}
