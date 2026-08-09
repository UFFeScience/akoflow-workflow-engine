package environment

import "time"

type EnvironmentVersionStatus string

const (
	EnvironmentVersionDraft     EnvironmentVersionStatus = "draft"
	EnvironmentVersionPublished EnvironmentVersionStatus = "published"
	EnvironmentVersionRetired   EnvironmentVersionStatus = "retired"
)

type Environment struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type EnvironmentVersion struct {
	ID                string                   `json:"id"`
	EnvironmentID     string                   `json:"environmentId"`
	Version           int                      `json:"version"`
	Status            EnvironmentVersionStatus `json:"status"`
	NetworkModel      string                   `json:"networkModel"`
	InterferenceModel string                   `json:"interferenceModel"`
	CostModel         string                   `json:"costModel"`
	StorageModel      string                   `json:"storageModel"`
	ConfigurationHash string                   `json:"configurationHash"`
	CreatedAt         time.Time                `json:"createdAt"`
	PublishedAt       *time.Time               `json:"publishedAt,omitempty"`
}

type EnvironmentRuntime struct {
	EnvironmentVersionID string         `json:"environmentVersionId"`
	RuntimeID            string         `json:"runtimeId"`
	Role                 string         `json:"role,omitempty"`
	Configuration        map[string]any `json:"configuration,omitempty"`
}
