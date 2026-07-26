//go:build integration

package integration

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/messaging"
)

func TestConn_PublishConsume_Roundtrip(t *testing.T) {
	conn := setupRabbit(t)

	q, err := conn.DeclareServiceQueue("prodtest-roundtrip", []string{"widget.created"})
	require.NoError(t, err)
	assert.Equal(t, "prodtest-roundtrip.events.q", q)

	ev, err := messaging.NewEvent("widget.created", "corr-1", "saga-1", map[string]string{"foo": "bar"})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	require.NoError(t, conn.Publish(ctx, ev))

	received := make(chan messaging.Event, 1)
	consumeCtx, consumeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer consumeCancel()

	go func() {
		_ = conn.Consume(consumeCtx, "prodtest-roundtrip", func(_ context.Context, got messaging.Event) error {
			received <- got
			consumeCancel()
			return nil
		})
	}()

	select {
	case got := <-received:
		assert.Equal(t, ev.EventID, got.EventID)
		assert.Equal(t, "widget.created", got.EventName)
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for consumed message")
	}
}

func TestConn_Retry_EventuallyDeadLetters(t *testing.T) {
	conn := setupRabbit(t)

	svc := "prodtest-retry"
	_, err := conn.DeclareServiceQueue(svc, []string{"widget.failing"})
	require.NoError(t, err)

	ev, err := messaging.NewEvent("widget.failing", "corr-2", "saga-2", map[string]string{"foo": "baz"})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	require.NoError(t, conn.Publish(ctx, ev))

	var attempts int32
	handlerErr := errors.New("simulated handler failure")

	// The retry queue has a 30s TTL before dead-lettering back to the main
	// exchange, so allow enough wall-clock time for MaxRetries+1 round trips.
	consumeCtx, consumeCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer consumeCancel()

	go func() {
		_ = conn.Consume(consumeCtx, svc, func(_ context.Context, got messaging.Event) error {
			if got.EventID != ev.EventID {
				return nil
			}
			atomic.AddInt32(&attempts, 1)
			return handlerErr
		})
	}()

	dlqCh := conn.Channel()
	deadline := time.Now().Add(4 * time.Minute)
	var msg amqp.Delivery
	var found bool
	for time.Now().Before(deadline) {
		var ok bool
		var err error
		msg, ok, err = dlqCh.Get(svc+".events.dlq", false)
		require.NoError(t, err)
		if ok {
			found = true
			break
		}
		time.Sleep(2 * time.Second)
	}
	consumeCancel()

	require.True(t, found, "expected a message parked in the DLQ after %d attempts", atomic.LoadInt32(&attempts))
	assert.Contains(t, string(msg.Body), ev.EventID)
	_ = msg.Ack(false)
	assert.Equal(t, int32(messaging.MaxRetries+1), atomic.LoadInt32(&attempts))
}
