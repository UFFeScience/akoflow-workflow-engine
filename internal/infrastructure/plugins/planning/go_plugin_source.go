package planning

import (
	"context"
	"fmt"
	"path/filepath"
	"plugin"

	"github.com/UFFeScience/akoflow/internal/domain"
)

// GoPlugin loads the planning contract. A plugin must export either a
// `Planner` value implementing Plugin or a `Plan` function with the exact
// signature below.
type GoPlugin struct{ Path string }

func (p GoPlugin) Plan(ctx context.Context, request domain.PlanningRequest) (domain.SchedulePlan, error) {
	loaded, err := plugin.Open(filepath.Clean(p.Path))
	if err != nil {
		return domain.SchedulePlan{}, err
	}
	if symbol, err := loaded.Lookup("Planner"); err == nil {
		planner, ok := symbol.(Plugin)
		if !ok {
			return domain.SchedulePlan{}, fmt.Errorf("plugin Planner does not implement planning.Plugin")
		}
		return planner.Plan(ctx, request)
	}
	symbol, err := loaded.Lookup("Plan")
	if err != nil {
		return domain.SchedulePlan{}, fmt.Errorf("plugin exports neither Planner nor Plan: %w", err)
	}
	plan, ok := symbol.(func(context.Context, domain.PlanningRequest) (domain.SchedulePlan, error))
	if !ok {
		return domain.SchedulePlan{}, fmt.Errorf("plugin Plan has an incompatible signature")
	}
	return plan(ctx, request)
}
