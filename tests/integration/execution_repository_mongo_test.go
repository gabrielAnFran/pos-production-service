//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/db"
)

func seedExecution(t *testing.T, ctx context.Context, repo *db.ExecutionRepositoryMongo, osID string) {
	t.Helper()
	now := time.Now().UTC()
	exec := &entities.Execution{
		ID:        osID + "-id",
		OSID:      osID,
		BudgetID:  "budget-1",
		Status:    entities.StatusDiagnosing,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repo.Create(ctx, exec, nil))
}

func TestExecutionRepositoryMongo_UpdateStatus_ValidTransitions(t *testing.T) {
	ctx := context.Background()
	client, database := setupMongo(t)
	repo := db.NewExecutionRepositoryMongo(client, database)

	seedExecution(t, ctx, repo, "os-transitions-1")

	updated, err := repo.UpdateStatus(ctx, "os-transitions-1", entities.StatusRepairing, "starting repair", nil)
	require.NoError(t, err)
	assert.Equal(t, entities.StatusRepairing, updated.Status)
	assert.Nil(t, updated.CompletedAt)

	updated, err = repo.UpdateStatus(ctx, "os-transitions-1", entities.StatusCompleted, "done", &repositories.OutboxDoc{
		EventID:   "ev-1",
		EventName: "ExecutionCompleted",
		Payload:   []byte(`{}`),
	})
	require.NoError(t, err)
	assert.Equal(t, entities.StatusCompleted, updated.Status)
	require.NotNil(t, updated.CompletedAt)

	found, err := repo.FindByOSID(ctx, "os-transitions-1")
	require.NoError(t, err)
	assert.Equal(t, entities.StatusCompleted, found.Status)

	history, err := database.Collection(db.RepairHistoryCollection).CountDocuments(ctx, map[string]any{"os_id": "os-transitions-1"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), history)

	outbox, err := repo.FetchUnpublishedOutbox(ctx, 10)
	require.NoError(t, err)
	require.Len(t, outbox, 1)
	assert.Equal(t, "ExecutionCompleted", outbox[0].EventName)

	require.NoError(t, repo.MarkOutboxPublished(ctx, []string{outbox[0].ID}))
	remaining, err := repo.FetchUnpublishedOutbox(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestExecutionRepositoryMongo_UpdateStatus_Failed(t *testing.T) {
	ctx := context.Background()
	client, database := setupMongo(t)
	repo := db.NewExecutionRepositoryMongo(client, database)

	seedExecution(t, ctx, repo, "os-fail-1")

	updated, err := repo.UpdateStatus(ctx, "os-fail-1", entities.StatusFailed, "damaged beyond repair", nil)
	require.NoError(t, err)
	assert.Equal(t, entities.StatusFailed, updated.Status)
}

func TestExecutionRepositoryMongo_UpdateStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()
	client, database := setupMongo(t)
	repo := db.NewExecutionRepositoryMongo(client, database)

	seedExecution(t, ctx, repo, "os-invalid-1")

	_, err := repo.UpdateStatus(ctx, "os-invalid-1", entities.StatusCompleted, "", nil)
	require.ErrorIs(t, err, entities.ErrInvalidTransition)
}

func TestExecutionRepositoryMongo_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	client, database := setupMongo(t)
	repo := db.NewExecutionRepositoryMongo(client, database)

	_, err := repo.UpdateStatus(ctx, "does-not-exist", entities.StatusRepairing, "", nil)
	require.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestExecutionRepositoryMongo_FindByOSID_NotFound(t *testing.T) {
	ctx := context.Background()
	client, database := setupMongo(t)
	repo := db.NewExecutionRepositoryMongo(client, database)

	_, err := repo.FindByOSID(ctx, "missing")
	require.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestExecutionRepositoryMongo_EventProcessedIdempotency(t *testing.T) {
	ctx := context.Background()
	client, database := setupMongo(t)
	repo := db.NewExecutionRepositoryMongo(client, database)

	processed, err := repo.IsEventProcessed(ctx, "ev-idem-1")
	require.NoError(t, err)
	assert.False(t, processed)

	require.NoError(t, repo.MarkEventProcessed(ctx, "ev-idem-1"))

	processed, err = repo.IsEventProcessed(ctx, "ev-idem-1")
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestExecutionRepositoryMongo_Create_DuplicateOSID(t *testing.T) {
	ctx := context.Background()
	client, database := setupMongo(t)
	repo := db.NewExecutionRepositoryMongo(client, database)

	seedExecution(t, ctx, repo, "os-dup-1")

	now := time.Now().UTC()
	dup := &entities.Execution{ID: "other-id", OSID: "os-dup-1", Status: entities.StatusDiagnosing, CreatedAt: now, UpdatedAt: now}
	err := repo.Create(ctx, dup, nil)
	require.Error(t, err)
}
