package filesystem

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type Driver struct {
	typeName domain.StorageType
	root     string
}

func New(typeName domain.StorageType, root string) (*Driver, error) {
	if typeName != domain.StorageLocal && typeName != domain.StoragePVC &&
		typeName != domain.StorageNFS && typeName != domain.StorageLustre {
		return nil, fmt.Errorf("filesystem storage type %q is unsupported", typeName)
	}
	absolute, err := filepath.Abs(root)
	if err != nil || strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("filesystem storage root is required")
	}
	return &Driver{typeName: typeName, root: filepath.Clean(absolute)}, nil
}

func (d *Driver) Type() domain.StorageType { return d.typeName }

func (d *Driver) Put(_ context.Context, request ports.PutObjectRequest) (domain.DataLocation, error) {
	destination, err := d.resolve(request.Key)
	if err != nil {
		return domain.DataLocation{}, err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return domain.DataLocation{}, err
	}
	temporary := destination + ".akoflow-part"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
	if err != nil {
		return domain.DataLocation{}, err
	}
	_, copyErr := io.Copy(file, request.Source)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(temporary)
		return domain.DataLocation{}, fmt.Errorf("write storage object: %w", firstError(copyErr, closeErr))
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return domain.DataLocation{}, err
	}
	return domain.DataLocation{URI: (&url.URL{Scheme: "file", Path: destination}).String(),
		Status: domain.DataLocationAvailable}, nil
}

func (d *Driver) Get(_ context.Context, request ports.GetObjectRequest) error {
	path, err := filePath(request.Location.URI)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(request.Target, file)
	return err
}

func (d *Driver) Stat(_ context.Context, location domain.DataLocation) (ports.ObjectStat, error) {
	path, err := filePath(location.URI)
	if err != nil {
		return ports.ObjectStat{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return ports.ObjectStat{}, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return ports.ObjectStat{}, err
	}
	return ports.ObjectStat{SizeBytes: size, Checksum: fmt.Sprintf("sha256:%x", hash.Sum(nil))}, nil
}

func (d *Driver) Delete(_ context.Context, location domain.DataLocation) error {
	path, err := filePath(location.URI)
	if err != nil {
		return err
	}
	resolved, err := d.resolve(strings.TrimPrefix(path, d.root+string(filepath.Separator)))
	if err != nil || resolved != filepath.Clean(path) {
		return fmt.Errorf("location is outside storage root")
	}
	return os.Remove(path)
}

func (d *Driver) resolve(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage key %q escapes its root", key)
	}
	return filepath.Join(d.root, clean), nil
}

func filePath(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "file" {
		return "", fmt.Errorf("file location URI is required")
	}
	return filepath.Clean(parsed.Path), nil
}

func firstError(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
