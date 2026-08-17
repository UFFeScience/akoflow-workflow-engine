package filesystem

import (
	"bytes"
	"context"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

func TestDriverRoundTripAndRootProtection(t *testing.T) {
	driver, err := New(domain.StoragePVC, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	location, err := driver.Put(context.Background(), ports.PutObjectRequest{
		Key: "run/activity/result.txt", Source: bytes.NewBufferString("result"),
	})
	if err != nil {
		t.Fatal(err)
	}
	stat, err := driver.Stat(context.Background(), location)
	if err != nil || stat.SizeBytes != 6 || stat.Checksum == "" {
		t.Fatalf("stat=%+v err=%v", stat, err)
	}
	var output bytes.Buffer
	if err := driver.Get(context.Background(), ports.GetObjectRequest{Location: location, Target: &output}); err != nil || output.String() != "result" {
		t.Fatalf("output=%q err=%v", output.String(), err)
	}
	if _, err := driver.Put(context.Background(), ports.PutObjectRequest{Key: "../escape", Source: bytes.NewReader(nil)}); err == nil {
		t.Fatal("path traversal must fail")
	}
	if err := driver.Delete(context.Background(), location); err != nil {
		t.Fatal(err)
	}
}
