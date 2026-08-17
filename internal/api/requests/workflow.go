package requests

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type Workflow struct {
	Name string       `json:"name"`
	Spec WorkflowSpec `json:"spec"`
}

type WorkflowSpec struct {
	Namespace        string             `json:"namespace"`
	Image            string             `json:"image,omitempty"`
	StorageClassName string             `json:"storageClassName,omitempty"`
	StorageSize      string             `json:"storageSize,omitempty"`
	StoragePolicy    StoragePolicy      `json:"storagePolicy,omitempty"`
	MountPath        string             `json:"mountPath,omitempty"`
	Activities       []WorkflowActivity `json:"activities"`
}

type StoragePolicy struct {
	Type string `json:"type,omitempty"`
}

type WorkflowActivity struct {
	Name             string   `json:"name"`
	Image            string   `json:"image,omitempty"`
	Runtime          string   `json:"runtime,omitempty"`
	Run              string   `json:"run"`
	MemoryLimit      string   `json:"memoryLimit,omitempty"`
	CPULimit         string   `json:"cpuLimit,omitempty"`
	DependsOn        []string `json:"dependsOn,omitempty"`
	ResourceSelector string   `json:"resourceSelector,omitempty"`
	KeepDisk         bool     `json:"keepDisk,omitempty"`
	MountPath        string   `json:"mountPath,omitempty"`
}

func (request Workflow) Domain() (domain.WorkflowDefinition, error) {
	workflowID := identifier(request.Name)
	if workflowID == "" || request.Spec.Namespace == "" {
		return domain.WorkflowDefinition{}, fmt.Errorf("workflow name and spec.namespace are required")
	}
	versionID := workflowID + "-v1"
	typeID := workflowID + "-activity"
	definition := domain.WorkflowDefinition{
		ID: workflowID, ExternalID: workflowID, Name: request.Name, Namespace: request.Spec.Namespace,
		Types:   []domain.ActivityType{{ID: typeID, Name: request.Name + " activity", DefaultImage: request.Spec.Image}},
		Version: domain.WorkflowVersion{ID: versionID, WorkflowID: workflowID, Version: 1, DefinitionHash: versionID},
	}
	names := make(map[string]string, len(request.Spec.Activities))
	for _, value := range request.Spec.Activities {
		id := identifier(value.Name)
		if id == "" {
			return domain.WorkflowDefinition{}, fmt.Errorf("activity name is required")
		}
		if _, exists := names[value.Name]; exists {
			return domain.WorkflowDefinition{}, fmt.Errorf("duplicate activity %q", value.Name)
		}
		names[value.Name] = id
	}
	for index, value := range request.Spec.Activities {
		activity, err := activityDomain(value, request.Spec.Image, versionID, typeID, index)
		if err != nil {
			return domain.WorkflowDefinition{}, err
		}
		definition.Version.Activities = append(definition.Version.Activities, activity)
		for _, predecessor := range value.DependsOn {
			predecessorID, exists := names[predecessor]
			if !exists {
				return domain.WorkflowDefinition{}, fmt.Errorf("activity %q depends on unknown activity %q", value.Name, predecessor)
			}
			definition.Version.Dependencies = append(definition.Version.Dependencies, domain.ActivityDependency{
				ActivityID: activity.ID, DependsOnActivityID: predecessorID, Type: "control",
			})
		}
	}
	return definition, nil
}

func activityDomain(value WorkflowActivity, defaultImage, versionID, typeID string, index int) (domain.Activity, error) {
	cpu, err := cpuValue(value.CPULimit)
	if err != nil {
		return domain.Activity{}, fmt.Errorf("activity %q cpuLimit: %w", value.Name, err)
	}
	memory, err := byteValue(value.MemoryLimit)
	if err != nil {
		return domain.Activity{}, fmt.Errorf("activity %q memoryLimit: %w", value.Name, err)
	}
	image := value.Image
	if image == "" {
		image = defaultImage
	}
	if image == "" || value.Run == "" {
		return domain.Activity{}, fmt.Errorf("activity %q image and run are required", value.Name)
	}
	return domain.Activity{
		ID: identifier(value.Name), WorkflowVersionID: versionID, ActivityTypeID: typeID,
		ExternalID: identifier(value.Name), Name: value.Name, Kind: domain.ActivityKindTask,
		Capabilities: []domain.ActivityCapability{domain.ActivityCapabilityReal},
		Command:      domain.ActivityCommand{Image: image, Entrypoint: "sh", Arguments: []string{"-c", value.Run}},
		Resources:    domain.ActivityResources{CPU: cpu, MemoryBytes: memory},
		Policy:       domain.ActivityPolicy{TimeoutSeconds: 3600, MaxAttempts: 1}, Priority: len(value.DependsOn) + index,
		Metadata: map[string]any{"runtime": value.Runtime, "resourceSelector": value.ResourceSelector,
			"keepDisk": value.KeepDisk, "mountPath": value.MountPath},
	}, nil
}

func identifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Trim(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			return r
		}
		return '-'
	}, value), "-")
}

func cpuValue(value string) (float64, error) {
	if value == "" {
		return 0.1, nil
	}
	if strings.HasSuffix(value, "m") {
		milli, err := strconv.ParseFloat(strings.TrimSuffix(value, "m"), 64)
		return milli / 1000, err
	}
	return strconv.ParseFloat(value, 64)
}

func byteValue(value string) (int64, error) {
	if value == "" {
		return 16 * 1024 * 1024, nil
	}
	multipliers := map[string]float64{"Ki": 1024, "Mi": 1024 * 1024, "Gi": 1024 * 1024 * 1024}
	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(value, suffix) {
			number, err := strconv.ParseFloat(strings.TrimSuffix(value, suffix), 64)
			return int64(number * multiplier), err
		}
	}
	return strconv.ParseInt(value, 10, 64)
}
