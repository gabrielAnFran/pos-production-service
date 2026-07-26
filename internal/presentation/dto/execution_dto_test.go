package dto

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
)

func TestFromEntity(t *testing.T) {
	now := time.Now().UTC()
	e := &entities.Execution{
		ID:        "e1",
		OSID:      "os-1",
		BudgetID:  "b-1",
		Status:    entities.StatusDiagnosing,
		CreatedAt: now,
		UpdatedAt: now,
	}
	got := FromEntity(e)
	assert.Equal(t, "e1", got.ID)
	assert.Equal(t, "os-1", got.OSID)
	assert.Equal(t, entities.StatusDiagnosing, got.Status)
}

func TestWriteProblem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteProblem(c, 404, "not found", "no execution")

	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var got Problem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "about:blank", got.Type)
	assert.Equal(t, "not found", got.Title)
	assert.Equal(t, 404, got.Status)
	assert.Equal(t, "no execution", got.Detail)
}
