package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gabrielAnFran/pos-production-service/internal/application/usecases"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/config"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/db"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/messaging"
)

const serviceName = "production-service"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, database, err := db.Connect(ctx, cfg.MongoURI)
	if err != nil {
		slog.Error("failed to connect to mongo", "error", err)
		os.Exit(1)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	conn, err := messaging.Dial(cfg.AMQPURL)
	if err != nil {
		slog.Error("failed to connect to amqp", "error", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.DeclareServiceQueue(serviceName, []string{"StartExecutionCommand"}); err != nil {
		slog.Error("failed to declare service queue", "error", err)
		os.Exit(1)
	}

	repo := db.NewExecutionRepositoryMongo(client, database)
	startExecution := usecases.NewStartExecutionUseCase(repo)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		messaging.RunOutboxDispatcher(ctx, conn, repo, time.Duration(cfg.DispatchIntervalMS)*time.Millisecond, 100)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := conn.Consume(ctx, serviceName, startExecution.Handle); err != nil && ctx.Err() == nil {
			slog.Error("consumer stopped unexpectedly", "error", err)
		}
	}()

	slog.Info("production-service worker started")
	<-ctx.Done()
	slog.Info("shutting down worker")
	wg.Wait()
}
