package mapper_engine_api

import (
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

func TestWorkflowMappings(t *testing.T) {
	w := workflow_entity.Workflow{Id: 3, Name: "wf", Status: 1, Spec: workflow_entity.WorkflowSpec{Activities: []workflow_activity_entity.WorkflowActivities{{Id: 2, Name: "task"}}}}
	mapped := MapEngineWorkflowEntityToApiWorkflowEntity(w)
	if mapped.Id != 3 || mapped.Name != "wf" || len(mapped.Spec.Activities) != 1 {
		t.Fatalf("unexpected mapping: %+v", mapped)
	}
	list := MapEngineWorkflowEntityToApiWorkflowEntityList([]workflow_entity.Workflow{w, w})
	if len(list) != 2 {
		t.Fatal("workflow list mapping failed")
	}
	if got := MapEngineWorkflowEntityToApiWorkflowEntityList(nil); len(got) != 0 {
		t.Fatal("nil list must stay empty")
	}
}

func TestRuntimeMappings(t *testing.T) {
	r := runtime_entity.Runtime{Name: "k8s", Status: 1, Metadata: map[string]string{"a": "b"}}
	mapped := MapEngineRuntimeEntityToApiRuntimeEntity(r)
	if mapped.Name != "k8s" || mapped.Status != 1 || mapped.Metadata["a"] != "b" {
		t.Fatalf("unexpected mapping: %+v", mapped)
	}
	if got := MapEngineRuntimeEntityToApiRuntimeEntityList([]runtime_entity.Runtime{r}); len(got) != 1 {
		t.Fatal("runtime list mapping failed")
	}
	if got := MapEngineRuntimeEntityToApiRuntimeEntityList(nil); len(got) != 0 {
		t.Fatal("nil list must stay empty")
	}
}
