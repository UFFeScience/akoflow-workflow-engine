package create_schedule_api_service

import (
	"encoding/base64"
	"errors"
	"os"
	"plugin"
	"runtime"
	"strings"
	"testing"

	schedule_entity "github.com/UFFeScience/akoflow/internal/domain/planning/schedule"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
)

type repositoryFake struct {
	schedule_repository.IScheduleRepository
	value schedule_entity.ScheduleEntity
	err   error
	calls int
}

func (f *repositoryFake) CreateSchedule(name, kind, code, path string) (schedule_entity.ScheduleEntity, error) {
	f.calls++
	if f.value.Name == "" {
		f.value = schedule_entity.ScheduleEntity{ID: 1, Name: name, Type: kind, Code: code, PluginSoPath: path}
	}
	return f.value, f.err
}

type lookupFake struct {
	symbol plugin.Symbol
	err    error
}

func (f lookupFake) Lookup(string) (plugin.Symbol, error) { return f.symbol, f.err }
func serviceFixture(repo schedule_repository.IScheduleRepository) *CreateScheduleApiService {
	return NewWithDependencies(repo, func() ([]byte, error) { return []byte("go version " + runtime.Version()), nil }, func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }, func(string, []byte, os.FileMode) error { return nil }, func(string, string) error { return nil }, func(string) (PluginLookup, error) { return lookupFake{symbol: func(any) float64 { return 1 }}, nil })
}
func TestValidateUserCodeCompilesAndExecutes(t *testing.T) {
	s := serviceFixture(&repositoryFake{})
	valid, path := s.ValidateUserCode("package main")
	if !valid || !strings.HasSuffix(path, ".so") {
		t.Fatalf("valid=%v path=%q", valid, path)
	}
}
func TestValidateUserCodeRejectsVersionAndBuildFailures(t *testing.T) {
	s := serviceFixture(&repositoryFake{})
	s.versionOutput = func() ([]byte, error) { return nil, errors.New("go") }
	if ok, _ := s.ValidateUserCode("x"); ok {
		t.Fatal("version error")
	}
	s.versionOutput = func() ([]byte, error) { return []byte("different"), nil }
	if ok, _ := s.ValidateUserCode("x"); ok {
		t.Fatal("version mismatch")
	}
	s.versionOutput = func() ([]byte, error) { return []byte(runtime.Version()), nil }
	s.buildPlugin = func(string, string) error { return errors.New("build") }
	if ok, _ := s.ValidateUserCode("x"); ok {
		t.Fatal("build failure")
	}
}
func TestValidateUsesExistingPlugin(t *testing.T) {
	s := serviceFixture(&repositoryFake{})
	s.statFile = func(string) (os.FileInfo, error) { return nil, nil }
	builds := 0
	s.buildPlugin = func(string, string) error { builds++; return nil }
	if ok, _ := s.ValidateUserCode("x"); !ok || builds != 0 {
		t.Fatalf("ok=%v builds=%d", ok, builds)
	}
}
func TestCompilePluginHandlesWriteBuildAndSuccess(t *testing.T) {
	s := serviceFixture(&repositoryFake{})
	s.writeFile = func(string, []byte, os.FileMode) error { return errors.New("write") }
	if s.compilePlugin("a", "b", "code") {
		t.Fatal("write")
	}
	s.writeFile = func(string, []byte, os.FileMode) error { return nil }
	s.buildPlugin = func(string, string) error { return errors.New("build") }
	if s.compilePlugin("a", "b", "code") {
		t.Fatal("build")
	}
	s.buildPlugin = func(string, string) error { return nil }
	if !s.compilePlugin("a", "b", "code") {
		t.Fatal("success")
	}
}
func TestExecutePluginValidatesOpenLookupAndSymbol(t *testing.T) {
	s := serviceFixture(&repositoryFake{})
	s.pluginOpener = func(string) (PluginLookup, error) { return nil, errors.New("open") }
	if s.executePlugin("x") {
		t.Fatal("open")
	}
	s.pluginOpener = func(string) (PluginLookup, error) { return lookupFake{err: errors.New("lookup")}, nil }
	if s.executePlugin("x") {
		t.Fatal("lookup")
	}
	s.pluginOpener = func(string) (PluginLookup, error) { return lookupFake{symbol: "bad"}, nil }
	if s.executePlugin("x") {
		t.Fatal("symbol")
	}
	s.pluginOpener = func(string) (PluginLookup, error) { return lookupFake{symbol: func(any) float64 { return 2 }}, nil }
	if !s.executePlugin("x") {
		t.Fatal("success")
	}
}
func TestCreateScheduleValidationPersistenceAndErrors(t *testing.T) {
	repo := &repositoryFake{}
	s := serviceFixture(repo)
	s.validateCode = func(string) (bool, string) { return true, "plugin.so" }
	code := base64.StdEncoding.EncodeToString([]byte("code"))
	result, err := s.CreateSchedule("prism", "go", code)
	if err != nil || result.Name != "prism" || repo.calls != 1 {
		t.Fatalf("result=%+v calls=%d err=%v", result, repo.calls, err)
	}
	if _, err = s.CreateSchedule("x", "x", "not-base64"); err == nil {
		t.Fatal("base64")
	}
	s.validateCode = func(string) (bool, string) { return false, "" }
	if _, err = s.CreateSchedule("x", "x", code); err == nil {
		t.Fatal("validation")
	}
	repo.err = errors.New("db")
	s.validateCode = func(string) (bool, string) { return true, "x" }
	if _, err = s.CreateSchedule("x", "x", code); err == nil {
		t.Fatal("repository")
	}
}
func TestNewInitializesDependencies(t *testing.T) {
	s := New()
	if s.scheduleRepository == nil || s.versionOutput == nil || s.statFile == nil || s.writeFile == nil || s.buildPlugin == nil || s.pluginOpener == nil || s.validateCode == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
