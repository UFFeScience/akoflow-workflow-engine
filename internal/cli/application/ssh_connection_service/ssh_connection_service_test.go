package ssh_connection_service

import (
	ssh_client_entity "github.com/UFFeScience/akoflow/internal/cli/domain/ssh_client"
	"testing"
)

func TestSSHHostsAndConnectionFailures(t *testing.T) {
	s := New()
	main := ssh_client_entity.SSHClient{Host: "127.0.0.1", Port: 1, Username: "u", Password: "p"}
	worker := ssh_client_entity.SSHClient{Host: "127.0.0.1", Port: 1, Username: "w", Password: "p"}
	s.AddHost(main)
	s.AddHost(worker)
	if s.GetMainNode().Username != "u" || len(s.GetAllHosts()) != 2 || len(s.GetWorkerNodes()) != 1 {
		t.Fatal("host selection")
	}
	if _, err := s.connect(main); err == nil {
		t.Fatal("connection expected to fail")
	}
	if got := s.ExecuteCommandsOnHost(main, []string{"true"}); got != "" {
		t.Fatal("failed host output")
	}
	s.EstablishConnectionWithHosts()
	s.ExecuteCommandsInMultipleHost([]string{"true"})
	s.CloseConnections()
}

func TestSSHIdentityFailuresAndNoWorkers(t *testing.T) {
	s := New()
	host := ssh_client_entity.SSHClient{Host: "localhost", Port: 22, Username: "u", IdentityFile: "/missing/key"}
	s.AddHost(host)
	if len(s.GetWorkerNodes()) != 0 {
		t.Fatal("workers")
	}
	if _, err := s.connect(host); err == nil {
		t.Fatal("missing key")
	}
}
