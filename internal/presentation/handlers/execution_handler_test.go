package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielAnFran/pos-production-service/internal/application/usecases"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
)

// fakeRepo is a minimal in-memory repositories.ExecutionRepository for
// handler-level HTTP tests.
type fakeRepo struct {
	byOSID map[string]*entities.Execution
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byOSID: map[string]*entities.Execution{}}
}

func (f *fakeRepo) Create(_ context.Context, exec *entities.Execution, _ *repositories.OutboxDoc) error {
	f.byOSID[exec.OSID] = exec
	return nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, osID string, next entities.ExecutionStatus, notes string, _ *repositories.OutboxDoc) (*entities.Execution, error) {
	exec, ok := f.byOSID[osID]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	if err := exec.TransitionTo(next); err != nil {
		return nil, err
	}
	exec.Notes = notes
	return exec, nil
}

func (f *fakeRepo) FindByOSID(_ context.Context, osID string) (*entities.Execution, error) {
	exec, ok := f.byOSID[osID]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return exec, nil
}

func (f *fakeRepo) FetchUnpublishedOutbox(context.Context, int) ([]repositories.OutboxDoc, error) {
	return nil, nil
}
func (f *fakeRepo) MarkOutboxPublished(context.Context, []string) error    { return nil }
func (f *fakeRepo) IsEventProcessed(context.Context, string) (bool, error) { return false, nil }
func (f *fakeRepo) MarkEventProcessed(context.Context, string) error       { return nil }

func newTestRouter(repo repositories.ExecutionRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	updater := usecases.NewUpdateExecutionStatusUseCase(repo)
	execHandler := NewExecutionHandler(repo, updater)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.GET("/executions/:os_id", execHandler.Get)
	v1.PATCH("/executions/:os_id", execHandler.UpdateStatus)
	return r
}

func TestGet_NotFound(t *testing.T) {
	repo := newFakeRepo()
	r := newTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/missing", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGet_Found(t *testing.T) {
	repo := newFakeRepo()
	now := time.Now().UTC()
	repo.byOSID["os-1"] = &entities.Execution{ID: "e1", OSID: "os-1", Status: entities.StatusDiagnosing, CreatedAt: now, UpdatedAt: now}
	r := newTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/os-1", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "os-1", body["os_id"])
}

func TestUpdateStatus_Success(t *testing.T) {
	repo := newFakeRepo()
	now := time.Now().UTC()
	repo.byOSID["os-1"] = &entities.Execution{ID: "e1", OSID: "os-1", Status: entities.StatusDiagnosing, CreatedAt: now, UpdatedAt: now}
	r := newTestRouter(repo)

	body, _ := json.Marshal(map[string]string{"status": "REPAIRING"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/executions/os-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateStatus_InvalidTransition(t *testing.T) {
	repo := newFakeRepo()
	now := time.Now().UTC()
	repo.byOSID["os-1"] = &entities.Execution{ID: "e1", OSID: "os-1", Status: entities.StatusDiagnosing, CreatedAt: now, UpdatedAt: now}
	r := newTestRouter(repo)

	body, _ := json.Marshal(map[string]string{"status": "COMPLETED"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/executions/os-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateStatus_NotFound(t *testing.T) {
	repo := newFakeRepo()
	r := newTestRouter(repo)

	body, _ := json.Marshal(map[string]string{"status": "REPAIRING"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/executions/missing", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateStatus_BadBody(t *testing.T) {
	repo := newFakeRepo()
	r := newTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/executions/os-1", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
