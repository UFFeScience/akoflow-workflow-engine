package get_schedule_api_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain/planning/schedule"
	"github.com/stretchr/testify/require"
)

type scheduleRepositoryStub struct {
	schedule schedule_entity.ScheduleEntity
	err      error
}

func (s scheduleRepositoryStub) ListAllSchedules() ([]schedule_entity.ScheduleEntity, error) {
	return nil, nil
}
func (s scheduleRepositoryStub) CreateSchedule(string, string, string, string) (schedule_entity.ScheduleEntity, error) {
	return schedule_entity.ScheduleEntity{}, nil
}
func (s scheduleRepositoryStub) GetScheduleByName(string) (schedule_entity.ScheduleEntity, error) {
	return s.schedule, s.err
}

func TestGetScheduleByNameMapsSchedule(t *testing.T) {
	result, err := NewWithRepository(scheduleRepositoryStub{schedule: schedule_entity.ScheduleEntity{ID: 4, Name: "prism", Type: "plugin", Code: "code"}}).GetScheduleByName("prism")
	require.NoError(t, err)
	require.Equal(t, 4, result.ID)
	require.Equal(t, "prism", result.Name)
}

func TestGetScheduleByNamePropagatesError(t *testing.T) {
	expected := errors.New("not found")
	_, err := NewWithRepository(scheduleRepositoryStub{err: expected}).GetScheduleByName("missing")
	require.ErrorIs(t, err, expected)
}
