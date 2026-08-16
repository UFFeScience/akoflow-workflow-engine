package execution

import (
	"context"
	"fmt"
	"sort"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/plugins/planning"
)

type SimulationExecutor struct {
	validator planning.ValidatePlanService
}

func NewSimulationExecutor() *SimulationExecutor {
	return &SimulationExecutor{validator: planning.NewValidatePlanService()}
}

func (e *SimulationExecutor) Execute(ctx context.Context, request Request) (domain.ExecutionTrace, error) {
	if request.Run.Mode != domain.ExecutionModeSimulation {
		return domain.ExecutionTrace{}, fmt.Errorf("simulation executor cannot execute mode %q", request.Run.Mode)
	}
	if err := e.validator.Validate(request.Plan, request.Workflow, request.Resources); err != nil {
		return domain.ExecutionTrace{}, err
	}
	select {
	case <-ctx.Done():
		return domain.ExecutionTrace{}, ctx.Err()
	default:
	}

	resourceByID := make(map[string]domain.Resource, len(request.Resources))
	for _, resource := range request.Resources {
		resourceByID[resource.ID] = resource
	}
	assignmentByActivity := make(map[string]domain.PlanAssignment, len(request.Plan.Assignments))
	for _, assignment := range request.Plan.Assignments {
		assignmentByActivity[assignment.ActivityID] = assignment
	}
	profileByPair := make(map[string]domain.ActivityResourceProfile, len(request.ActivityProfiles))
	for _, profile := range request.ActivityProfiles {
		profileByPair[profile.ActivityTypeID+"\x00"+profile.ResourceID] = profile
	}
	activityByID := make(map[string]domain.Activity, len(request.Workflow.Activities))
	for _, activity := range request.Workflow.Activities {
		activityByID[activity.ID] = activity
	}
	predecessors := make(map[string][]string)
	for _, dependency := range request.Workflow.Dependencies {
		predecessors[dependency.ActivityID] = append(predecessors[dependency.ActivityID], dependency.DependsOnActivityID)
	}
	dataBytes := make(map[string]int64)
	for _, dependency := range request.Workflow.DataDependencies {
		key := dependency.ProducerActivityID + "\x00" + dependency.ConsumerActivityID
		dataBytes[key] += dependency.SizeBytes
	}

	// A resource lane is serialized in the exact order selected by the planner.
	lanePrevious := make(map[string]string)
	byLane := make(map[string][]domain.PlanAssignment)
	for _, assignment := range request.Plan.Assignments {
		lane := assignment.ResourceID + "\x00" + assignment.CoreID + "\x00" + assignment.SlotID
		byLane[lane] = append(byLane[lane], assignment)
	}
	for _, lane := range byLane {
		sort.Slice(lane, func(i, j int) bool { return lane[i].OrderOnResource < lane[j].OrderOnResource })
		for i := 1; i < len(lane); i++ {
			lanePrevious[lane[i].ActivityID] = lane[i-1].ActivityID
		}
	}

	completed := make(map[string]domain.TaskExecution, len(request.Workflow.Activities))
	transfers := make([]domain.DataTransfer, 0)
	remaining := len(request.Workflow.Activities)
	for remaining > 0 {
		progress := false
		for _, activity := range request.Workflow.Activities {
			if _, done := completed[activity.ID]; done {
				continue
			}
			if !allCompleted(predecessors[activity.ID], completed) {
				continue
			}
			previous := lanePrevious[activity.ID]
			if previous != "" {
				if _, done := completed[previous]; !done {
					continue
				}
			}

			assignment := assignmentByActivity[activity.ID]
			resource := resourceByID[assignment.ResourceID]
			dataReadyAt := 0.0
			transferTotal := 0.0
			for _, predecessorID := range predecessors[activity.ID] {
				predecessorExecution := completed[predecessorID]
				readyAt := predecessorExecution.FinishedAt
				bytes := dataBytes[predecessorID+"\x00"+activity.ID]
				if predecessorExecution.AllocatedResourceID != resource.ID && bytes > 0 {
					link, ok := resolveLink(request.NetworkLinks, predecessorExecution.AllocatedResourceID, resource.ID)
					if !ok {
						return domain.ExecutionTrace{}, fmt.Errorf("no network link from %q to %q", predecessorExecution.AllocatedResourceID, resource.ID)
					}
					duration := link.TransferSeconds(bytes)
					transfer := domain.DataTransfer{
						ID:             fmt.Sprintf("%s:%s:%s", request.Run.ID, predecessorID, activity.ID),
						ExecutionRunID: request.Run.ID, ProducerActivityID: predecessorID,
						ConsumerActivityID: activity.ID, SourceResourceID: predecessorExecution.AllocatedResourceID,
						TargetResourceID: resource.ID, Bytes: bytes, StartedAt: predecessorExecution.FinishedAt,
						FinishedAt: predecessorExecution.FinishedAt + duration, DurationSeconds: duration,
						Cost: float64(bytes) * link.PricePerByte,
					}
					transfers = append(transfers, transfer)
					readyAt = transfer.FinishedAt
					transferTotal += duration
				}
				if readyAt > dataReadyAt {
					dataReadyAt = readyAt
				}
			}
			resourceReadyAt := 0.0
			if previous != "" {
				resourceReadyAt = completed[previous].FinishedAt
			}
			start := max(dataReadyAt, resourceReadyAt)
			runtimeSeconds := resolveRuntime(activity, resource, profileByPair)
			overhead := resource.BootOverheadSeconds + resource.ContainerOverhead
			finish := start + runtimeSeconds + overhead
			execution := domain.TaskExecution{
				ID: fmt.Sprintf("%s:%s", request.Run.ID, activity.ID), ExecutionRunID: request.Run.ID,
				PlanAssignmentID: assignment.ID, ActivityID: activity.ID,
				PlannedResourceID: resource.ID, AllocatedResourceID: resource.ID, Attempt: 1,
				Status: domain.TaskCompleted, ReadyAt: dataReadyAt, DataReadyAt: dataReadyAt,
				QueuedAt: dataReadyAt, StartedAt: start, FinishedAt: finish,
				RuntimeSeconds: runtimeSeconds, QueueSeconds: start - dataReadyAt,
				TransferSeconds: transferTotal, OverheadSeconds: overhead,
				Cost: (runtimeSeconds + overhead) * resource.PricePerSecond,
			}
			completed[activity.ID] = execution
			remaining--
			progress = true
		}
		if !progress {
			return domain.ExecutionTrace{}, fmt.Errorf("plan cannot progress: dependency and resource orders form a cycle")
		}
	}

	tasks := make([]domain.TaskExecution, 0, len(completed))
	metrics := domain.ExecutionMetrics{Feasible: true}
	for _, activity := range request.Workflow.Activities {
		task := completed[activity.ID]
		tasks = append(tasks, task)
		metrics.MakespanSeconds = max(metrics.MakespanSeconds, task.FinishedAt)
		metrics.ComputeSeconds += task.RuntimeSeconds
		metrics.QueueSeconds += task.QueueSeconds
		metrics.TransferSeconds += task.TransferSeconds
		metrics.OverheadSeconds += task.OverheadSeconds
		metrics.Cost += task.Cost
	}
	for _, transfer := range transfers {
		metrics.Cost += transfer.Cost
	}
	if request.Plan.DeadlineSeconds > 0 && metrics.MakespanSeconds > request.Plan.DeadlineSeconds {
		metrics.Feasible = false
	}
	if request.Plan.Budget > 0 && metrics.Cost > request.Plan.Budget {
		metrics.Feasible = false
	}
	return domain.ExecutionTrace{
		RunID: request.Run.ID, PlanID: request.Plan.ID, Mode: request.Run.Mode,
		Predicted: request.Plan.Predicted, Executed: metrics, Tasks: tasks, Transfers: transfers,
	}, nil
}

func allCompleted(ids []string, completed map[string]domain.TaskExecution) bool {
	for _, id := range ids {
		if _, ok := completed[id]; !ok {
			return false
		}
	}
	return true
}

func resolveRuntime(activity domain.Activity, resource domain.Resource, profiles map[string]domain.ActivityResourceProfile) float64 {
	if profile, ok := profiles[activity.ActivityTypeID+"\x00"+resource.ID]; ok {
		return profile.RuntimeSeconds
	}
	if base, ok := numberMetadata(activity.Metadata, "baseRuntimeSeconds"); ok && resource.ComputeSpeedup > 0 {
		return base / resource.ComputeSpeedup
	}
	if activity.Simulation != nil {
		return activity.Simulation.DurationSeconds
	}
	return 0
}

func numberMetadata(metadata map[string]any, key string) (float64, bool) {
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	default:
		return 0, false
	}
}

func resolveLink(links []domain.NetworkLink, source, target string) (domain.NetworkLink, bool) {
	for _, link := range links {
		if link.SourceResourceID == source && link.TargetResourceID == target {
			return link, true
		}
		if link.Bidirectional && link.SourceResourceID == target && link.TargetResourceID == source {
			return link, true
		}
	}
	return domain.NetworkLink{}, false
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
