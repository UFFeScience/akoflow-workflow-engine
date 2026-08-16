package internal_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRequiredArchitectureDirectoriesExist(t *testing.T) {
	required := []string{
		"domain/workflow", "domain/environment", "domain/resource",
		"domain/planning", "domain/execution", "application/services",
		"application/ports", "infrastructure/database",
		"infrastructure/plugins", "runtime/local", "runtime/kubernetes",
		"runtime/slurm", "execution/orchestrator", "execution/lifecycle",
		"execution/simulation", "api/handlers",
	}
	for _, path := range required {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Errorf("required architecture directory %q is missing", path)
		}
	}
}

func TestInternalCodeDoesNotImportRemovedServerTree(t *testing.T) {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imported := range file.Imports {
			value, unquoteErr := strconv.Unquote(imported.Path.Value)
			if unquoteErr != nil {
				return unquoteErr
			}
			if strings.Contains(value, "/pkg/server/") {
				t.Errorf("%s still imports removed server tree: %s", path, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
