package lifecycle

import (
	"context"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type LifecyclePhase string

const (
	PhaseCreateUser         LifecyclePhase = "create_user"
	PhaseCreateEnvironment  LifecyclePhase = "create_environment"
	PhasePrepareData        LifecyclePhase = "prepare_data"
	PhaseExecuteActivity    LifecyclePhase = "execute_activity"
	PhaseCollectResult      LifecyclePhase = "collect_result"
	PhaseCleanupEnvironment LifecyclePhase = "cleanup_environment"
)

type LifecycleResult struct {
	DurationSeconds float64
	Metadata        map[string]any
}

// ActivityLifecycle preserves the AkoFlow lifecycle boundary: every
// lifecycle operation explicitly receives the activity and its execution
// assignment rather than keeping mutable service state.
type ActivityLifecycle interface {
	Phase() LifecyclePhase
	Execute(context.Context, domain.ActivityExecutionContext) (LifecycleResult, error)
}

type LifecyclePipeline struct {
	steps []ActivityLifecycle
}

func NewLifecyclePipeline(steps ...ActivityLifecycle) LifecyclePipeline {
	return LifecyclePipeline{steps: steps}
}

func (p LifecyclePipeline) Execute(ctx context.Context, executionContext domain.ActivityExecutionContext) (float64, error) {
	var duration float64
	for _, step := range p.steps {
		result, err := step.Execute(ctx, executionContext)
		if err != nil {
			return duration, err
		}
		duration += result.DurationSeconds
	}
	return duration, nil
}
