package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/gabrielAnFran/pos-production-service/internal/application/usecases"
)

func TestNewRouter_Healthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeRepo()
	updater := usecases.NewUpdateExecutionStatusUseCase(repo)
	execHandler := NewExecutionHandler(repo, updater)
	healthHandler := NewHealthHandler(nil)

	r := NewRouter(execHandler, healthHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewRouter_ExecutionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeRepo()
	updater := usecases.NewUpdateExecutionStatusUseCase(repo)
	execHandler := NewExecutionHandler(repo, updater)
	healthHandler := NewHealthHandler(nil)

	r := NewRouter(execHandler, healthHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/missing", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
