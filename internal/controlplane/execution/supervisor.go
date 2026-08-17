package execution

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type ActivityController interface {
	Start(context.Context, domain.ActivityExecutionContext) (domain.ActivityHandle, error)
	Inspect(context.Context, string, domain.ExecutionMode) (*domain.ActivityHandle, error)
}

type Config struct {
	PollInterval time.Duration
	MaxParallel  int
}

type Supervisor struct {
	executions ports.ExecutionStore
	activities ActivityController
	simulation ports.PlanExecutor
	config     Config
}

func New(executions ports.ExecutionStore, activities ActivityController, simulation ports.PlanExecutor, config Config) (*Supervisor, error) {
	if executions == nil || activities == nil || simulation == nil {
		return nil, fmt.Errorf("execution repository, activity controller and simulator are required")
	}
	if config.PollInterval <= 0 {
		config.PollInterval = time.Second
	}
	if config.MaxParallel <= 0 {
		config.MaxParallel = 8
	}
	return &Supervisor{executions: executions, activities: activities, simulation: simulation, config: config}, nil
}

func (s *Supervisor) Execute(ctx context.Context, request ports.ExecutionRequest) (trace domain.ExecutionTrace, err error) {
	request.Run.SchedulePlanID = request.Plan.ID
	request.Run.Status = domain.ExecutionRunRunning
	if err := validateRequest(request); err != nil {
		return domain.ExecutionTrace{}, err
	}
	if err := s.executions.CreateRun(ctx, request.Run); err != nil {
		return domain.ExecutionTrace{}, fmt.Errorf("create execution run: %w", err)
	}
	defer func() {
		if err != nil {
			_ = s.executions.FailRun(context.WithoutCancel(ctx), request.Run.ID, err.Error())
		}
	}()
	if request.Run.Mode == domain.ExecutionModeSimulation {
		trace, err = s.simulation.Execute(ctx, request)
	} else {
		trace, err = s.executeActivities(ctx, request)
	}
	if err != nil {
		return domain.ExecutionTrace{}, err
	}
	if request.Run.Mode == domain.ExecutionModeInteractive {
		return trace, nil
	}
	if err = s.executions.CompleteRun(ctx, trace); err != nil {
		return domain.ExecutionTrace{}, fmt.Errorf("complete execution run: %w", err)
	}
	return trace, nil
}

func (s *Supervisor) executeActivities(ctx context.Context, request ports.ExecutionRequest) (domain.ExecutionTrace, error) {
	activities := indexActivities(request.Workflow.Activities)
	resources := indexResources(request.Resources)
	assignments := indexAssignments(request.Plan.Assignments)
	predecessors := make(map[string][]string)
	for _, dependency := range request.Workflow.Dependencies {
		predecessors[dependency.ActivityID] = append(predecessors[dependency.ActivityID], dependency.DependsOnActivityID)
	}
	completed := make(map[string]domain.TaskExecution)
	running := make(map[string]domain.ActivityHandle)
	tasks := make(map[string]domain.TaskExecution)

	for len(completed) < len(activities) {
		if err := ctx.Err(); err != nil {
			return domain.ExecutionTrace{}, err
		}
		if err := s.inspectRunning(ctx, request.Run.Mode, running, completed, tasks); err != nil {
			return domain.ExecutionTrace{}, err
		}
		if request.Run.Mode == domain.ExecutionModeInteractive && len(running) > 0 {
			return runningTrace(request, tasks), nil
		}
		ready := readyActivities(activities, predecessors, completed, running, tasks)
		available := s.config.MaxParallel - len(running)
		if available > len(ready) {
			available = len(ready)
		}
		if err := s.startReadyActivities(
			ctx, request, ready[:available], activities, assignments, resources,
			running, completed, tasks,
		); err != nil {
			return domain.ExecutionTrace{}, err
		}
		if len(running) == 0 && len(ready) == 0 && len(completed) < len(activities) {
			return domain.ExecutionTrace{}, fmt.Errorf("execution cannot progress: dependency cycle or incomplete plan")
		}
		if len(running) > 0 {
			select {
			case <-ctx.Done():
				return domain.ExecutionTrace{}, ctx.Err()
			case <-time.After(s.config.PollInterval):
			}
		}
	}
	return completedTrace(request, tasks), nil
}

func (s *Supervisor) inspectRunning(
	ctx context.Context,
	mode domain.ExecutionMode,
	running map[string]domain.ActivityHandle,
	completed map[string]domain.TaskExecution,
	tasks map[string]domain.TaskExecution,
) error {
	for activityID, handle := range running {
		observed, err := s.activities.Inspect(ctx, handle.ID, mode)
		if err != nil {
			return fmt.Errorf("inspect activity %q: %w", activityID, err)
		}
		task := tasks[activityID]
		switch observed.Status {
		case domain.HandleCompleted:
			completeTask(&task, *observed)
			if err := s.executions.SaveTask(ctx, task); err != nil {
				return err
			}
			tasks[activityID], completed[activityID] = task, task
			delete(running, activityID)
		case domain.HandleFailed, domain.HandleStopped:
			task.Status, task.FailureReason = domain.TaskFailed, observed.Failure
			_ = s.executions.SaveTask(ctx, task)
			return fmt.Errorf("activity %q failed: %s", activityID, observed.Failure)
		}
	}
	return nil
}

