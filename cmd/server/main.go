package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gabrielAnFran/pos-production-service/internal/application/usecases"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/config"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/db"
	"github.com/gabrielAnFran/pos-production-service/internal/presentation/handlers"
)

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

	repo := db.NewExecutionRepositoryMongo(client, database)
	updater := usecases.NewUpdateExecutionStatusUseCase(repo)

	execHandler := handlers.NewExecutionHandler(repo, updater)
	healthHandler := handlers.NewHealthHandler(client)
	router := handlers.NewRouter(execHandler, healthHandler)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		slog.Info("production-service server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
}
