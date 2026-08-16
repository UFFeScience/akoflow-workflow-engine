package health_check_runtime_check_service

import "testing"

type runtimeFake struct {
	healthy bool
	calls   int
}

func (f *runtimeFake) HealthCheck() bool { f.calls++; return f.healthy }
func TestHandleHealthyUnhealthyAndMissingRuntime(t *testing.T) {
	healthy := &runtimeFake{healthy: true}
	if err := NewWithDependencies(func(string) RuntimeHealth { return healthy }).Handle("local"); err != nil || healthy.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, healthy.calls)
	}
	if err := NewWithDependencies(func(string) RuntimeHealth { return &runtimeFake{} }).Handle("bad"); err == nil {
		t.Fatal("expected unhealthy error")
	}
	if err := NewWithDependencies(func(string) RuntimeHealth { return nil }).Handle("missing"); err == nil {
		t.Fatal("expected missing error")
	}
}
func TestConstructorsInitializeResolver(t *testing.T) {
	if New().runtimeResolver == nil || NewHealthCheckRuntimeCheckService().runtimeResolver == nil {
		t.Fatal("resolver not initialized")
	}
}
