package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
)

type ExecutionRepositoryMongo struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewExecutionRepositoryMongo(client *mongo.Client, database *mongo.Database) *ExecutionRepositoryMongo {
	return &ExecutionRepositoryMongo{client: client, database: database}
}

func (r *ExecutionRepositoryMongo) withTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) (interface{}, error)) error {
	session, err := r.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, fn)
	return err
}

func (r *ExecutionRepositoryMongo) Create(ctx context.Context, exec *entities.Execution, outboxEvent *repositories.OutboxDoc) error {
	return r.withTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, err := r.database.Collection(ExecutionsCollection).InsertOne(sessCtx, exec); err != nil {
			return nil, err
		}
		if outboxEvent != nil {
			if err := insertOutbox(sessCtx, r.database, outboxEvent); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
}

func (r *ExecutionRepositoryMongo) UpdateStatus(ctx context.Context, osID string, next entities.ExecutionStatus, notes string, outboxEvent *repositories.OutboxDoc) (*entities.Execution, error) {
	var updated *entities.Execution
	err := r.withTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		var exec entities.Execution
		if err := r.database.Collection(ExecutionsCollection).FindOne(sessCtx, bson.M{"os_id": osID}).Decode(&exec); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, repositories.ErrNotFound
			}
			return nil, err
		}

		from := exec.Status
		if err := exec.TransitionTo(next); err != nil {
			return nil, err
		}
		if notes != "" {
			exec.Notes = notes
		}

		_, err := r.database.Collection(ExecutionsCollection).UpdateOne(sessCtx,
			bson.M{"os_id": osID},
			bson.M{"$set": bson.M{
				"status":       exec.Status,
				"notes":        exec.Notes,
				"completed_at": exec.CompletedAt,
				"updated_at":   exec.UpdatedAt,
			}},
		)
		if err != nil {
			return nil, err
		}

		history := entities.RepairHistory{
			ID:         uuid.New().String(),
			OSID:       osID,
			FromStatus: from,
			ToStatus:   exec.Status,
			Notes:      notes,
			At:         time.Now().UTC(),
		}
		if _, err := r.database.Collection(RepairHistoryCollection).InsertOne(sessCtx, history); err != nil {
			return nil, err
		}

		if outboxEvent != nil {
			if err := insertOutbox(sessCtx, r.database, outboxEvent); err != nil {
				return nil, err
			}
		}

		updated = &exec
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *ExecutionRepositoryMongo) FindByOSID(ctx context.Context, osID string) (*entities.Execution, error) {
	var exec entities.Execution
	if err := r.database.Collection(ExecutionsCollection).FindOne(ctx, bson.M{"os_id": osID}).Decode(&exec); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return &exec, nil
}

func (r *ExecutionRepositoryMongo) FetchUnpublishedOutbox(ctx context.Context, batch int) ([]repositories.OutboxDoc, error) {
	cur, err := r.database.Collection(OutboxCollection).Find(ctx,
		bson.M{"published_at": nil},
		newFindOptions(batch),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var docs []repositories.OutboxDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *ExecutionRepositoryMongo) MarkOutboxPublished(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now().UTC()
	_, err := r.database.Collection(OutboxCollection).UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"$set": bson.M{"published_at": now}},
	)
	return err
}

func (r *ExecutionRepositoryMongo) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	err := r.database.Collection(ProcessedEventsCollection).FindOne(ctx, bson.M{"event_id": eventID}).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}

func (r *ExecutionRepositoryMongo) MarkEventProcessed(ctx context.Context, eventID string) error {
	_, err := r.database.Collection(ProcessedEventsCollection).InsertOne(ctx, bson.M{
		"_id":          uuid.New().String(),
		"event_id":     eventID,
		"processed_at": time.Now().UTC(),
	})
	return err
}

func insertOutbox(ctx context.Context, database *mongo.Database, doc *repositories.OutboxDoc) error {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now().UTC()
	}
	_, err := database.Collection(OutboxCollection).InsertOne(ctx, doc)
	return err
}