func (s *Supervisor) startReadyActivities(
	ctx context.Context,
	request ports.ExecutionRequest,
	ready []string,
	activities map[string]domain.Activity,
	assignments map[string]domain.PlanAssignment,
	resources map[string]domain.Resource,
	running map[string]domain.ActivityHandle,
	completed map[string]domain.TaskExecution,
	tasks map[string]domain.TaskExecution,
) error {
	for _, activityID := range ready {
		activity, assignment := activities[activityID], assignments[activityID]
		resource, ok := resources[assignment.ResourceID]
		if !ok {
			return fmt.Errorf("resource %q not found", assignment.ResourceID)
		}
		handle, err := s.activities.Start(ctx, domain.ActivityExecutionContext{
			Run: request.Run, Workflow: request.Workflow, Activity: activity,
			Assignment: assignment, Resource: resource,
		})
		if err != nil {
			return fmt.Errorf("start activity %q: %w", activityID, err)
		}
		task := newRunningTask(request.Run.ID, activityID, assignment, resource, handle)
		tasks[activityID], running[activityID] = task, handle
		if handle.Status == domain.HandleCompleted {
			completeTask(&task, handle)
			tasks[activityID], completed[activityID] = task, task
			delete(running, activityID)
		}
		if err := s.executions.SaveTask(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func newRunningTask(
	runID string,
	activityID string,
	assignment domain.PlanAssignment,
	resource domain.Resource,
	handle domain.ActivityHandle,
) domain.TaskExecution {
	return domain.TaskExecution{
		ID: runID + ":" + activityID, ExecutionRunID: runID,
		PlanAssignmentID: assignment.ID, ActivityID: activityID,
		PlannedResourceID: assignment.ResourceID, AllocatedResourceID: resource.ID,
		Attempt: 1, Status: domain.TaskRunning, StartedAt: handle.StartedAt,
	}
}

func completeTask(task *domain.TaskExecution, handle domain.ActivityHandle) {
	task.Status, task.FinishedAt = domain.TaskCompleted, handle.FinishedAt
	task.RuntimeSeconds = maxFloat(0, handle.FinishedAt-handle.StartedAt)
}

func validateRequest(request ports.ExecutionRequest) error {
	if request.Run.ID == "" || request.Plan.ID == "" || request.Workflow.ID == "" {
		return fmt.Errorf("run, plan and workflow identifiers are required")
	}
	if request.Run.Mode != domain.ExecutionModeReal && request.Run.Mode != domain.ExecutionModeSimulation && request.Run.Mode != domain.ExecutionModeInteractive {
		return fmt.Errorf("unsupported execution mode %q", request.Run.Mode)
	}
	if len(request.Plan.Assignments) != len(request.Workflow.Activities) {
		return fmt.Errorf("plan must assign every activity exactly once")
	}
	return nil
}

func indexActivities(values []domain.Activity) map[string]domain.Activity {
	result := make(map[string]domain.Activity, len(values))
	for _, value := range values {
		result[value.ID] = value
	}
	return result
}
func indexResources(values []domain.Resource) map[string]domain.Resource {
	result := make(map[string]domain.Resource, len(values))
	for _, value := range values {
		result[value.ID] = value
	}
	return result
}
func indexAssignments(values []domain.PlanAssignment) map[string]domain.PlanAssignment {
	result := make(map[string]domain.PlanAssignment, len(values))
	for _, value := range values {
		result[value.ActivityID] = value
	}
	return result
}

func readyActivities(activities map[string]domain.Activity, predecessors map[string][]string, completed map[string]domain.TaskExecution, running map[string]domain.ActivityHandle, tasks map[string]domain.TaskExecution) []string {
	ready := make([]string, 0)
	for id := range activities {
		if _, ok := completed[id]; ok {
			continue
		}
		if _, ok := running[id]; ok {
			continue
		}
		if _, attempted := tasks[id]; attempted {
			continue
		}
		all := true
		for _, predecessor := range predecessors[id] {
			if _, ok := completed[predecessor]; !ok {
				all = false
				break
			}
		}
		if all {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	return ready
}

func completedTrace(request ports.ExecutionRequest, tasks map[string]domain.TaskExecution) domain.ExecutionTrace {
	trace := runningTrace(request, tasks)
	trace.Executed.Feasible = true
	for _, task := range trace.Tasks {
		trace.Executed.MakespanSeconds = maxFloat(trace.Executed.MakespanSeconds, task.FinishedAt)
		trace.Executed.ComputeSeconds += task.RuntimeSeconds
		trace.Executed.Cost += task.Cost
	}
	return trace
}
func runningTrace(request ports.ExecutionRequest, tasks map[string]domain.TaskExecution) domain.ExecutionTrace {
	result := make([]domain.TaskExecution, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, task)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ActivityID < result[j].ActivityID })
	return domain.ExecutionTrace{RunID: request.Run.ID, PlanID: request.Plan.ID, Mode: request.Run.Mode, Predicted: request.Plan.Predicted, Tasks: result}
}
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
