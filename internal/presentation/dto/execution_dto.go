package dto

import (
	"time"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
)

type UpdateStatusRequest struct {
	Status entities.ExecutionStatus `json:"status" binding:"required"`
	Notes  string                   `json:"notes"`
}

type ExecutionResponse struct {
	ID           string                   `json:"id"`
	OSID         string                   `json:"os_id"`
	BudgetID     string                   `json:"budget_id"`
	Status       entities.ExecutionStatus `json:"status"`
	TechnicianID string                   `json:"technician_id,omitempty"`
	Notes        string                   `json:"notes,omitempty"`
	StartedAt    *time.Time               `json:"started_at,omitempty"`
	CompletedAt  *time.Time               `json:"completed_at,omitempty"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

func FromEntity(e *entities.Execution) ExecutionResponse {
	return ExecutionResponse{
		ID:           e.ID,
		OSID:         e.OSID,
		BudgetID:     e.BudgetID,
		Status:       e.Status,
		TechnicianID: e.TechnicianID,
		Notes:        e.Notes,
		StartedAt:    e.StartedAt,
		CompletedAt:  e.CompletedAt,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
