package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/messaging"
)

type ExecutionCompletedPayload struct {
	OSID        string    `json:"os_id"`
	CompletedAt time.Time `json:"completed_at"`
	RepairNotes string    `json:"repair_notes"`
}

type ExecutionFailedPayload struct {
	OSID   string `json:"os_id"`
	Reason string `json:"reason"`
}

type UpdateExecutionStatusUseCase struct {
	repo repositories.ExecutionRepository
}

func NewUpdateExecutionStatusUseCase(repo repositories.ExecutionRepository) *UpdateExecutionStatusUseCase {
	return &UpdateExecutionStatusUseCase{repo: repo}
}

// Handle transitions the execution identified by osID to next, recording
// repair_history and, for COMPLETED/FAILED, an outbox event. REPAIRING is an
// internal transition not present in the event catalog, so no event is
// emitted for it.
func (uc *UpdateExecutionStatusUseCase) Handle(ctx context.Context, osID string, next entities.ExecutionStatus, notes string, correlationID string) (*entities.Execution, error) {
	var outboxDoc *repositories.OutboxDoc

	switch next {
	case entities.StatusCompleted:
		ev, err := messaging.NewEvent("ExecutionCompleted", correlationID, "", ExecutionCompletedPayload{
			OSID:        osID,
			CompletedAt: time.Now().UTC(),
			RepairNotes: notes,
		})
		if err != nil {
			return nil, fmt.Errorf("build ExecutionCompleted event: %w", err)
		}
		outboxDoc, err = toOutboxDoc(ev)
		if err != nil {
			return nil, err
		}
	case entities.StatusFailed:
		ev, err := messaging.NewEvent("ExecutionFailed", correlationID, "", ExecutionFailedPayload{
			OSID:   osID,
			Reason: notes,
		})
		if err != nil {
			return nil, fmt.Errorf("build ExecutionFailed event: %w", err)
		}
		outboxDoc, err = toOutboxDoc(ev)
		if err != nil {
			return nil, err
		}
	}

	exec, err := uc.repo.UpdateStatus(ctx, osID, next, notes, outboxDoc)
	if err != nil {
		return nil, err
	}
	return exec, nil
}

func toOutboxDoc(ev messaging.Event) (*repositories.OutboxDoc, error) {
	b, err := json.Marshal(ev)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}
	return &repositories.OutboxDoc{
		EventID:   ev.EventID,
		EventName: ev.EventName,
		Payload:   b,
	}, nil
}
