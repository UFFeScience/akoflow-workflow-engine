package health

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(request *http.Request) (*http.Response, error) { return f(request) }

type logSpy struct{ calls int }

func (l *logSpy) Infof(string, ...interface{}) { l.calls++ }

func TestConfigurationAccessors(t *testing.T) {
	service := NewWithDependencies(http.DefaultClient, &logSpy{}).
		SetHost("localhost").SetPort("8080").SetServices([]string{"database"})
	if service.GetHost() != "localhost" || service.GetPort() != "8080" || len(service.services) != 1 {
		t.Fatal("configuration was not retained")
	}
	if New() == nil {
		t.Fatal("default service is nil")
	}
	standardLogger{}.Infof("test message")
}

func TestRunReportsHealthyAndUnhealthyServices(t *testing.T) {
	logger := &logSpy{}
	responses := []string{
		`{"service":"database","status":"OK"}`,
		`{"service":"worker","status":"FAILED","message":"offline"}`,
	}
	index := 0
	client := doerFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" {
			t.Fatal("unexpected request")
		}
		response := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responses[index]))}
		index++
		return response, nil
	})
	NewWithDependencies(client, logger).SetHost("localhost").SetPort("8080").SetServices([]string{"database", "worker"}).Run()
	if logger.calls != 2 {
		t.Fatalf("log calls = %d", logger.calls)
	}
}

func TestRunReportsTransportAndDecodeErrors(t *testing.T) {
	logger := &logSpy{}
	transportFailure := doerFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	NewWithDependencies(transportFailure, logger).SetHost("localhost").SetPort("8080").SetServices([]string{"database"}).Run()

	invalidJSON := doerFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{"))}, nil
	})
	NewWithDependencies(invalidJSON, logger).SetHost("localhost").SetPort("8080").SetServices([]string{"worker"}).Run()
	if logger.calls != 2 {
		t.Fatalf("log calls = %d", logger.calls)
	}
}

func TestCheckServiceRejectsInvalidURL(t *testing.T) {
	logger := &logSpy{}
	service := NewWithDependencies(http.DefaultClient, logger).SetHost("bad\nhost").SetPort("8080")
	service.checkService(http.DefaultClient, "database")
	if logger.calls != 1 {
		t.Fatalf("log calls = %d", logger.calls)
	}
}
