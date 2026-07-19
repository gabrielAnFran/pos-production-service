package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/messaging"
)

func TestStartExecution_Handle_CreatesExecutionAndOutboxEvent(t *testing.T) {
	repo := newFakeExecutionRepository()
	uc := NewStartExecutionUseCase(repo)

	ev, err := messaging.NewEvent("StartExecutionCommand", "corr-1", "saga-1", StartExecutionCommandPayload{
		OSID:     "os-123",
		BudgetID: "budget-456",
	})
	require.NoError(t, err)

	require.NoError(t, uc.Handle(context.Background(), ev))

	exec, err := repo.FindByOSID(context.Background(), "os-123")
	require.NoError(t, err)
	assert.Equal(t, entities.StatusDiagnosing, exec.Status)
	assert.Equal(t, "budget-456", exec.BudgetID)
	assert.Equal(t, startExecutionTechnicianPlaceholder, exec.TechnicianID)

	require.Len(t, repo.outbox, 1)
	assert.Equal(t, "ExecutionStarted", repo.outbox[0].EventName)

	processed, err := repo.IsEventProcessed(context.Background(), ev.EventID)
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestStartExecution_Handle_Idempotent(t *testing.T) {
	repo := newFakeExecutionRepository()
	uc := NewStartExecutionUseCase(repo)

	ev, err := messaging.NewEvent("StartExecutionCommand", "corr-1", "saga-1", StartExecutionCommandPayload{
		OSID:     "os-123",
		BudgetID: "budget-456",
	})
	require.NoError(t, err)

	require.NoError(t, uc.Handle(context.Background(), ev))
	require.NoError(t, uc.Handle(context.Background(), ev))

	require.Len(t, repo.outbox, 1, "second call should be a no-op due to idempotency check")
}

func TestStartExecution_Handle_BadPayload(t *testing.T) {
	repo := newFakeExecutionRepository()
	uc := NewStartExecutionUseCase(repo)

	ev := messaging.Event{EventID: "bad-1", EventName: "StartExecutionCommand", Payload: []byte("not-json")}
	err := uc.Handle(context.Background(), ev)
	assert.Error(t, err)
}
