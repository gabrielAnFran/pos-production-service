package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
	"github.com/gabrielAnFran/pos-production-service/internal/infrastructure/messaging"
)

// startExecutionTechnicianPlaceholder is used because this challenge scope
// has no technician-assignment system yet.
const startExecutionTechnicianPlaceholder = "auto-assigned"

type StartExecutionCommandPayload struct {
	OSID     string `json:"os_id"`
	BudgetID string `json:"budget_id"`
}

type ExecutionStartedPayload struct {
	OSID         string    `json:"os_id"`
	StartedAt    time.Time `json:"started_at"`
	TechnicianID string    `json:"technician_id"`
}

type StartExecutionUseCase struct {
	repo repositories.ExecutionRepository
}

func NewStartExecutionUseCase(repo repositories.ExecutionRepository) *StartExecutionUseCase {
	return &StartExecutionUseCase{repo: repo}
}

// Handle processes a StartExecutionCommand event, idempotently.
func (uc *StartExecutionUseCase) Handle(ctx context.Context, ev messaging.Event) error {
	processed, err := uc.repo.IsEventProcessed(ctx, ev.EventID)
	if err != nil {
		return fmt.Errorf("check idempotency: %w", err)
	}
	if processed {
		return nil
	}

	var cmd StartExecutionCommandPayload
	if err := json.Unmarshal(ev.Payload, &cmd); err != nil {
		return fmt.Errorf("unmarshal StartExecutionCommand: %w", err)
	}

	now := time.Now().UTC()
	exec := &entities.Execution{
		ID:           uuid.New().String(),
		OSID:         cmd.OSID,
		BudgetID:     cmd.BudgetID,
		Status:       entities.StatusDiagnosing,
		TechnicianID: startExecutionTechnicianPlaceholder,
		StartedAt:    &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	outEvent, err := messaging.NewEvent("ExecutionStarted", ev.CorrelationID, ev.SagaID, ExecutionStartedPayload{
		OSID:         cmd.OSID,
		StartedAt:    now,
		TechnicianID: startExecutionTechnicianPlaceholder,
	})
	if err != nil {
		return fmt.Errorf("build ExecutionStarted event: %w", err)
	}
	outBytes, err := json.Marshal(outEvent)
	if err != nil {
		return fmt.Errorf("marshal ExecutionStarted event: %w", err)
	}
	outboxDoc := &repositories.OutboxDoc{
		EventID:   outEvent.EventID,
		EventName: outEvent.EventName,
		Payload:   outBytes,
	}

	if err := uc.repo.Create(ctx, exec, outboxDoc); err != nil {
		return fmt.Errorf("create execution: %w", err)
	}

	if err := uc.repo.MarkEventProcessed(ctx, ev.EventID); err != nil {
		return fmt.Errorf("mark event processed: %w", err)
	}

	return nil
}
