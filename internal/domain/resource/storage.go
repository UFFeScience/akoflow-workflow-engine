package resource

type StorageType string

const (
	StorageLocal  StorageType = "local"
	StoragePVC    StorageType = "pvc"
	StorageNFS    StorageType = "nfs"
	StorageS3     StorageType = "s3"
	StorageLustre StorageType = "lustre"
)

type StorageResource struct {
	ID                   string                  `json:"id"`
	EnvironmentVersionID string                  `json:"environmentVersionId"`
	Name                 string                  `json:"name"`
	Type                 StorageType             `json:"type"`
	Endpoint             string                  `json:"endpoint,omitempty"`
	CapacityBytes        int64                   `json:"capacityBytes"`
	Shared               bool                    `json:"shared"`
	ReadOnly             bool                    `json:"readOnly"`
	Configuration        map[string]any          `json:"configuration,omitempty"`
	CredentialReference  string                  `json:"credentialReference,omitempty"`
	Metadata             map[string]any          `json:"metadata,omitempty"`
	RuntimeBindings      []StorageRuntimeBinding `json:"runtimeBindings,omitempty"`
}

// StorageRuntimeBinding declares that a runtime can access a storage resource.
// Default is used when an activity does not explicitly select a storage.
type StorageRuntimeBinding struct {
	RuntimeID     string         `json:"runtimeId"`
	Default       bool           `json:"default,omitempty"`
	HostPath      string         `json:"hostPath,omitempty"`
	ContainerPath string         `json:"containerPath,omitempty"`
	ReadOnly      bool           `json:"readOnly,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
}
