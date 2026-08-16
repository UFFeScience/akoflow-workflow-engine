package garbage_collector_remove_storage_service

import (
	"errors"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	connector_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type gcStorage struct {
	ports.StorageRepository
	updates, detached int
	err               error
}

func (s *gcStorage) Update(ports.UpdateStorageParams) error { s.updates++; return s.err }
func (s *gcStorage) UpdateDetached(int) error               { s.detached++; return nil }

type gcRuntime struct {
	runtime_repository.IRuntimeRepository
	runtime *runtime_entity.Runtime
}

func (r *gcRuntime) GetByName(string) (*runtime_entity.Runtime, error) { return r.runtime, nil }
func TestGarbageCollectorResourcePaths(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "persistentvolumeclaims") {
			w.WriteHeader(204)
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(200)
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"pod"}}]}`))
	}))
	defer server.Close()
	rt := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(server.URL, "https://")}, "", "")
	storage := &gcStorage{}
	runtimeRepo := &gcRuntime{runtime: rt}
	s := &GarbageCollectorRemoveStorageService{namespace: "ns", storageRepository: storage, runtimeRepository: runtimeRepo, connector: connector_k8s.New()}
	a := workflow_activity_entity.WorkflowActivities{Id: 3, Name: "task", Runtime: "k8s"}
	s.RemoveStorages()
	s.removeResource(a)
	if storage.updates != 1 || storage.detached != 1 {
		t.Fatal("remove")
	}
	s.handleKeepDisk(a)
	if storage.updates != 2 {
		t.Fatal("keep")
	}
	storage.err = errors.New("db")
	s.handleKeepDisk(a)
	runtimeRepo.runtime = nil
	s.removeResource(a)
}
