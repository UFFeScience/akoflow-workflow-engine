package connector_pvc_k8s

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	nfs "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/entity/nfs"
)

func pvcConnector(s *httptest.Server) *ConnectorPvcK8s {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return &ConnectorPvcK8s{client: s.Client(), runtime: r}
}

func TestPVCCrud(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(201)
		case http.MethodDelete:
			w.WriteHeader(204)
		default:
			w.WriteHeader(200)
		}
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "persistentvolumeclaims") {
			_, _ = w.Write([]byte(`[]`))
		} else {
			_, _ = w.Write([]byte(`{"kind":"PersistentVolumeClaim","metadata":{"name":"p"}}`))
		}
	}))
	defer s.Close()
	c := pvcConnector(s)
	if _, err := c.ListPvcs("ns"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreatePersistentVolumeClain("p", "ns", "1Gi", "fast"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetPersistentVolumeClain("p", "ns"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeletePersistentVolumeClaim("p", "ns"); err != nil {
		t.Fatal(err)
	}
	p := nfs.PersistentVolumeClaim{Metadata: nfs.Metadata{Name: "p", Namespace: "ns"}}
	if !c.CreatePvc(p).Success {
		t.Fatal("CreatePvc")
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}

func TestPVCFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); _, _ = w.Write([]byte("fail")) }))
	c := pvcConnector(s)
	if _, err := c.ListPvcs("ns"); err == nil {
		t.Fatal("list status")
	}
	if _, err := c.CreatePersistentVolumeClain("p", "ns", "1Gi", "x"); err == nil {
		t.Fatal("create status")
	}
	if err := c.DeletePersistentVolumeClaim("p", "ns"); err == nil {
		t.Fatal("delete status")
	}
	if c.CreatePvc(nfs.PersistentVolumeClaim{Metadata: nfs.Metadata{Namespace: "ns"}}).Success {
		t.Fatal("create status")
	}
	s.Close()
	if _, err := c.GetPersistentVolumeClain("p", "ns"); err == nil {
		t.Fatal("network")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if _, err := pvcConnector(bad).ListPvcs("ns"); err == nil {
		t.Fatal("JSON")
	}
}
