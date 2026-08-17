package observation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

func TestRunRecordsCreatedModifiedAndDeletedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "modified"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "deleted"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	command := []string{"sh", "-c", "printf new > created; printf changed > modified; rm deleted"}
	manifest, err := Run(context.Background(), Config{
		RunID: "run", ActivityID: "activity", Attempt: 1, Runtime: "local",
		Root: root, ManifestPath: manifestPath,
	}, command)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]domain.ArtifactChange{
		"created": domain.ArtifactCreated, "modified": domain.ArtifactModified,
		"deleted": domain.ArtifactDeleted,
	}
	for _, file := range manifest.Files {
		delete(want, file.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing observations: %v; manifest=%+v", want, manifest)
	}
	if len(manifest.Phases) != 3 || manifest.Summary.CreatedFiles != 1 ||
		manifest.Summary.ModifiedFiles != 1 || manifest.Summary.DeletedFiles != 1 {
		t.Fatalf("lifecycle observation=%+v", manifest)
	}
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatal(err)
	}
}

func TestRunWritesManifestWhenCommandFails(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	manifest, err := Run(context.Background(), Config{
		Root: t.TempDir(), ManifestPath: manifestPath,
	}, []string{"sh", "-c", "exit 7"})
	if err == nil || manifest.ExitCode != 7 {
		t.Fatalf("manifest=%+v err=%v", manifest, err)
	}
	if _, statErr := os.Stat(manifestPath); statErr != nil {
		t.Fatal(statErr)
	}
}
