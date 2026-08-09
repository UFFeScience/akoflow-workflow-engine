package list_runtimes_api_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	"github.com/stretchr/testify/require"
)

type runtimeRepositoryStub struct {
	runtimes []runtime_entity.Runtime
	err      error
}

func (s runtimeRepositoryStub) CreateOrUpdate(string, int, map[string]string)     {}
func (s runtimeRepositoryStub) GetAll() ([]runtime_entity.Runtime, error)         { return s.runtimes, s.err }
func (s runtimeRepositoryStub) GetByName(string) (*runtime_entity.Runtime, error) { return nil, s.err }
func (s runtimeRepositoryStub) UpdateStatus(*runtime_entity.Runtime, int) error   { return s.err }

func TestListAllRuntimesMapsRepositoryEntities(t *testing.T) {
	service := NewWithRepository(runtimeRepositoryStub{runtimes: []runtime_entity.Runtime{{Name: "k8s", Status: 1}}})
	result, err := service.ListAllRuntimes()
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "k8s", result[0].Name)
}

func TestListAllRuntimesPropagatesError(t *testing.T) {
	expected := errors.New("runtime catalog unavailable")
	_, err := NewWithRepository(runtimeRepositoryStub{err: expected}).ListAllRuntimes()
	require.ErrorIs(t, err, expected)
}
