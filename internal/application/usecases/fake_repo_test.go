package usecases

import (
	"context"
	"errors"
	"sync"

	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
)

var errAlreadyExists = errors.New("execution already exists")

// fakeExecutionRepository is a hand-rolled in-memory implementation of
// repositories.ExecutionRepository for use-case unit tests.
type fakeExecutionRepository struct {
	mu              sync.Mutex
	byOSID          map[string]*entities.Execution
	outbox          []repositories.OutboxDoc
	processedEvents map[string]bool

	forceCreateErr error
	forceUpdateErr error
}

func newFakeExecutionRepository() *fakeExecutionRepository {
	return &fakeExecutionRepository{
		byOSID:          map[string]*entities.Execution{},
		processedEvents: map[string]bool{},
	}
}

func (f *fakeExecutionRepository) Create(_ context.Context, exec *entities.Execution, outboxEvent *repositories.OutboxDoc) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.forceCreateErr != nil {
		return f.forceCreateErr
	}
	if _, exists := f.byOSID[exec.OSID]; exists {
		return errAlreadyExists
	}
	cp := *exec
	f.byOSID[exec.OSID] = &cp
	if outboxEvent != nil {
		f.outbox = append(f.outbox, *outboxEvent)
	}
	return nil
}

func (f *fakeExecutionRepository) UpdateStatus(_ context.Context, osID string, next entities.ExecutionStatus, notes string, outboxEvent *repositories.OutboxDoc) (*entities.Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.forceUpdateErr != nil {
		return nil, f.forceUpdateErr
	}
	exec, ok := f.byOSID[osID]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	if err := exec.TransitionTo(next); err != nil {
		return nil, err
	}
	if notes != "" {
		exec.Notes = notes
	}
	if outboxEvent != nil {
		f.outbox = append(f.outbox, *outboxEvent)
	}
	cp := *exec
	return &cp, nil
}

func (f *fakeExecutionRepository) FindByOSID(_ context.Context, osID string) (*entities.Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	exec, ok := f.byOSID[osID]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	cp := *exec
	return &cp, nil
}

func (f *fakeExecutionRepository) FetchUnpublishedOutbox(_ context.Context, batch int) ([]repositories.OutboxDoc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if batch > len(f.outbox) {
		batch = len(f.outbox)
	}
	return append([]repositories.OutboxDoc{}, f.outbox[:batch]...), nil
}

func (f *fakeExecutionRepository) MarkOutboxPublished(_ context.Context, ids []string) error {
	return nil
}

func (f *fakeExecutionRepository) IsEventProcessed(_ context.Context, eventID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.processedEvents[eventID], nil
}

func (f *fakeExecutionRepository) MarkEventProcessed(_ context.Context, eventID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.processedEvents[eventID] = true
	return nil
}
