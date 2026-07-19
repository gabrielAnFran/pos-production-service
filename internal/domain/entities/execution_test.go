package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransitionTo_Valid(t *testing.T) {
	cases := []struct {
		from, to ExecutionStatus
	}{
		{StatusDiagnosing, StatusRepairing},
		{StatusDiagnosing, StatusFailed},
		{StatusRepairing, StatusCompleted},
		{StatusRepairing, StatusFailed},
	}

	for _, tc := range cases {
		e := &Execution{Status: tc.from}
		err := e.TransitionTo(tc.to)
		require.NoError(t, err, "%s -> %s should be valid", tc.from, tc.to)
		assert.Equal(t, tc.to, e.Status)
	}
}

func TestTransitionTo_CompletedSetsCompletedAt(t *testing.T) {
	e := &Execution{Status: StatusRepairing}
	require.NoError(t, e.TransitionTo(StatusCompleted))
	require.NotNil(t, e.CompletedAt)
}

func TestTransitionTo_Invalid(t *testing.T) {
	cases := []struct {
		from, to ExecutionStatus
	}{
		{StatusDiagnosing, StatusCompleted},
		{StatusRepairing, StatusDiagnosing},
		{StatusCompleted, StatusRepairing},
		{StatusCompleted, StatusDiagnosing},
		{StatusCompleted, StatusFailed},
		{StatusFailed, StatusRepairing},
		{StatusFailed, StatusDiagnosing},
		{StatusFailed, StatusCompleted},
		{StatusDiagnosing, StatusDiagnosing},
	}

	for _, tc := range cases {
		e := &Execution{Status: tc.from}
		err := e.TransitionTo(tc.to)
		assert.ErrorIs(t, err, ErrInvalidTransition, "%s -> %s should be invalid", tc.from, tc.to)
		assert.Equal(t, tc.from, e.Status, "status must not change on invalid transition")
	}
}
