package utils_read_file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFilePanicsForMissingFile(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	New().ReadFile(filepath.Join(t.TempDir(), "missing"))
}

func TestGetRootProjectPath(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(original)
	root := t.TempDir()
	nested := filepath.Join(root, "cmd", "server")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	if got := New().GetRootProjectPath(); got != root {
		t.Fatalf("got %q want %q", got, root)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if got := New().GetRootProjectPath(); got != root {
		t.Fatalf("ordinary cwd changed: %q", got)
	}
}

func TestReadFile(t *testing.T) {
	utils := New()

	// Create a temporary file
	file, err := os.CreateTemp("", "testfile")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	// Write some content to the file
	content := "Hello, World!"
	_, err = file.Write([]byte(content))
	require.NoError(t, err)
	file.Close()

	// Test reading the file
	result := utils.ReadFile(file.Name())
	assert.Equal(t, content, result)
}

func TestReadFile_FileNotFound(t *testing.T) {
	utils := New()

	assert.Panics(t, func() {
		utils.ReadFile("non_existent_file.txt")
	}, "expected panic for non-existent file")
}
