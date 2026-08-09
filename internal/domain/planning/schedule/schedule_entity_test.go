package schedule_entity

import "testing"

func TestScheduleEntityAccessorsAndDefaults(t *testing.T) {
	s := New(ScheduleEntity{ID: 7, Type: "beam", Code: "code", Name: "prism", PluginSoPath: "ignored"})
	if s.GetId() != 7 || s.GetType() != "beam" || s.GetCode() != "code" || s.GetName() != "prism" {
		t.Fatalf("unexpected schedule: %+v", s)
	}
	if (ScheduleEntity{}).GetName() != "default" {
		t.Fatal("empty name must use default")
	}
}
