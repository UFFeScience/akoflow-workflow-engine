package lifecycle

import (
	"context"
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/stretchr/testify/require"
)

type lifecycleStub struct {
	phase    LifecyclePhase
	duration float64
	err      error
	calls    *int
}

func (s lifecycleStub) Phase() LifecyclePhase { return s.phase }
func (s lifecycleStub) Execute(context.Context, domain.ActivityExecutionContext) (LifecycleResult, error) {
	*s.calls++
	return LifecycleResult{DurationSeconds: s.duration}, s.err
}

func TestLifecyclePipelineRunsServicesInOrderAndSumsDuration(t *testing.T) {
	calls := 0
	pipeline := NewLifecyclePipeline(
		lifecycleStub{phase: PhaseCreateUser, duration: 1, calls: &calls},
		lifecycleStub{phase: PhaseCreateEnvironment, duration: 2, calls: &calls},
	)
	duration, err := pipeline.Execute(context.Background(), domain.ActivityExecutionContext{})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, 3.0, duration)
}

func TestLifecyclePipelineStopsOnFailure(t *testing.T) {
	calls := 0
	expected := errors.New("cannot create environment")
	pipeline := NewLifecyclePipeline(
		lifecycleStub{phase: PhaseCreateUser, duration: 1, calls: &calls},
		lifecycleStub{phase: PhaseCreateEnvironment, err: expected, calls: &calls},
		lifecycleStub{phase: PhaseExecuteActivity, calls: &calls},
	)
	duration, err := pipeline.Execute(context.Background(), domain.ActivityExecutionContext{})
	require.ErrorIs(t, err, expected)
	require.Equal(t, 2, calls)
	require.Equal(t, 1.0, duration)
}
