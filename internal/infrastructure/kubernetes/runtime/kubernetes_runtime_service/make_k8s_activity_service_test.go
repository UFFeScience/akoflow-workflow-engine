package kubernetes_runtime_service

import (
	"encoding/base64"
	"strings"
	"testing"

	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

func activityFixture(distributed bool) (workflow_entity.Workflow, workflow_activity_entity.WorkflowActivities) {
	policy := ""
	if distributed {
		policy = workflow_entity.MODE_DISTRIBUTED
	}
	wf := workflow_entity.Workflow{Id: 2, Name: "wf", Spec: workflow_entity.WorkflowSpec{MountPath: "/data", StoragePolicy: workflow_entity.WorkflowSpecStoragePolicy{Type: policy}}}
	a := workflow_activity_entity.WorkflowActivities{Id: 3, Name: "task", Run: "echo ok", Image: "busybox", MemoryLimit: "1Gi"}
	return wf, a
}

func TestMakeK8sActivityBuilder(t *testing.T) {
	wf, a := activityFixture(false)
	m := newMakeK8sActivityService()
	if m.SetWorkflow(wf).SetIdWorkflowActivity(3).SetActivitySchedule(workflow_activity_entity.ActivitySchedule{ResourceID: "node-a"}) != &m {
		t.Fatal("fluent setters")
	}
	if m.getIdWorkflow() != 2 || m.getIdWorkflowActivity() != 3 || m.GetWorkflow().Name != "wf" {
		t.Fatal("state")
	}
	if got := m.makeJobVolumeMountPath(wf, a); got != "/data/task" {
		t.Fatalf("mount: %s", got)
	}
	a.MountPath = "/custom"
	if m.makeJobVolumeMountPath(wf, a) != "/custom" {
		t.Fatal("custom mount")
	}
	a.MountPath = ""
	if m.MakeResourceSelector(wf, a)["akoflow.io/resource-id"] != "node-a" {
		t.Fatal("schedule selector")
	}
	a.ResourceSelector = "zone=edge"
	if m.MakeResourceSelector(wf, a)["zone"] != "edge" {
		t.Fatal("explicit selector")
	}
	container := m.makeContainerActivity(wf, a)
	if container.Image != "busybox" || len(container.VolumeMounts) != 1 || len(container.Env) < 5 {
		t.Fatal("container")
	}
	decoded, err := base64.StdEncoding.DecodeString(m.makeContainerCommandActivity(wf, a))
	if err != nil || !strings.Contains(string(decoded), "echo ok") || !strings.Contains(string(decoded), "akouser_3") {
		t.Fatal("command")
	}
	if len(m.makeVolumesActivity(wf, a)) != 1 || m.makeVolumeThatWillBeUsedByCurrentActivity(wf, a).Name != "pvc-3-wfa" {
		t.Fatal("volumes")
	}
}

func TestMakeK8sActivityDistributedAndDefaults(t *testing.T) {
	wf, a := activityFixture(true)
	m := newMakeK8sActivityService()
	if m.getPortAkoFlowServer() == "" {
		t.Fatal("default port")
	}
	if m.makeJobVolumeMountPath(wf, a) != "/data" {
		t.Fatal("distributed mount")
	}
	if m.makeJobVolumeMounts(wf, a)[0].Name != wf.MakeWorkflowPersistentVolumeClaimName() {
		t.Fatal("distributed mount volume")
	}
	if m.makeVolumeThatWillBeUsedByCurrentActivity(wf, a).Name != wf.MakeWorkflowPersistentVolumeClaimName() {
		t.Fatal("distributed volume")
	}
	if !strings.Contains(m.setupCommandWorkdir(wf, a), "cd /data") {
		t.Fatal("workdir")
	}
	if m.MakeResourceSelector(wf, a) != nil {
		t.Fatal("unexpected selector")
	}
}
