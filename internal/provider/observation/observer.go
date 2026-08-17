package observation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type Config struct {
	RunID, ActivityID, Runtime, Root, ManifestPath string
	Attempt                                        int
}

type fileState struct {
	size, modified int64
}

func Run(ctx context.Context, config Config, command []string) (domain.ArtifactManifest, error) {
	if len(command) == 0 {
		return domain.ArtifactManifest{}, fmt.Errorf("observed command is required")
	}
	if config.Root == "" {
		config.Root = "."
	}
	preStarted := time.Now()
	before, preErr := snapshot(config.Root)
	preFinished := time.Now()
	started := time.Now()
	process := exec.CommandContext(ctx, command[0], command[1:]...)
	process.Dir = config.Root
	process.Stdin, process.Stdout, process.Stderr = os.Stdin, os.Stdout, os.Stderr
	processErr := process.Run()
	finished := time.Now()
	postStarted := time.Now()
	after, snapshotErr := snapshot(config.Root)
	postFinished := time.Now()
	hostname, _ := os.Hostname()
	manifest := domain.ArtifactManifest{
		SchemaVersion: 1, RunID: config.RunID, ActivityID: config.ActivityID,
		Attempt: config.Attempt, Runtime: config.Runtime, Hostname: hostname, Root: config.Root,
		StartedAt:  float64(started.UnixNano()) / 1e9,
		FinishedAt: float64(finished.UnixNano()) / 1e9,
		ExitCode:   exitCode(processErr),
		Phases: []domain.LifecycleObservation{
			phase("pre_execution", preStarted, preFinished, preErr),
			phase("execution", started, finished, processErr),
			phase("post_execution", postStarted, postFinished, snapshotErr),
		},
	}
	if snapshotErr == nil {
		manifest.Files = diff(before, after)
	}
	manifest.Summary = summarize(before, after, manifest.Files)
	writeErr := WriteManifest(config.ManifestPath, manifest)
	return manifest, errors.Join(processErr, preErr, snapshotErr, writeErr)
}

func phase(name string, started, finished time.Time, err error) domain.LifecycleObservation {
	status := "completed"
	message := ""
	if err != nil {
		status, message = "failed", err.Error()
	}
	return domain.LifecycleObservation{Phase: name, Status: status,
		StartedAt: float64(started.UnixNano()) / 1e9, FinishedAt: float64(finished.UnixNano()) / 1e9,
		DurationSeconds: finished.Sub(started).Seconds(), Error: message}
}

func summarize(before, after map[string]fileState, files []domain.ArtifactObservation) domain.ArtifactSummary {
	summary := domain.ArtifactSummary{InitialFiles: len(before), FinalFiles: len(after)}
	for _, file := range files {
		switch file.Change {
		case domain.ArtifactCreated:
			summary.CreatedFiles++
			summary.OutputBytes += file.SizeBytes
		case domain.ArtifactModified:
			summary.ModifiedFiles++
			summary.OutputBytes += file.SizeBytes
		case domain.ArtifactDeleted:
			summary.DeletedFiles++
		}
	}
	return summary
}

func WriteManifest(path string, manifest domain.ArtifactManifest) error {
	if path == "" {
		return fmt.Errorf("manifest path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func snapshot(root string) (map[string]fileState, error) {
	result := make(map[string]fileState)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrPermission) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = fileState{size: info.Size(), modified: info.ModTime().UnixNano()}
		return nil
	})
	return result, err
}

func diff(before, after map[string]fileState) []domain.ArtifactObservation {
	files := make([]domain.ArtifactObservation, 0)
	for path, current := range after {
		previous, existed := before[path]
		change := domain.ArtifactCreated
		if existed {
			if previous == current {
				continue
			}
			change = domain.ArtifactModified
		}
		files = append(files, domain.ArtifactObservation{
			Path: path, Change: change, SizeBytes: current.size, ModifiedUnixNano: current.modified,
		})
	}
	for path := range before {
		if _, exists := after[path]; !exists {
			files = append(files, domain.ArtifactObservation{Path: path, Change: domain.ArtifactDeleted})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
