//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

	"github.com/gabrielAnFran/pos-production-service/internal/application/usecases"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/db"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/messaging"
)

// TestStartExecution_EndToEnd exercises the outbox-pattern write path against
// a real, single-node Mongo replica set (transactions require a replica set
// even with a single member): insert a StartExecutionCommand event -> assert
// an execution_queue doc and an outbox doc were both written transactionally.
func TestStartExecution_EndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	mongoC, err := mongodb.Run(ctx, "mongo:7", mongodb.WithReplicaSet("rs0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = mongoC.Terminate(ctx) })

	uri, err := mongoC.ConnectionString(ctx)
	require.NoError(t, err)

	client, database, err := db.Connect(ctx, uri+"?replicaSet=rs0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	repo := db.NewExecutionRepositoryMongo(client, database)
	uc := usecases.NewStartExecutionUseCase(repo)

	ev, err := messaging.NewEvent("StartExecutionCommand", "corr-1", "saga-1", usecases.StartExecutionCommandPayload{
		OSID:     "os-e2e-1",
		BudgetID: "budget-e2e-1",
	})
	require.NoError(t, err)

	require.NoError(t, uc.Handle(ctx, ev))

	exec, err := repo.FindByOSID(ctx, "os-e2e-1")
	require.NoError(t, err)
	require.Equal(t, entities.StatusDiagnosing, exec.Status)
	require.Equal(t, "budget-e2e-1", exec.BudgetID)

	outbox, err := repo.FetchUnpublishedOutbox(ctx, 10)
	require.NoError(t, err)
	require.Len(t, outbox, 1)
	require.Equal(t, "ExecutionStarted", outbox[0].EventName)
}
