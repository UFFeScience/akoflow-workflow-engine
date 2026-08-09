package utils_parser_params_ssh_client

import "testing"

func TestParseSSHClientsAndRejectMalformedEntries(t *testing.T) {
	clients := New().SetIdentityFile("/key").Parse("alice:pass@host:22,invalid,bob:pass@host:notaport,no-password@host:22,user:pass@missingport")
	if len(clients) != 1 {
		t.Fatalf("expected one client, got %d", len(clients))
	}
	c := clients[0]
	if c.Username != "alice" || c.Password != "pass" || c.Host != "host" || c.Port != 22 || c.IdentityFile != "/key" {
		t.Fatalf("unexpected client: %+v", c)
	}
	if got := New().Parse(""); len(got) != 0 {
		t.Fatal("empty input must return no clients")
	}
}
