// Package storage implements bounded browsing and download preparation. It is
// intentionally independent from transfer planning: browsing never copies an
// object implicitly.
package storage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type Catalog interface {
	ListEnvironmentStorages(context.Context, string) ([]domain.StorageResource, error)
	FindStorage(context.Context, string) (*domain.StorageResource, error)
	SaveDownload(context.Context, domain.DownloadRun) error
	FindDownload(context.Context, string) (*domain.DownloadRun, error)
	PromoteData(context.Context, string, string, string, string, string, string, domain.FileEntry, string) error
	PromoteArtifact(context.Context, string, string, string, string, string, string, string, string, domain.FileEntry) error
	SaveIndexRun(context.Context, domain.IndexRun) error
	ListIndexRuns(context.Context, string) ([]domain.IndexRun, error)
}

func (s *BrowserCoordinator) IndexRuns(ctx context.Context, id string) ([]domain.IndexRun, error) {
	return s.catalog.ListIndexRuns(ctx, id)
}
func (s *BrowserCoordinator) StartIndex(ctx context.Context, id, runID string) (domain.IndexRun, error) {
	v, e := s.catalog.FindStorage(ctx, id)
	if e != nil || v == nil {
		return domain.IndexRun{}, notFound(e)
	}
	if !v.IndexPolicy.Enabled {
		return domain.IndexRun{}, fmt.Errorf("indexing is disabled for this storage")
	}
	run := domain.IndexRun{ID: runID, StorageID: id, Status: "running", CreatedAt: time.Now().UTC()}
	_ = s.catalog.SaveIndexRun(ctx, run)
	limit := v.IndexPolicy.MaxEntries
	if limit <= 0 || limit > 100000 {
		limit = 10000
	}
	depth := v.IndexPolicy.MaxDepth
	if depth <= 0 {
		depth = 5
	}
	if seconds := v.IndexPolicy.TimeoutSeconds; seconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
		defer cancel()
	}
	b, e := s.resolver.Browser(v.Type)
	if e == nil {
		roots := v.IndexPolicy.Roots
		if len(roots) == 0 {
			for _, r := range v.BrowseRoots {
				roots = append(roots, r.Path)
			}
		}
		var walk func(string, int) error
		walk = func(p string, d int) error {
			if e := ctx.Err(); e != nil {
				return e
			}
			if run.IndexedEntries >= limit {
				return nil
			}
			page, e := b.Browse(ctx, *v, domain.BrowseRequest{Path: p, Limit: 500})
			if e != nil {
				return e
			}
			for _, x := range page.Entries {
				if excluded(x.Path, v.IndexPolicy.Exclude) {
					continue
				}
				// Directories must remain traversable when an include filter exists,
				// otherwise a matching descendant could never be reached.
				if x.Type != domain.FileEntryDirectory && !included(x.Path, v.IndexPolicy.Include) {
					continue
				}
				run.IndexedEntries++
				if x.Type == domain.FileEntryDirectory && d < depth {
					if e = walk(x.Path, d+1); e != nil {
						return e
					}
				}
				if run.IndexedEntries >= limit {
					return nil
				}
			}
			return nil
		}
		for _, r := range roots {
			if e = walk(r, 0); e != nil {
				break
			}
		}
	}
	run.FinishedAt = time.Now().UTC()
	if e != nil {
		run.Status = "failed"
		run.Error = e.Error()
	} else {
		run.Status = "completed"
	}
	_ = s.catalog.SaveIndexRun(ctx, run)
	return run, e
}

func included(value string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		if ok, _ := path.Match(p, value); ok {
			return true
		}
		if ok, _ := path.Match(p, path.Base(value)); ok {
			return true
		}
	}
	return false
}
func excluded(value string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := path.Match(p, value); ok {
			return true
		}
		if ok, _ := path.Match(p, path.Base(value)); ok {
			return true
		}
	}
	return false
}

func (s *BrowserCoordinator) PromoteData(ctx context.Context, storageID, filePath, workflowVersionID, runID, activityID, id string) error {
	entry, e := s.Stat(ctx, storageID, filePath)
	if e != nil {
		return e
	}
	if entry.Type != domain.FileEntryFile {
		return fmt.Errorf("only files can be promoted")
	}
	sum, e := s.Checksum(ctx, storageID, filePath)
	if e != nil {
		return e
	}
	return s.catalog.PromoteData(ctx, storageID, filePath, workflowVersionID, runID, activityID, id, entry, sum)
}
func (s *BrowserCoordinator) PromoteArtifact(ctx context.Context, storageID, filePath, id, name, version, scope, scopeID string) error {
	if !strings.HasSuffix(strings.ToLower(filePath), ".sif") {
		return fmt.Errorf("only .sif files can be executable artifacts")
	}
	entry, e := s.Stat(ctx, storageID, filePath)
	if e != nil {
		return e
	}
	if entry.Type != domain.FileEntryFile {
		return fmt.Errorf("artifact must be a file")
	}
	sum, e := s.Checksum(ctx, storageID, filePath)
	if e != nil {
		return e
	}
	return s.catalog.PromoteArtifact(ctx, storageID, filePath, id, name, version, scope, scopeID, sum, entry)
}

