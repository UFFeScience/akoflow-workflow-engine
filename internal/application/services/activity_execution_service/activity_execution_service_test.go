package activity_execution_service

import (
	"context"
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type resolverFake struct {
	adapter ports.RuntimeAdapter
	err     error
}

func (f resolverFake) Resolve(domain.ExecutionMode, string) (ports.RuntimeAdapter, error) {
	return f.adapter, f.err
}

type runtimeFake struct {
	handle  domain.ActivityHandle
	stopped bool
}

func (f *runtimeFake) Mode() domain.ExecutionMode { return domain.ExecutionModeReal }
func (f *runtimeFake) Start(context.Context, domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	return f.handle, nil
}
func (f *runtimeFake) Inspect(context.Context, domain.ActivityHandle) (domain.ActivityHandle, error) {
	f.handle.Status = domain.HandleCompleted
	return f.handle, nil
}
func (f *runtimeFake) Stop(context.Context, domain.ActivityHandle) error {
	f.stopped = true
	return nil
}

type handlesFake struct {
	handle *domain.ActivityHandle
	err    error
}

func (f *handlesFake) Save(_ context.Context, handle domain.ActivityHandle) error {
	if f.err != nil {
		return f.err
	}
	f.handle = &handle
	return nil
}
func (f *handlesFake) Find(context.Context, string) (*domain.ActivityHandle, error) {
	return f.handle, f.err
}

func executionFixture() domain.ActivityExecutionContext {
	return domain.ActivityExecutionContext{
		Run: domain.ExecutionRun{ID: "run", Mode: domain.ExecutionModeReal},
		Activity: domain.Activity{ID: "activity", Name: "activity", Kind: domain.ActivityKindTask,
			Capabilities: []domain.ActivityCapability{domain.ActivityCapabilityReal},
			Command:      domain.ActivityCommand{Entrypoint: "echo"}},
		Resource: domain.Resource{ID: "node", RuntimeID: "local"},
	}
}

func TestStartPersistsRuntimeIndependentHandle(t *testing.T) {
	runtime := &runtimeFake{handle: domain.ActivityHandle{ID: "handle", RunID: "run", ActivityID: "activity", RuntimeID: "local", Status: domain.HandleRunning}}
	handles := &handlesFake{}
	got, err := New(resolverFake{adapter: runtime}, handles).Start(context.Background(), executionFixture())
	if err != nil || got.ID != "handle" || handles.handle == nil {
		t.Fatalf("unexpected start: %+v %v", got, err)
	}
}

func TestStartStopsRuntimeWhenHandleCannotBePersisted(t *testing.T) {
	runtime := &runtimeFake{handle: domain.ActivityHandle{ID: "handle", RunID: "run", ActivityID: "activity"}}
	_, err := New(resolverFake{adapter: runtime}, &handlesFake{err: errors.New("database")}).Start(context.Background(), executionFixture())
	if err == nil || !runtime.stopped {
		t.Fatal("runtime must be stopped after persistence failure")
	}
}

func TestStartRejectsUnsupportedMode(t *testing.T) {
	execution := executionFixture()
	execution.Run.Mode = domain.ExecutionModeInteractive
	_, err := New(resolverFake{}, &handlesFake{}).Start(context.Background(), execution)
	if err == nil {
		t.Fatal("unsupported activity mode must fail")
	}
}

func TestInspectAndStop(t *testing.T) {
	runtime := &runtimeFake{handle: domain.ActivityHandle{ID: "handle", RuntimeID: "local", Status: domain.HandleRunning}}
	handles := &handlesFake{handle: &runtime.handle}
	service := New(resolverFake{adapter: runtime}, handles)
	updated, err := service.Inspect(context.Background(), "handle", domain.ExecutionModeReal)
	if err != nil || updated.Status != domain.HandleCompleted {
		t.Fatal("inspect failed")
	}
	if err := service.Stop(context.Background(), "handle", domain.ExecutionModeReal); err != nil || !runtime.stopped || handles.handle.Status != domain.HandleStopped {
		t.Fatal("stop failed")
	}
}
