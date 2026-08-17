package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type CredentialResolver interface {
	Authorize(context.Context, *http.Request, string) error
}

type Driver struct {
	client      *http.Client
	credentials CredentialResolver
}

func New(client *http.Client, credentials CredentialResolver) *Driver {
	if client == nil {
		client = http.DefaultClient
	}
	return &Driver{client: client, credentials: credentials}
}

func (*Driver) Type() domain.StorageType { return domain.StorageS3 }

func (d *Driver) Put(ctx context.Context, request ports.PutObjectRequest) (domain.DataLocation, error) {
	objectURL, err := objectURL(request.Storage.Endpoint, request.Key)
	if err != nil {
		return domain.DataLocation{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPut, objectURL, request.Source)
	if err != nil {
		return domain.DataLocation{}, err
	}
	if request.Size > 0 {
		httpRequest.ContentLength = request.Size
	}
	if err := d.authorize(ctx, httpRequest, request.Storage.CredentialReference); err != nil {
		return domain.DataLocation{}, err
	}
	if err := d.do(httpRequest, nil); err != nil {
		return domain.DataLocation{}, err
	}
	return domain.DataLocation{URI: objectURL, Status: domain.DataLocationAvailable,
		StorageResourceID: request.Storage.ID,
		Metadata:          map[string]any{"credentialReference": request.Storage.CredentialReference}}, nil
}

func (d *Driver) Get(ctx context.Context, request ports.GetObjectRequest) error {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, request.Location.URI, nil)
	if err != nil {
		return err
	}
	if err := d.authorize(ctx, httpRequest, metadataString(request.Location.Metadata, "credentialReference")); err != nil {
		return err
	}
	return d.do(httpRequest, request.Target)
}

func (d *Driver) Stat(ctx context.Context, location domain.DataLocation) (ports.ObjectStat, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, location.URI, nil)
	if err != nil {
		return ports.ObjectStat{}, err
	}
	if err := d.authorize(ctx, request, metadataString(location.Metadata, "credentialReference")); err != nil {
		return ports.ObjectStat{}, err
	}
	response, err := d.client.Do(request)
	if err != nil {
		return ports.ObjectStat{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ports.ObjectStat{}, fmt.Errorf("S3 stat returned %s", response.Status)
	}
	size, _ := strconv.ParseInt(response.Header.Get("Content-Length"), 10, 64)
	checksum := strings.Trim(response.Header.Get("ETag"), `"`)
	return ports.ObjectStat{SizeBytes: size, Checksum: checksum}, nil
}

func (d *Driver) Delete(ctx context.Context, location domain.DataLocation) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, location.URI, nil)
	if err != nil {
		return err
	}
	if err := d.authorize(ctx, request, metadataString(location.Metadata, "credentialReference")); err != nil {
		return err
	}
	return d.do(request, nil)
}

func (d *Driver) authorize(ctx context.Context, request *http.Request, reference string) error {
	if reference == "" || d.credentials == nil {
		return nil
	}
	if err := d.credentials.Authorize(ctx, request, reference); err != nil {
		return fmt.Errorf("resolve storage credential: %w", err)
	}
	return nil
}

func (d *Driver) do(request *http.Request, target io.Writer) error {
	response, err := d.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return fmt.Errorf("S3 request returned %s", response.Status)
	}
	if target != nil {
		_, err = io.Copy(target, response.Body)
	}
	return err
}

func objectURL(endpoint, key string) (string, error) {
	if strings.TrimSpace(endpoint) == "" {
		return "", fmt.Errorf("S3 endpoint is required")
	}
	segments, err := escapedSegments(key)
	if err != nil {
		return "", err
	}
	escaped := strings.Join(segments, "/")
	value := strings.ReplaceAll(endpoint, "{key}", escaped)
	if !strings.Contains(endpoint, "{key}") {
		value = strings.TrimRight(endpoint, "/") + "/" + escaped
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("S3 endpoint must be HTTP(S)")
	}
	return parsed.String(), nil
}

func escapedSegments(key string) ([]string, error) {
	parts := strings.Split(strings.Trim(key, "/"), "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == ".." {
			return nil, fmt.Errorf("S3 object key cannot escape its root")
		}
		if part != "" && part != "." {
			result = append(result, url.PathEscape(part))
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("S3 object key is required")
	}
	return result, nil
}

func metadataString(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return value
}
