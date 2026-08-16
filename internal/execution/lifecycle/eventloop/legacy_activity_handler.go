package eventloop

import (
	"context"
	"encoding/json"
	"fmt"

	domainqueue "github.com/UFFeScience/akoflow/internal/domain/queue"
)

const EventActivitySubmissionRequested = "activity.submission.requested"

type ActivityRunner interface {
	Run(activityID int) error
}

type ActivitySubmissionPayload struct {
	ActivityID int `json:"activityId"`
}

type LegacyActivityHandler struct {
	runner ActivityRunner
}

func NewLegacyActivityHandler(runner ActivityRunner) *LegacyActivityHandler {
	return &LegacyActivityHandler{runner: runner}
}

func (h *LegacyActivityHandler) Handle(_ context.Context, job domainqueue.Job) error {
	if h.runner == nil {
		return fmt.Errorf("activity runner is not configured")
	}
	var payload ActivitySubmissionPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode activity submission: %w", err)
	}
	if payload.ActivityID <= 0 {
		return fmt.Errorf("activity id must be positive")
	}
	return h.runner.Run(payload.ActivityID)
}
