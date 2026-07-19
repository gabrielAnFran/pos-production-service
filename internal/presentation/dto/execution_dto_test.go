package dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
