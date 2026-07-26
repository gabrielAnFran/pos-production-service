//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/db"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/messaging"
)

// setupMongo spins up a single-node Mongo replica set and returns a
// connected client and database. Transactions require the replica set even
// with a single member, and a direct connection is required because
// rs.initiate() advertises the container's internal Docker IP, which is not
// routable from the test host.
func setupMongo(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()
	ctx := context.Background()

	mongoC, err := mongodb.Run(ctx, "mongo:7", mongodb.WithReplicaSet("rs0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = mongoC.Terminate(context.Background()) })

	uri, err := mongoC.ConnectionString(ctx)
	require.NoError(t, err)

	client, database, err := db.Connect(ctx, uri+"&directConnection=true")
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	return client, database
}

// setupRabbit spins up RabbitMQ and returns a dialed messaging.Conn.
func setupRabbit(t *testing.T) *messaging.Conn {
	t.Helper()
	ctx := context.Background()

	rabbitC, err := rabbitmq.Run(ctx, "rabbitmq:3.13-management-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rabbitC.Terminate(context.Background()) })

	url, err := rabbitC.AmqpURL(ctx)
	require.NoError(t, err)

	var conn *messaging.Conn
	require.Eventually(t, func() bool {
		conn, err = messaging.Dial(url)
		return err == nil
	}, 30*time.Second, 500*time.Millisecond)
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}
