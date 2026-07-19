package entities

import (
	"errors"
	"time"
)

type ExecutionStatus string

const (
	StatusDiagnosing ExecutionStatus = "DIAGNOSING"
	StatusRepairing  ExecutionStatus = "REPAIRING"
	StatusCompleted  ExecutionStatus = "COMPLETED"
	StatusFailed     ExecutionStatus = "FAILED"
)

var ErrInvalidTransition = errors.New("invalid execution status transition")

// validTransitions maps a current status to the set of statuses it may move to.
var validTransitions = map[ExecutionStatus]map[ExecutionStatus]bool{
	StatusDiagnosing: {StatusRepairing: true, StatusFailed: true},
	StatusRepairing:  {StatusCompleted: true, StatusFailed: true},
}

type Execution struct {
	ID           string          `bson:"_id"`
	OSID         string          `bson:"os_id"`
	BudgetID     string          `bson:"budget_id"`
	Status       ExecutionStatus `bson:"status"`
	TechnicianID string          `bson:"technician_id,omitempty"`
	Notes        string          `bson:"notes,omitempty"`
	StartedAt    *time.Time      `bson:"started_at,omitempty"`
	CompletedAt  *time.Time      `bson:"completed_at,omitempty"`
	CreatedAt    time.Time       `bson:"created_at"`
	UpdatedAt    time.Time       `bson:"updated_at"`
}

// TransitionTo validates and applies a status transition, mutating the
// execution's status (and timestamps where relevant). Callers are
// responsible for persisting the result and recording repair_history.
func (e *Execution) TransitionTo(next ExecutionStatus) error {
	allowed, ok := validTransitions[e.Status]
	if !ok || !allowed[next] {
		return ErrInvalidTransition
	}
	e.Status = next
	now := time.Now().UTC()
	e.UpdatedAt = now
	if next == StatusCompleted {
		e.CompletedAt = &now
	}
	return nil
}
