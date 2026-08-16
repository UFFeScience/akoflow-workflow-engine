package environment

import (
	"time"

	"github.com/UFFeScience/akoflow/internal/domain/resource"
	"github.com/UFFeScience/akoflow/internal/domain/workflow"
)

type EnvironmentVersionStatus string
type EnvironmentStatus string
type ConnectionType string
type DiscoveryRunStatus string

const (
	EnvironmentVersionDraft     EnvironmentVersionStatus = "draft"
	EnvironmentVersionPublished EnvironmentVersionStatus = "published"
	EnvironmentVersionRetired   EnvironmentVersionStatus = "retired"
)

const (
	EnvironmentDefined     EnvironmentStatus = "defined"
	EnvironmentConnecting  EnvironmentStatus = "connecting"
	EnvironmentConnected   EnvironmentStatus = "connected"
	EnvironmentDiscovering EnvironmentStatus = "discovering"
	EnvironmentReady       EnvironmentStatus = "ready"
	EnvironmentDegraded    EnvironmentStatus = "degraded"
	EnvironmentUnreachable EnvironmentStatus = "unreachable"

	ConnectionSSH        ConnectionType = "ssh"
	ConnectionKubernetes ConnectionType = "kubernetes"
	ConnectionCloud      ConnectionType = "cloud"
	ConnectionLocal      ConnectionType = "local"
	ConnectionAgent      ConnectionType = "agent"
)

type Environment struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Status      EnvironmentStatus `json:"status"`
	CreatedAt   time.Time         `json:"createdAt"`
}

type EnvironmentConnection struct {
	ID            string         `json:"id"`
	EnvironmentID string         `json:"environmentId"`
	Name          string         `json:"name"`
	Type          ConnectionType `json:"type"`
	Endpoint      string         `json:"endpoint,omitempty"`
	Username      string         `json:"username,omitempty"`
	CredentialRef string         `json:"credentialRef,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type Capabilities struct {
	Batch         bool `json:"batch"`
	Interactive   bool `json:"interactive"`
	Container     bool `json:"container"`
	Serverless    bool `json:"serverless"`
	GPU           bool `json:"gpu"`
	MPI           bool `json:"mpi"`
	SharedStorage bool `json:"sharedStorage"`
	DataStaging   bool `json:"dataStaging"`
	Cancellation  bool `json:"cancellation"`
	LogStreaming  bool `json:"logStreaming"`
	Simulation    bool `json:"simulation"`
}

type DiscoveryRun struct {
	ID                   string             `json:"id"`
	EnvironmentVersionID string             `json:"environmentVersionId"`
	Provider             string             `json:"provider"`
	Status               DiscoveryRunStatus `json:"status"`
	ResourcesFound       int                `json:"resourcesFound"`
	Error                string             `json:"error,omitempty"`
	StartedAt            time.Time          `json:"startedAt"`
	FinishedAt           *time.Time         `json:"finishedAt,omitempty"`
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
	Capabilities         Capabilities   `json:"capabilities"`
}

type Definition struct {
	Environment Environment                        `json:"environment"`
	Version     EnvironmentVersion                 `json:"version"`
	Runtimes    []EnvironmentRuntime               `json:"runtimes"`
	Resources   []resource.Resource                `json:"resources"`
	Links       []resource.NetworkLink             `json:"networkLinks"`
	Profiles    []workflow.ActivityResourceProfile `json:"activityResourceProfiles,omitempty"`
	Connections []EnvironmentConnection            `json:"connections,omitempty"`
}
