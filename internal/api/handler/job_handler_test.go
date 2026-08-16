package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"queenx/internal/ingestion"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockJobRepo struct {
	jobs map[string]*ingestion.Job
}

func (m *mockJobRepo) Create(ctx context.Context, job *ingestion.Job) error {
	m.jobs[job.ID] = job
	return nil
}

func (m *mockJobRepo) GetByID(ctx context.Context, id string) (*ingestion.Job, error) {
	if job, ok := m.jobs[id]; ok {
		return job, nil
	}
	return nil, ingestion.ErrJobNotFound
}

func (m *mockJobRepo) Update(ctx context.Context, job *ingestion.Job) error {
	m.jobs[job.ID] = job
	return nil
}

func (m *mockJobRepo) ClaimNextJob(ctx context.Context, leaseDuration time.Duration) (*ingestion.Job, error) {
	return nil, ingestion.ErrNoJobsAvailable
}

func TestJobHandler_CreateJob(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/ingest", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockJobRepo{jobs: make(map[string]*ingestion.Job)}
	h := NewJobHandler(repo)

	err := h.CreateJob(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Should return ID
	assert.Contains(t, rec.Body.String(), `"id":`)

	// Job should be in repo
	assert.Equal(t, 1, len(repo.jobs))
	for _, v := range repo.jobs {
		assert.Equal(t, ingestion.StatusPending, v.Status)
	}
}

func TestJobHandler_GetJob(t *testing.T) {
	e := echo.New()

	repo := &mockJobRepo{jobs: make(map[string]*ingestion.Job)}
	h := NewJobHandler(repo)

	// Insert mock job
	repo.jobs["job-1"] = &ingestion.Job{
		ID:       "job-1",
		Status:   ingestion.StatusRunning,
		Progress: "Scraping season 1",
	}

	// Found
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

	repo := &mockJobRepo{jobs: make(map[string]*ingestion.Job)}
	h := NewJobHandler(repo)

	repo.jobs["job-1"] = &ingestion.Job{
		ID:       "job-1",
		Status:   ingestion.StatusRunning,
		Progress: "Streaming",
	}

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("job-1")

	// Run in background because SSE blocks
	go func() {
		time.Sleep(100 * time.Millisecond) // Fast sleep to avoid flakes
		repo.jobs["job-1"].Status = ingestion.StatusCompleted
	}()

	err := h.GetJobProgressSSE(c)
	require.NoError(t, err)

	body := rec.Body.String()
	assert.True(t, strings.Contains(body, `data: {"status":"completed","progress":"Streaming"}`))
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
}
