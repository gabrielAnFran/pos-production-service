package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
)

// RunOutboxDispatcher polls repo for unpublished outbox docs and publishes
// them via conn, marking them published on success. It blocks until ctx is
// cancelled.
func RunOutboxDispatcher(ctx context.Context, conn *Conn, repo repositories.ExecutionRepository, interval time.Duration, batch int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dispatchOnce(ctx, conn, repo, batch)
		}
	}
}

func dispatchOnce(ctx context.Context, conn *Conn, repo repositories.ExecutionRepository, batch int) {
	docs, err := repo.FetchUnpublishedOutbox(ctx, batch)
	if err != nil {
		slog.Error("outbox: fetch unpublished failed", "error", err)
		return
	}
	if len(docs) == 0 {
		return
	}

	var publishedIDs []string
	for _, doc := range docs {
		var ev Event
		if err := json.Unmarshal(doc.Payload, &ev); err != nil {
			slog.Error("outbox: bad stored event payload, skipping", "outbox_id", doc.ID, "error", err)
			continue
		}
		if err := conn.Publish(ctx, ev); err != nil {
			slog.Warn("outbox: publish failed, will retry next tick", "outbox_id", doc.ID, "error", err)
			continue
		}
		publishedIDs = append(publishedIDs, doc.ID)
	}

	if len(publishedIDs) > 0 {
		if err := repo.MarkOutboxPublished(ctx, publishedIDs); err != nil {
			slog.Error("outbox: mark published failed", "error", err)
		}
	}
}