type Resolver interface {
	Browser(domain.StorageType) (ports.StorageBrowser, error)
}
type BrowserCoordinator struct {
	catalog  Catalog
	resolver Resolver
}

func NewBrowserCoordinator(c Catalog, r Resolver) *BrowserCoordinator {
	return &BrowserCoordinator{catalog: c, resolver: r}
}
func (s *BrowserCoordinator) List(ctx context.Context, environmentID string) ([]domain.StorageResource, error) {
	values, err := s.catalog.ListEnvironmentStorages(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	for index := range values {
		_, driverErr := s.resolver.Browser(values[index].Type)
		available := driverErr == nil
		values[index].Capabilities = domain.StorageCapabilities{
			Browse: available, Read: available, Download: available,
			Write:              available && !values[index].ReadOnly,
			Upload:             available && !values[index].ReadOnly,
			Checksum:           available,
			ComputeNodeVisible: values[index].Shared || len(values[index].RuntimeBindings) > 0,
		}
		if values[index].Type == domain.StorageS3 || values[index].Type == domain.StorageMinIO {
			values[index].Capabilities.PresignedDownload = available
		}
		if available {
			values[index].Health.Status = domain.StorageHealth("healthy")
			values[index].Health.Message = "storage driver is available"
		} else {
			values[index].Health.Status = domain.StorageHealth("unavailable")
			values[index].Health.Message = driverErr.Error()
		}
	}
	return values, nil
}
func (s *BrowserCoordinator) Roots(ctx context.Context, id string) ([]domain.StorageBrowseRoot, error) {
	v, e := s.catalog.FindStorage(ctx, id)
	if e != nil || v == nil {
		return nil, notFound(e)
	}
	return v.BrowseRoots, nil
}
func (s *BrowserCoordinator) Browse(ctx context.Context, id string, r domain.BrowseRequest) (domain.BrowsePage, error) {
	v, e := s.catalog.FindStorage(ctx, id)
	if e != nil || v == nil {
		return domain.BrowsePage{}, notFound(e)
	}
	b, e := s.resolver.Browser(v.Type)
	if e != nil {
		return domain.BrowsePage{}, e
	}
	return b.Browse(ctx, *v, r)
}
func (s *BrowserCoordinator) Stat(ctx context.Context, id, path string) (domain.FileEntry, error) {
	v, e := s.catalog.FindStorage(ctx, id)
	if e != nil || v == nil {
		return domain.FileEntry{}, notFound(e)
	}
	b, e := s.resolver.Browser(v.Type)
	if e != nil {
		return domain.FileEntry{}, e
	}
	return b.BrowseStat(ctx, *v, path)
}
func (s *BrowserCoordinator) StartDownload(ctx context.Context, id, path, runID string) (domain.DownloadRun, error) {
	v, e := s.catalog.FindStorage(ctx, id)
	if e != nil || v == nil {
		return domain.DownloadRun{}, notFound(e)
	}
	b, e := s.resolver.Browser(v.Type)
	if e != nil {
		return domain.DownloadRun{}, e
	}
	entry, e := b.BrowseStat(ctx, *v, path)
	if e != nil {
		return domain.DownloadRun{}, e
	}
	if entry.Type == domain.FileEntryDirectory {
		return domain.DownloadRun{}, fmt.Errorf("directories require an archive job")
	}
	run := domain.DownloadRun{ID: runID, StorageID: id, Path: path, Status: domain.DownloadReady, Strategy: "stream", SizeBytes: entry.SizeBytes, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if e = s.catalog.SaveDownload(ctx, run); e != nil {
		return domain.DownloadRun{}, e
	}
	return run, nil
}
func (s *BrowserCoordinator) OpenDownload(ctx context.Context, id string) (io.ReadCloser, domain.FileEntry, error) {
	run, e := s.catalog.FindDownload(ctx, id)
	if e != nil || run == nil {
		return nil, domain.FileEntry{}, notFound(e)
	}
	v, e := s.catalog.FindStorage(ctx, run.StorageID)
	if e != nil || v == nil {
		return nil, domain.FileEntry{}, notFound(e)
	}
	b, e := s.resolver.Browser(v.Type)
	if e != nil {
		return nil, domain.FileEntry{}, e
	}
	return b.Open(ctx, *v, run.Path)
}
func (s *BrowserCoordinator) Download(ctx context.Context, id string) (*domain.DownloadRun, error) {
	v, e := s.catalog.FindDownload(ctx, id)
	if e != nil || v == nil {
		return nil, notFound(e)
	}
	return v, nil
}
func (s *BrowserCoordinator) Checksum(ctx context.Context, id, path string) (string, error) {
	v, e := s.catalog.FindStorage(ctx, id)
	if e != nil || v == nil {
		return "", notFound(e)
	}
	b, e := s.resolver.Browser(v.Type)
	if e != nil {
		return "", e
	}
	body, entry, e := b.Open(ctx, *v, path)
	if e != nil {
		return "", e
	}
	defer body.Close()
	if entry.Type != domain.FileEntryFile {
		return "", fmt.Errorf("checksum requires a file")
	}
	h := sha256.New()
	_, e = io.Copy(h, body)
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), e
}
func (s *BrowserCoordinator) Delete(ctx context.Context, id, path string) error {
	v, e := s.catalog.FindStorage(ctx, id)
	if e != nil || v == nil {
		return notFound(e)
	}
	b, e := s.resolver.Browser(v.Type)
	if e != nil {
		return e
	}
	return b.Remove(ctx, *v, path)
}
func (s *BrowserCoordinator) QueueCopy(ctx context.Context, id, path, destination, runID string) (domain.DownloadRun, error) {
	entry, e := s.Stat(ctx, id, path)
	if e != nil {
		return domain.DownloadRun{}, e
	}
	now := time.Now().UTC()
	run := domain.DownloadRun{
		ID: runID, StorageID: id, Path: path,
		Status: domain.DownloadStreaming, Strategy: "copy:" + destination,
		SizeBytes: entry.SizeBytes, CreatedAt: now, UpdatedAt: now,
	}
	if e = s.catalog.SaveDownload(ctx, run); e != nil {
		return run, e
	}
	dst, e := s.catalog.FindStorage(ctx, destination)
	if e != nil || dst == nil {
		run.Status = domain.DownloadFailed
		run.Error = "destination storage is unavailable"
		_ = s.catalog.SaveDownload(ctx, run)
		return run, notFound(e)
	}
	src, e := s.catalog.FindStorage(ctx, id)
	if e != nil || src == nil {
		return run, notFound(e)
	}
	sb, e := s.resolver.Browser(src.Type)
	if e != nil {
		return run, e
	}
	db, e := s.resolver.Browser(dst.Type)
	if e != nil {
		return run, e
	}
	body, _, e := sb.Open(ctx, *src, path)
	if e == nil {
		e = db.Write(ctx, *dst, path, body, entry.SizeBytes)
		body.Close()
	}
	if e != nil {
		run.Status = domain.DownloadFailed
		run.Error = e.Error()
	} else {
		run.Status = domain.DownloadCompleted
		run.TransferredBytes = entry.SizeBytes
	}
	run.UpdatedAt = time.Now().UTC()
	_ = s.catalog.SaveDownload(ctx, run)
	return run, e
}
func (s *BrowserCoordinator) QueueArchive(ctx context.Context, id, path, runID string) (domain.DownloadRun, error) {
	entry, e := s.Stat(ctx, id, path)
	if e != nil {
		return domain.DownloadRun{}, e
	}
	if entry.Type != domain.FileEntryDirectory {
		return domain.DownloadRun{}, fmt.Errorf("archive requires a directory")
	}
	now := time.Now().UTC()
	run := domain.DownloadRun{
		ID: runID, StorageID: id, Path: path,
		Status: domain.DownloadFailed, Strategy: "archive",
		Error:     "archive worker is not configured; submit an archive worker for this storage",
		CreatedAt: now, UpdatedAt: now,
	}
	e = s.catalog.SaveDownload(ctx, run)
	return run, e
}
func notFound(e error) error {
	if e != nil {
		return e
	}
	return fmt.Errorf("storage resource not found")
}

type Registry map[domain.StorageType]ports.StorageBrowser

func (r Registry) Browser(t domain.StorageType) (ports.StorageBrowser, error) {
	b := r[t]
	if b == nil {
		return nil, fmt.Errorf("storage browser %q is unavailable", t)
	}
	return b, nil
}
