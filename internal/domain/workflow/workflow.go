package workflow

type WorkflowVersion struct {
	ID               string                   `json:"id"`
	WorkflowID       string                   `json:"workflowId"`
	Version          int                      `json:"version"`
	DefinitionHash   string                   `json:"definitionHash"`
	Activities       []Activity               `json:"activities"`
	Dependencies     []ActivityDependency     `json:"dependencies"`
	DataDependencies []ActivityDataDependency `json:"dataDependencies,omitempty"`
}

type ActivityType struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Application      string         `json:"application,omitempty"`
	DefaultImage     string         `json:"defaultImage,omitempty"`
	CPUIntensity     float64        `json:"cpuIntensity"`
	MemoryIntensity  float64        `json:"memoryIntensity"`
	IOIntensity      float64        `json:"ioIntensity"`
	NetworkIntensity float64        `json:"networkIntensity"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type Activity struct {
	ID                   string         `json:"id"`
	WorkflowVersionID    string         `json:"workflowVersionId"`
	ActivityTypeID       string         `json:"activityTypeId"`
	ExternalID           string         `json:"externalId"`
	Name                 string         `json:"name"`
	Command              string         `json:"command"`
	Image                string         `json:"image"`
	Priority             int            `json:"priority"`
	CPURequired          float64        `json:"cpuRequired"`
	MemoryRequiredBytes  int64          `json:"memoryRequiredBytes"`
	StorageRequiredBytes int64          `json:"storageRequiredBytes"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

type ActivityDependency struct {
	ActivityID          string `json:"activityId"`
	DependsOnActivityID string `json:"dependsOnActivityId"`
	Type                string `json:"type,omitempty"`
}

type ActivityDataDependency struct {
	ProducerActivityID string `json:"producerActivityId"`
	ConsumerActivityID string `json:"consumerActivityId"`
	LogicalName        string `json:"logicalName"`
	SizeBytes          int64  `json:"sizeBytes"`
}

type ActivityResourceProfile struct {
	ID                   string         `json:"id"`
	ActivityTypeID       string         `json:"activityTypeId"`
	ResourceID           string         `json:"resourceId"`
	RuntimeSeconds       float64        `json:"runtimeSeconds"`
	RuntimeStdDevSeconds float64        `json:"runtimeStdDevSeconds"`
	CPUUtilization       float64        `json:"cpuUtilization"`
	PeakMemoryBytes      int64          `json:"peakMemoryBytes"`
	DiskReadBytes        int64          `json:"diskReadBytes"`
	DiskWriteBytes       int64          `json:"diskWriteBytes"`
	EnergyJoules         float64        `json:"energyJoules"`
	Source               string         `json:"source"`
	SampleSize           int            `json:"sampleSize"`
	ModelVersion         string         `json:"modelVersion"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}
