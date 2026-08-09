package workflow_entity

import (
	"encoding/base64"
	"reflect"
	"sort"
	"testing"

	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
)

func TestWorkflowConstructionAndAccessors(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("name: wf\nspec:\n  runtime: local\n  namespace: ns\n  mountPath: /data\n  storagePolicy:\n    type: distributed\n    storageClassName: fast\n    storageSize: 2Gi\n  volumes:\n    - /a:/b\n  activities:\n    - name: a\n      runtime: k8s\n    - name: b\n      runtime: local\n"))
	id, status := 4, 2
	w := New(WorkflowNewParams{WorkflowBase64: raw, Id: &id, Status: &status})
	if w.Name != "wf" || w.GetId() != 4 || !w.Validate() || w.ToYaml() != "" {
		t.Fatalf("unexpected workflow: %+v", w)
	}
	if !w.IsStoragePolicyDistributed() || w.IsStoragePolicyStandalone() || w.GetMode() != MODE_DISTRIBUTED {
		t.Fatal("distributed mode failed")
	}
	if w.GetNamespace() != "ns" || w.GetMountPath() != "/data" || w.GetStorageClassName() != "fast" || w.GetStorageSize() != "2Gi" || w.GetStoragePolicyType() != "distributed" {
		t.Fatal("storage accessors failed")
	}
	if w.MakeVolumeNameDistributed() != "wf-volume-4" || w.MakeStorageClassNameDistributed() != "akoflow-nfs-4" || w.MakeWorkflowPersistentVolumeClaimName() != "wf-pvc-4-nfs" {
		t.Fatal("generated names failed")
	}
	volumes := w.GetVolumes()
	if len(volumes) != 1 || volumes[0].GetLocalPath() != "/a" || volumes[0].GetRemotePath() != "/b" {
		t.Fatal("volume parsing failed")
	}
	runtimes := w.GetRuntimeId()
	sort.Strings(runtimes)
	if len(runtimes) != 2 || runtimes[0] != "k8s" || runtimes[1] != "local" {
		t.Fatalf("runtime dedup failed: %v", runtimes)
	}
	if _, err := base64.StdEncoding.DecodeString(w.GetBase64Workflow()); err != nil {
		t.Fatal(err)
	}
	db := DatabaseToWorkflow(ParamsDatabaseToWorkflow{WorkflowDatabase: WorkflowDatabase{ID: 5, Status: 1, RawWorkflow: raw}})
	if db.Id != 5 || db.Status != 1 {
		t.Fatal("database conversion failed")
	}
}

func TestWorkflowModesAndInvalidInput(t *testing.T) {
	if got := New(WorkflowNewParams{WorkflowBase64: "%%%"}); !reflect.DeepEqual(got, Workflow{}) {
		t.Fatal("invalid input must return zero value")
	}
	standalone := Workflow{}
	if !standalone.IsStoragePolicyStandalone() || standalone.GetMode() != MODE_STANDALONE {
		t.Fatal("default must be standalone")
	}
	unknown := Workflow{Spec: WorkflowSpec{StoragePolicy: WorkflowSpecStoragePolicy{Type: "other"}}}
	if unknown.GetMode() != "" {
		t.Fatal("unknown mode must be empty")
	}
	withRuntime := Workflow{Spec: WorkflowSpec{Runtime: "same", Activities: []workflow_activity_entity.WorkflowActivities{{Runtime: "same"}}}}
	if len(withRuntime.GetRuntimeId()) != 1 {
		t.Fatal("duplicates must be removed")
	}
}
