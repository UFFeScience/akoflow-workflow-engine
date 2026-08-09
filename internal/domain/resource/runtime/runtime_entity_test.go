package runtime_entity

import "testing"

func TestRuntimeMetadataAndAccessors(t *testing.T) {
	r := NewRuntime("k8s1", 1, map[string]string{
		"K8S1_API_SERVER_TOKEN": "token",
		"K8S1_API_SERVER_HOST":  "host",
		"K8S1_REGION":           "region",
	}, "created", "updated")
	if r.GetName() != "k8s1" || r.GetStatus() != 1 || r.GetCreatedAt() != "created" || r.GetUpdatedAt() != "updated" {
		t.Fatalf("unexpected runtime: %+v", r)
	}
	if r.GetMetadataApiServerToken() != "token" || r.GetMetadataApiServerHost() != "host" || r.GetCurrentRuntimeMetadata("region") != "region" {
		t.Fatal("metadata lookup failed")
	}
	if r.GetCurrentRuntimeMetadata("missing") != "" {
		t.Fatal("missing metadata must be empty")
	}
	if len(r.GetMetadata()) != 3 {
		t.Fatal("metadata accessor failed")
	}
	if NewRuntime("empty", 0, nil, "", "").GetMetadataApiServerToken() != "" {
		t.Fatal("missing token must be empty")
	}
	if NewRuntime("empty", 0, nil, "", "").GetMetadataApiServerHost() != "" {
		t.Fatal("missing host must be empty")
	}
}
