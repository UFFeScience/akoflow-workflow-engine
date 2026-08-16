package queue

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusLeased    Status = "leased"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

const (
	CategoryEnvironment   = "environment"
	CategoryOrchestration = "orchestration"
	CategoryExecution     = "execution"
	CategoryMonitoring    = "monitoring"
	CategoryTransfer      = "transfer"
	CategoryMaintenance   = "maintenance"
)

type Job struct {
	ID             string
	Category       string
	Type           string
	AggregateType  string
	AggregateID    string
	Payload        []byte
	Status         Status
	Priority       int
	AvailableAt    time.Time
	LeaseOwner     string
	LeaseExpiresAt *time.Time
	Attempts       int
	MaxAttempts    int
	IdempotencyKey string
	LastError      string
	CreatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

func New(category, eventType string, payload []byte, now time.Time) (Job, error) {
	if category == "" || eventType == "" {
		return Job{}, errors.New("queue job category and type are required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Job{
		ID:          newID(),
		Category:    category,
		Type:        eventType,
		Payload:     append([]byte{}, payload...),
		Status:      StatusPending,
		AvailableAt: now,
		MaxAttempts: 5,
		CreatedAt:   now,
	}, nil
}

func (j Job) Validate() error {
	if j.ID == "" || j.Category == "" || j.Type == "" {
		return errors.New("queue job id, category and type are required")
	}
	if j.MaxAttempts < 1 {
		return errors.New("queue job max attempts must be positive")
	}
	return nil
}

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
}
