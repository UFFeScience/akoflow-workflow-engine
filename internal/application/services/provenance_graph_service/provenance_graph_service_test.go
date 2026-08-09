package provenance_graph_service

import (
	"errors"
	"testing"

	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/storages_repository"
)

type storageFake struct {
	storages_repository.IStorageRepository
	values []storages_repository.StorageDatabase
}

func (f storageFake) FindByWorkflow(int) []storages_repository.StorageDatabase { return f.values }

type graphActivityFake struct {
	activity_repository.IActivityRepository
	names map[int]string
}

func (f graphActivityFake) Find(id int) (workflow_activity_entity.WorkflowActivities, error) {
	if name, ok := f.names[id]; ok {
		return workflow_activity_entity.WorkflowActivities{Id: id, Name: name}, nil
	}
	return workflow_activity_entity.WorkflowActivities{}, errors.New("missing")
}

func TestBuildGraphCreatesInputsOutputsFallbacksAndSkipsDirectories(t *testing.T) {
	initial := `[{"Permissions":"-rw","Path":"./data","Name":"input.dat"},{"Permissions":"drwx","Path":"./data","Name":"dir"}]`
	end := `[{"Permissions":"-rw","Path":"./data","Name":"input.dat"},{"Permissions":"-rw","Path":"./data","Name":"output.dat"},{"Permissions":"drwx","Path":"./data","Name":"dir"}]`
	service := New(storageFake{values: []storages_repository.StorageDatabase{{ActivityId: 1, InitialFileList: initial, EndFileList: end}, {ActivityId: 2, InitialFileList: `[{"Permissions":"-rw","Path":"./data","Name":"output.dat"}]`, EndFileList: `[]`}}}, graphActivityFake{names: map[int]string{1: "producer"}})
	graph, err := service.BuildGraph(7)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Nodes) != 4 {
		t.Fatalf("unexpected nodes: %+v", graph.Nodes)
	}
	if len(graph.Edges) != 3 {
		t.Fatalf("unexpected edges: %+v", graph.Edges)
	}
	types := map[string]string{}
	for _, node := range graph.Nodes {
		types[node.Id] = node.Type
	}
	if types["file:./data/input.dat"] != "preExisting" || types["file:./data/output.dat"] != "file" || types["activity:activity_2"] != "activity" {
		t.Fatalf("unexpected node types: %v", types)
	}
}

func TestBuildGraphHandlesEmptyAndMalformedSnapshots(t *testing.T) {
	graph, err := New(storageFake{values: []storages_repository.StorageDatabase{{ActivityId: 1, InitialFileList: `{`, EndFileList: `bad`}}}, graphActivityFake{names: map[int]string{1: "a"}}).BuildGraph(1)
	if err != nil || len(graph.Nodes) != 1 || len(graph.Edges) != 0 {
		t.Fatalf("unexpected graph: %+v %v", graph, err)
	}
}
