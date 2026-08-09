package list_schedules_api_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain/planning/schedule"
	"github.com/stretchr/testify/require"
)

type scheduleRepositoryStub struct {
	schedules []schedule_entity.ScheduleEntity
	err       error
}

func (s scheduleRepositoryStub) ListAllSchedules() ([]schedule_entity.ScheduleEntity, error) {
	return s.schedules, s.err
}
func (s scheduleRepositoryStub) CreateSchedule(string, string, string, string) (schedule_entity.ScheduleEntity, error) {
	return schedule_entity.ScheduleEntity{}, nil
}
func (s scheduleRepositoryStub) GetScheduleByName(string) (schedule_entity.ScheduleEntity, error) {
	return schedule_entity.ScheduleEntity{}, s.err
}

func TestListAllSchedulesMapsRepositoryEntities(t *testing.T) {
	service := NewWithRepository(scheduleRepositoryStub{schedules: []schedule_entity.ScheduleEntity{{ID: 1, Name: "prism", Type: "plugin"}}})
	result, err := service.ListAllSchedules()
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "prism", result[0].Name)
}

func TestListAllSchedulesPropagatesRepositoryError(t *testing.T) {
	expected := errors.New("database unavailable")
	_, err := NewWithRepository(scheduleRepositoryStub{err: expected}).ListAllSchedules()
	require.ErrorIs(t, err, expected)
}
