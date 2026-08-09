package planning

import (
	"fmt"
	"sort"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type ValidatePlanService struct{}

func NewValidatePlanService() ValidatePlanService { return ValidatePlanService{} }

func (ValidatePlanService) Validate(plan domain.SchedulePlan, workflow domain.WorkflowVersion, resources []domain.Resource) error {
	if plan.WorkflowVersionID != workflow.ID {
		return fmt.Errorf("plan workflow version %q does not match %q", plan.WorkflowVersionID, workflow.ID)
	}
	resourceByID := make(map[string]domain.Resource, len(resources))
	for _, resource := range resources {
		resourceByID[resource.ID] = resource
	}
	activityByID := make(map[string]domain.Activity, len(workflow.Activities))
	for _, activity := range workflow.Activities {
		activityByID[activity.ID] = activity
	}
	assignmentByActivity := make(map[string]domain.PlanAssignment, len(plan.Assignments))
	for _, assignment := range plan.Assignments {
		activity, exists := activityByID[assignment.ActivityID]
		if !exists {
			return fmt.Errorf("assignment %q references unknown activity %q", assignment.ID, assignment.ActivityID)
		}
		if _, duplicate := assignmentByActivity[assignment.ActivityID]; duplicate {
			return fmt.Errorf("activity %q has more than one assignment", assignment.ActivityID)
		}
		resource, exists := resourceByID[assignment.ResourceID]
		if !exists {
			return fmt.Errorf("assignment %q references unknown resource %q", assignment.ID, assignment.ResourceID)
		}
		if resource.EnvironmentVersionID != plan.EnvironmentVersionID {
			return fmt.Errorf("resource %q belongs to another environment version", resource.ID)
		}
		if !resource.Schedulable {
			return fmt.Errorf("resource %q is not schedulable", resource.ID)
		}
		if resource.CPUCapacity < activity.CPURequired {
			return fmt.Errorf("resource %q lacks CPU for activity %q", resource.ID, activity.ID)
		}
		if resource.MemoryBytes < activity.MemoryRequiredBytes {
			return fmt.Errorf("resource %q lacks memory for activity %q", resource.ID, activity.ID)
		}
		if assignment.PredictedFinishAt < assignment.PredictedStartAt {
			return fmt.Errorf("assignment %q finishes before it starts", assignment.ID)
		}
		assignmentByActivity[assignment.ActivityID] = assignment
	}
	if len(assignmentByActivity) != len(activityByID) {
		return fmt.Errorf("plan covers %d of %d activities", len(assignmentByActivity), len(activityByID))
	}
	if err := validateDAG(workflow); err != nil {
		return err
	}
	return validateResourceOrders(plan.Assignments, workflow.Dependencies)
}

func validateDAG(workflow domain.WorkflowVersion) error {
	indegree := make(map[string]int, len(workflow.Activities))
	successors := make(map[string][]string, len(workflow.Activities))
	for _, activity := range workflow.Activities {
		indegree[activity.ID] = 0
	}
	for _, dependency := range workflow.Dependencies {
		if _, ok := indegree[dependency.ActivityID]; !ok {
			return fmt.Errorf("dependency references unknown activity %q", dependency.ActivityID)
		}
		if _, ok := indegree[dependency.DependsOnActivityID]; !ok {
			return fmt.Errorf("dependency references unknown predecessor %q", dependency.DependsOnActivityID)
		}
		if dependency.ActivityID == dependency.DependsOnActivityID {
			return fmt.Errorf("activity %q depends on itself", dependency.ActivityID)
		}
		indegree[dependency.ActivityID]++
		successors[dependency.DependsOnActivityID] = append(successors[dependency.DependsOnActivityID], dependency.ActivityID)
	}
	queue := make([]string, 0)
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, successor := range successors[id] {
			indegree[successor]--
			if indegree[successor] == 0 {
				queue = append(queue, successor)
			}
		}
	}
	if visited != len(indegree) {
		return fmt.Errorf("workflow dependencies contain a cycle")
	}
	return nil
}

func validateResourceOrders(assignments []domain.PlanAssignment, dependencies []domain.ActivityDependency) error {
	assignmentByActivity := make(map[string]domain.PlanAssignment, len(assignments))
	byLane := make(map[string][]domain.PlanAssignment)
	for _, assignment := range assignments {
		assignmentByActivity[assignment.ActivityID] = assignment
		lane := assignment.ResourceID + "\x00" + assignment.CoreID + "\x00" + assignment.SlotID
		byLane[lane] = append(byLane[lane], assignment)
	}
	for lane, items := range byLane {
		sort.Slice(items, func(i, j int) bool { return items[i].OrderOnResource < items[j].OrderOnResource })
		for i := 1; i < len(items); i++ {
			if items[i-1].OrderOnResource == items[i].OrderOnResource {
				return fmt.Errorf("resource lane %q contains duplicate order %d", lane, items[i].OrderOnResource)
			}
		}
	}
	for _, dependency := range dependencies {
		activityAssignment := assignmentByActivity[dependency.ActivityID]
		predecessorAssignment := assignmentByActivity[dependency.DependsOnActivityID]
		if activityAssignment.ResourceID == predecessorAssignment.ResourceID &&
			activityAssignment.CoreID == predecessorAssignment.CoreID &&
			activityAssignment.SlotID == predecessorAssignment.SlotID &&
			activityAssignment.OrderOnResource <= predecessorAssignment.OrderOnResource {
			return fmt.Errorf("resource order places activity %q before predecessor %q", dependency.ActivityID, dependency.DependsOnActivityID)
		}
	}
	return nil
}
