package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
)

func seedExecution(t *testing.T, repo *fakeExecutionRepository, osID string, status entities.ExecutionStatus) {
	t.Helper()
	now := time.Now().UTC()
	require.NoError(t, repo.Create(context.Background(), &entities.Execution{
		ID:        "exec-" + osID,
		OSID:      osID,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil))
}

func TestUpdateExecutionStatus_ToRepairing_NoOutboxEvent(t *testing.T) {
	repo := newFakeExecutionRepository()
	seedExecution(t, repo, "os-1", entities.StatusDiagnosing)
	uc := NewUpdateExecutionStatusUseCase(repo)

	exec, err := uc.Handle(context.Background(), "os-1", entities.StatusRepairing, "", "corr-1")
	require.NoError(t, err)
	assert.Equal(t, entities.StatusRepairing, exec.Status)
	assert.Empty(t, repo.outbox, "REPAIRING is not in the event catalog")
}

func TestUpdateExecutionStatus_ToCompleted_EmitsExecutionCompleted(t *testing.T) {
	repo := newFakeExecutionRepository()
	seedExecution(t, repo, "os-1", entities.StatusRepairing)
	uc := NewUpdateExecutionStatusUseCase(repo)

	exec, err := uc.Handle(context.Background(), "os-1", entities.StatusCompleted, "fixed brakes", "corr-1")
	require.NoError(t, err)
	assert.Equal(t, entities.StatusCompleted, exec.Status)
	require.Len(t, repo.outbox, 1)
	assert.Equal(t, "ExecutionCompleted", repo.outbox[0].EventName)
}

func TestUpdateExecutionStatus_ToFailed_EmitsExecutionFailed(t *testing.T) {
	repo := newFakeExecutionRepository()
	seedExecution(t, repo, "os-1", entities.StatusDiagnosing)
	uc := NewUpdateExecutionStatusUseCase(repo)

	exec, err := uc.Handle(context.Background(), "os-1", entities.StatusFailed, "unrepairable", "corr-1")
	require.NoError(t, err)
	assert.Equal(t, entities.StatusFailed, exec.Status)
	require.Len(t, repo.outbox, 1)
	assert.Equal(t, "ExecutionFailed", repo.outbox[0].EventName)
}

func TestUpdateExecutionStatus_InvalidTransition(t *testing.T) {
	repo := newFakeExecutionRepository()
	seedExecution(t, repo, "os-1", entities.StatusDiagnosing)
	uc := NewUpdateExecutionStatusUseCase(repo)

	_, err := uc.Handle(context.Background(), "os-1", entities.StatusCompleted, "", "corr-1")
	assert.ErrorIs(t, err, entities.ErrInvalidTransition)
	assert.Empty(t, repo.outbox, "no event should be staged when the transition itself fails")
}

func TestUpdateExecutionStatus_NotFound(t *testing.T) {
	repo := newFakeExecutionRepository()
	uc := NewUpdateExecutionStatusUseCase(repo)

	_, err := uc.Handle(context.Background(), "missing-os", entities.StatusRepairing, "", "corr-1")
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}
