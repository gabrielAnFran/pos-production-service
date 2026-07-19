package repositories

import (
	"context"
	"time"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
)

// OutboxDoc is a stored, not-yet-published domain event. Payload holds the
// marshaled messaging.Event so the dispatcher can unmarshal and publish it
// without the db package depending on messaging.
type OutboxDoc struct {
	ID          string     `bson:"_id"`
	EventID     string     `bson:"event_id"`
	EventName   string     `bson:"event_name"`
	Payload     []byte     `bson:"payload"`
	CreatedAt   time.Time  `bson:"created_at"`
	PublishedAt *time.Time `bson:"published_at"`
}

type ExecutionRepository interface {
	// Create inserts a new execution and its associated outbox event
	// transactionally. outboxEvent may be nil to skip emitting an event.
	Create(ctx context.Context, exec *entities.Execution, outboxEvent *OutboxDoc) error

	// UpdateStatus loads the execution by os_id, applies the transition,
	// persists the new state plus a repair_history row and (optionally) an
	// outbox event, all within a single transaction.
	UpdateStatus(ctx context.Context, osID string, next entities.ExecutionStatus, notes string, outboxEvent *OutboxDoc) (*entities.Execution, error)

	FindByOSID(ctx context.Context, osID string) (*entities.Execution, error)

	FetchUnpublishedOutbox(ctx context.Context, batch int) ([]OutboxDoc, error)
	MarkOutboxPublished(ctx context.Context, ids []string) error

	IsEventProcessed(ctx context.Context, eventID string) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID string) error
}
