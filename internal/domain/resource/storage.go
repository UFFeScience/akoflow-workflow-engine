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
	ID                   string         `json:"id"`
	EnvironmentVersionID string         `json:"environmentVersionId"`
	Name                 string         `json:"name"`
	Type                 StorageType    `json:"type"`
	Endpoint             string         `json:"endpoint,omitempty"`
	CapacityBytes        int64          `json:"capacityBytes"`
	Shared               bool           `json:"shared"`
	ReadOnly             bool           `json:"readOnly"`
	Configuration        map[string]any `json:"configuration,omitempty"`
	CredentialReference  string         `json:"credentialReference,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}
