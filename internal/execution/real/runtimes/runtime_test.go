package runtimes

import "testing"

func TestNormalizeAndResolveKnownRuntimes(t *testing.T) {
	cases := map[string]string{"k8s://cluster": "k8s", "hpc://partition": "hpc", "local": "local", "docker": "docker", "singularity": "singularity"}
	for input, want := range cases {
		if got := normalizeRuntime(input); got != want {
			t.Fatalf("normalize %q=%q", input, got)
		}
		if runtime := GetRuntimeInstance(input); runtime == nil {
			t.Fatalf("runtime %q missing", input)
		}
	}
	if GetRuntimeInstance("unknown") != nil {
		t.Fatal("unknown runtime must be nil")
	}
}
