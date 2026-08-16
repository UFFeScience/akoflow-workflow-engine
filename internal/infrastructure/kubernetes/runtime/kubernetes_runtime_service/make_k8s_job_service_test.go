package kubernetes_runtime_service

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type jobActivityRepo struct {
	ports.ActivityRepository
	activity workflow_activity_entity.WorkflowActivities
}

func (r *jobActivityRepo) Find(int) (workflow_activity_entity.WorkflowActivities, error) {
	return r.activity, nil
}
func (r *jobActivityRepo) FindPreActivity(int) (workflow_activity_entity.WorkflowPreActivityDatabase, error) {
	return workflow_activity_entity.WorkflowPreActivityDatabase{Name: "pre"}, nil
}
func (r *jobActivityRepo) GetActivityScheduleByActivityId(int) (workflow_activity_entity.ActivitySchedule, error) {
	return workflow_activity_entity.ActivitySchedule{}, nil
}

type jobWorkflowRepo struct {
	ports.WorkflowRepository
	workflow workflow_entity.Workflow
}

func (r *jobWorkflowRepo) Find(int) (workflow_entity.Workflow, error) { return r.workflow, nil }

func jobBuilderFixture() MakeK8sJobService {
	wf, a := activityFixture(false)
	a.ResourceSelector = "zone=edge"
	return MakeK8sJobService{
		namespace: "lab", dependencies: []workflow_activity_entity.WorkflowActivities{}, idWorkflow: wf.Id, idWorkflowActivity: a.Id, workflow: wf,
		workflowRepository: &jobWorkflowRepo{workflow: wf}, activityRepository: &jobActivityRepo{activity: a},
		makeK8sActivityService: newMakeK8sActivityService(), makeK8sActivityDistributedService: newMakeK8sActivityDistributedService(),
		makeK8sActivityStandaloneService: newMakeK8sActivityStandaloneService(), makeK8sActivityPreactivityService: newMakeK8sActivityPreactivityService(), mode: MODE_STANDALONE,
	}
}

func TestMakeK8sJobModes(t *testing.T) {
	m := jobBuilderFixture()
	if m.GetNamespace() != "lab" || m.GetMode() != MODE_STANDALONE || m.GetWorkflow().Id != 2 || m.GetIdWorkflow() != 2 || m.GetIdWorkflowActivity() != 3 || m.GetDependencies() == nil {
		t.Fatal("accessors")
	}
	if job, err := m.MakeK8sJob(); err != nil || job.Kind != "Job" {
		t.Fatalf("standalone: %v", err)
	}
	if job, err := m.UseDistributedMode().MakeK8sJob(); err != nil || job.Kind != "Job" {
		t.Fatalf("distributed: %v", err)
	}
	m.SetDependencies([]workflow_activity_entity.WorkflowActivities{{Id: 1, Name: "dep"}})
	if job, err := m.UsePreactivityMode().MakeK8sJob(); err != nil || len(job.Spec.Template.Spec.Volumes) != 2 {
		t.Fatalf("preactivity: %v", err)
	}
	if m.UseStandaloneMode().SetNamespace("other").SetWorkflow(m.workflow).SetIdWorkflow(2).SetIdWorkflowActivity(3) == nil {
		t.Fatal("setters")
	}
}

func TestMakeK8sJobValidation(t *testing.T) {
	invalid := jobBuilderFixture()
	invalid.namespace = ""
	if _, err := invalid.MakeK8sJob(); err == nil {
		t.Fatal("standalone validation")
	}
	if _, err := invalid.UseDistributedMode().MakeK8sJob(); err == nil {
		t.Fatal("distributed validation")
	}
	if _, err := invalid.UsePreactivityMode().MakeK8sJob(); err == nil {
		t.Fatal("preactivity validation")
	}
	constructed := NewMakeK8sJobService()
	if constructed.GetMode() != MODE_STANDALONE {
		t.Fatal("constructor")
	}
}
