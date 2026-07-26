//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/db"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/messaging"
)

// TestRunOutboxDispatcher_PublishesAndMarks exercises the outbox dispatcher
// against a real Mongo-backed repository and a real RabbitMQ connection: a
// pending outbox row should be published and then marked as published so it
// isn't redelivered.
func TestRunOutboxDispatcher_PublishesAndMarks(t *testing.T) {
	ctx := context.Background()
	client, database := setupMongo(t)
	repo := db.NewExecutionRepositoryMongo(client, database)
	conn := setupRabbit(t)

	svc := "prodtest-dispatcher"
	_, err := conn.DeclareServiceQueue(svc, []string{"DispatcherTestEvent"})
	require.NoError(t, err)

	now := time.Now().UTC()
	exec := &entities.Execution{
		ID:        "dispatch-exec-1",
		OSID:      "os-dispatch-1",
		BudgetID:  "budget-1",
		Status:    entities.StatusDiagnosing,
		CreatedAt: now,
		UpdatedAt: now,
	}
	ev, err := messaging.NewEvent("DispatcherTestEvent", "corr-1", "saga-1", map[string]string{"os_id": "os-dispatch-1"})
	require.NoError(t, err)
	payload, err := json.Marshal(ev)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, exec, &repositories.OutboxDoc{
		EventID:   ev.EventID,
		EventName: ev.EventName,
		Payload:   payload,
	}))

	pending, err := repo.FetchUnpublishedOutbox(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	dispatchCtx, dispatchCancel := context.WithCancel(ctx)
	go messaging.RunOutboxDispatcher(dispatchCtx, conn, repo, 200*time.Millisecond, 10)
	t.Cleanup(dispatchCancel)

	received := make(chan messaging.Event, 1)
	consumeCtx, consumeCancel := context.WithTimeout(ctx, 15*time.Second)
	defer consumeCancel()
	go func() {
		_ = conn.Consume(consumeCtx, svc, func(_ context.Context, got messaging.Event) error {
			received <- got
			consumeCancel()
			return nil
		})
	}()

	select {
	case got := <-received:
		assert.Equal(t, ev.EventID, got.EventID)
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for dispatched event")
	}

	require.Eventually(t, func() bool {
		remaining, err := repo.FetchUnpublishedOutbox(ctx, 10)
		return err == nil && len(remaining) == 0
	}, 5*time.Second, 200*time.Millisecond, "outbox row should be marked published")
}
