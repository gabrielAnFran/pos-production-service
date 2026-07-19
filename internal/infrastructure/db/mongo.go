package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

const (
	ExecutionsCollection      = "execution_queue"
	RepairHistoryCollection   = "repair_history"
	OutboxCollection          = "outbox"
	ProcessedEventsCollection = "processed_events"

	defaultDBName = "production"
)

// Connect dials Mongo and ensures the indexes required by this service exist.
// Returns the client and the resolved database handle.
func Connect(ctx context.Context, uri string) (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, nil, err
	}
	database := client.Database(dbName(uri))
	if err := ensureIndexes(ctx, database); err != nil {
		return nil, nil, err
	}
	return client, database, nil
}

func ensureIndexes(ctx context.Context, database *mongo.Database) error {
	execIdx := database.Collection(ExecutionsCollection).Indexes()
	if _, err := execIdx.CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "os_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}}},
	}); err != nil {
		return err
	}

	outboxIdx := database.Collection(OutboxCollection).Indexes()
	if _, err := outboxIdx.CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "created_at", Value: 1}}},
		{Keys: bson.D{{Key: "event_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	}); err != nil {
		return err
	}

	processedIdx := database.Collection(ProcessedEventsCollection).Indexes()
	if _, err := processedIdx.CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "event_id", Value: 1}}, Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	return nil
}

// dbName extracts the database name embedded in the connection URI, falling
// back to a fixed default when the URI has none (e.g. no path segment).
func dbName(uri string) string {
	cs, err := connstring.Parse(uri)
	if err != nil || cs.Database == "" {
		return defaultDBName
	}
	return cs.Database
}
