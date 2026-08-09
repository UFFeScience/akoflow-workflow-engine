package workflow_activity_entity

import (
	"encoding/base64"
	"reflect"
	"testing"
)

func str(value string) *string { return &value }

func TestWorkflowActivityHelpers(t *testing.T) {
	a := WorkflowActivities{Id: 3, Name: "task", ProcId: "p", Runtime: "k8s", MountPath: "/data", ResourceSelector: "disk=ssd", DependsOn: []string{"root"}, MemoryLimit: "512Mi", CpuLimit: "250m"}
	if a.GetName() != "task" || a.GetNameJob() != "activity-3-task" || a.GetPreActivityName() != "preactivity-3" || a.GetVolumeName() != "pvc-3-wfa" || a.GetId() != 3 || a.GetProcId() != "p" || a.GetMountPath() != "/data" {
		t.Fatal("accessors failed")
	}
	if a.GetResourceSelector()["disk"] != "ssd" || !a.HasResourceSelector() || !a.HasDependencies() {
		t.Fatal("selector/dependency failed")
	}
	if a.GetMemoryRequired() != 512 || a.GetCpuRequired() != 250 || a.GetRuntimeId() != "k8s" {
		t.Fatal("resource parsing failed")
	}
	if _, err := base64.StdEncoding.DecodeString(a.GetBase64Activities()); err != nil {
		t.Fatal(err)
	}
	if (WorkflowActivities{}).GetMountPath() != "" || (WorkflowActivities{}).GetResourceSelector() != nil || (WorkflowActivities{}).HasDependencies() || (WorkflowActivities{}).GetMemoryRequired() != 0 || (WorkflowActivities{}).GetCpuRequired() != 0 {
		t.Fatal("zero-value behavior failed")
	}
	if (WorkflowPreActivityDatabase{ActivityId: 9}).GetPreActivityName() != "preactivity-9" {
		t.Fatal("preactivity name failed")
	}
}

func TestDatabaseToWorkflowActivities(t *testing.T) {
	raw := WorkflowActivities{Run: "echo", Runtime: "fallback", MemoryLimit: "1Mi", CpuLimit: "2m", DependsOn: []string{"a"}, ResourceSelector: "a=b", MountPath: "/m", KeepDisk: true}.GetBase64Activities()
	db := WorkflowActivityDatabase{Id: 2, WorkflowId: 4, Name: "task", Image: "img", Runtime: "docker", ResourceK8sBase64: raw, Status: 1, ProcId: str("proc"), CreatedAt: str("c"), StartedAt: str("s"), FinishedAt: str("f")}
	a := DatabaseToWorkflowActivities(ParamsDatabaseToWorkflowActivities{WorkflowActivityDatabase: db})
	if a.Id != 2 || a.Runtime != "docker" || a.Run != "echo" || !a.KeepDisk || a.CreatedAt != "c" || a.StartedAt != "s" || a.FinishedAt != "f" {
		t.Fatalf("unexpected conversion: %+v", a)
	}
	db.Runtime = ""
	if DatabaseToWorkflowActivities(ParamsDatabaseToWorkflowActivities{WorkflowActivityDatabase: db}).Runtime != "fallback" {
		t.Fatal("YAML runtime fallback failed")
	}
	if got := DatabaseToWorkflowActivities(ParamsDatabaseToWorkflowActivities{WorkflowActivityDatabase: WorkflowActivityDatabase{ResourceK8sBase64: "%%%"}}); !reflect.DeepEqual(got, WorkflowActivities{}) {
		t.Fatal("invalid base64 must return zero value")
	}
	invalidYAML := base64.StdEncoding.EncodeToString([]byte(":"))
	if got := DatabaseToWorkflowActivities(ParamsDatabaseToWorkflowActivities{WorkflowActivityDatabase: WorkflowActivityDatabase{ResourceK8sBase64: invalidYAML}}); !reflect.DeepEqual(got, WorkflowActivities{}) {
		t.Fatal("invalid YAML must return zero value")
	}
}

func TestWorkflowActivityPanicsOnInvalidResources(t *testing.T) {
	for name, fn := range map[string]func(){
		"memory":  func() { WorkflowActivities{MemoryLimit: "bad"}.GetMemoryRequired() },
		"cpu":     func() { WorkflowActivities{CpuLimit: "bad"}.GetCpuRequired() },
		"runtime": func() { WorkflowActivities{}.GetRuntimeId() },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			fn()
		})
	}
}
