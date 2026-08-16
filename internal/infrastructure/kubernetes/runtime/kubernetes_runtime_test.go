package kubernetes_runtime

import "testing"

func TestKubernetesRuntimeSimpleContract(t *testing.T) {
	k := &KubernetesRuntime{}
	if k.SetRuntimeName("cluster") != k || k.SetRuntimeType("k8s") != k || k.GetRuntimeName() != "cluster" {
		t.Fatal("setters")
	}
	if k.StartConnection() != nil || k.StopConnection() != nil {
		t.Fatal("connection")
	}
	if !k.DeleteJob(1, 2) || k.GetStatus(1, 2) != "" {
		t.Fatal("contract")
	}
}
