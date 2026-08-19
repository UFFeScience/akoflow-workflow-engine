// Package domain is the stable facade for AkoFlow's bounded domain contexts.
// New behavior belongs in the workflow, environment, resource, planning or
// execution subpackage; aliases here keep application ports concise.
package domain

import (
	"github.com/UFFeScience/akoflow/internal/domain/environment"
	"github.com/UFFeScience/akoflow/internal/domain/execution"
	"github.com/UFFeScience/akoflow/internal/domain/planning"
	"github.com/UFFeScience/akoflow/internal/domain/resource"
	"github.com/UFFeScience/akoflow/internal/domain/workflow"
)

type EnvironmentVersionStatus = environment.EnvironmentVersionStatus
type EnvironmentStatus = environment.EnvironmentStatus
type ConnectionType = environment.ConnectionType
type Environment = environment.Environment
type EnvironmentVersion = environment.EnvironmentVersion
type EnvironmentRuntime = environment.EnvironmentRuntime
type EnvironmentConnection = environment.EnvironmentConnection
type EnvironmentCapabilities = environment.Capabilities
type DiscoveryRun = environment.DiscoveryRun
type EnvironmentDefinition = environment.Definition

const (
	EnvironmentVersionDraft     = environment.EnvironmentVersionDraft
	EnvironmentVersionPublished = environment.EnvironmentVersionPublished
	EnvironmentVersionRetired   = environment.EnvironmentVersionRetired
	EnvironmentDefined          = environment.EnvironmentDefined
	EnvironmentConnecting       = environment.EnvironmentConnecting
	EnvironmentConnected        = environment.EnvironmentConnected
	EnvironmentDiscovering      = environment.EnvironmentDiscovering
	EnvironmentReady            = environment.EnvironmentReady
	EnvironmentDegraded         = environment.EnvironmentDegraded
	EnvironmentUnreachable      = environment.EnvironmentUnreachable
	ConnectionSSH               = environment.ConnectionSSH
	ConnectionKubernetes        = environment.ConnectionKubernetes
	ConnectionCloud             = environment.ConnectionCloud
	ConnectionLocal             = environment.ConnectionLocal
	ConnectionAgent             = environment.ConnectionAgent
)

type ResourceType = resource.ResourceType
type ResourceExecutionTarget = resource.ResourceExecutionTarget
type Resource = resource.Resource
type ResourceRelation = resource.ResourceRelation
type ResourceRelationType = resource.ResourceRelationType
type ResourceSnapshot = resource.ResourceSnapshot
type NetworkLink = resource.NetworkLink
type NetworkTopology = resource.NetworkTopology

const (
	ResourceCluster               = resource.ResourceCluster
	ResourceNodePool              = resource.ResourceNodePool
	ResourceKubernetesMachine     = resource.ResourceKubernetesMachine
	ResourceHPCPartition          = resource.ResourceHPCPartition
	ResourceHPCMachine            = resource.ResourceHPCMachine
	ResourceCloudVM               = resource.ResourceCloudVM
	ResourceFogDevice             = resource.ResourceFogDevice
	ResourceLocalMachine          = resource.ResourceLocalMachine
	ResourceServerlessPlatform    = resource.ResourceServerlessPlatform
	ResourceServerlessFunction    = resource.ResourceServerlessFunction
	ResourceBatchQueue            = resource.ResourceBatchQueue
	ResourceKubernetesNamespace   = resource.ResourceKubernetesNamespace
	ResourceSlurmReservation      = resource.ResourceSlurmReservation
	ExecutionTargetBatch          = resource.ExecutionTargetBatch
	ExecutionTargetDirect         = resource.ExecutionTargetDirect
	ResourceRelationContains      = resource.ResourceRelationContains
	ResourceRelationMemberOf      = resource.ResourceRelationMemberOf
	ResourceRelationAccessibleVia = resource.ResourceRelationAccessibleVia
)

type WorkflowVersion = workflow.WorkflowVersion
type WorkflowDefinition = workflow.Definition
type ActivityType = workflow.ActivityType
type Activity = workflow.Activity
type ActivityKind = workflow.ActivityKind
type ActivityCapability = workflow.ActivityCapability
type ActivityCommand = workflow.ActivityCommand
type ActivityResources = workflow.ActivityResources
type ServiceSpec = workflow.ServiceSpec
type ActivitySimulation = workflow.ActivitySimulation
type ActivityPolicy = workflow.ActivityPolicy
type ActivityDependency = workflow.ActivityDependency
type ActivityDataDependency = workflow.ActivityDataDependency
type ActivityResourceProfile = workflow.ActivityResourceProfile

type PlanningSource = planning.PlanningSource
type ExecutionMode = planning.ExecutionMode
type PredictedMetrics = planning.PredictedMetrics
type SchedulePlan = planning.SchedulePlan
type PlanAssignment = planning.PlanAssignment
type PlanningRequest = planning.PlanningRequest

const (
	PlanningSourcePlugin          = planning.PlanningSourcePlugin
	PlanningSourceImported        = planning.PlanningSourceImported
	ExecutionModeReal             = planning.ExecutionModeReal
	ExecutionModeSimulation       = planning.ExecutionModeSimulation
	ExecutionModeInteractive      = planning.ExecutionModeInteractive
	ActivityKindTask              = workflow.ActivityKindTask
	ActivityKindService           = workflow.ActivityKindService
	ActivityKindInteractive       = workflow.ActivityKindInteractive
	ActivityCapabilityReal        = workflow.ActivityCapabilityReal
	ActivityCapabilitySimulation  = workflow.ActivityCapabilitySimulation
	ActivityCapabilityInteractive = workflow.ActivityCapabilityInteractive
)

type ExecutionRunStatus = execution.ExecutionRunStatus
type TaskExecutionStatus = execution.TaskExecutionStatus
type ExecutionRun = execution.ExecutionRun
type TaskExecution = execution.TaskExecution
type ExecutionMetrics = execution.ExecutionMetrics
type ExecutionTrace = execution.ExecutionTrace
type DataTransfer = execution.DataTransfer
type ActivityExecutionContext = execution.ActivityExecutionContext
type ActivityHandle = execution.ActivityHandle
type ActivityHandleStatus = execution.ActivityHandleStatus
type ArtifactChange = execution.ArtifactChange
type ArtifactObservation = execution.ArtifactObservation
type ArtifactManifest = execution.ArtifactManifest
type LifecycleObservation = execution.LifecycleObservation
type ArtifactSummary = execution.ArtifactSummary
type StorageType = resource.StorageType
type DataLocationStatus = execution.DataLocationStatus
type StorageResource = resource.StorageResource
type StorageRuntimeBinding = resource.StorageRuntimeBinding
type DataObject = execution.DataObject
type DataObjectInstance = execution.DataObjectInstance
type DataLocation = execution.DataLocation

const (
	ExecutionRunCreated   = execution.ExecutionRunCreated
	ExecutionRunRunning   = execution.ExecutionRunRunning
	ExecutionRunCompleted = execution.ExecutionRunCompleted
	ExecutionRunFailed    = execution.ExecutionRunFailed
	TaskBlocked           = execution.TaskBlocked
	TaskReady             = execution.TaskReady
	TaskPreparing         = execution.TaskPreparing
	TaskRunning           = execution.TaskRunning
	TaskCompleted         = execution.TaskCompleted
	TaskFailed            = execution.TaskFailed
	TaskCancelled         = execution.TaskCancelled
	HandleStarting        = execution.HandleStarting
	HandleRunning         = execution.HandleRunning
	HandleCompleted       = execution.HandleCompleted
	HandleFailed          = execution.HandleFailed
	HandleStopped         = execution.HandleStopped
	ArtifactCreated       = execution.ArtifactCreated
	ArtifactModified      = execution.ArtifactModified
	ArtifactDeleted       = execution.ArtifactDeleted
	StorageLocal          = resource.StorageLocal
	StoragePVC            = resource.StoragePVC
	StorageNFS            = resource.StorageNFS
	StorageS3             = resource.StorageS3
	StorageLustre         = resource.StorageLustre
	DataLocationEphemeral = execution.DataLocationEphemeral
	DataLocationStaging   = execution.DataLocationStaging
	DataLocationAvailable = execution.DataLocationAvailable
	DataLocationFailed    = execution.DataLocationFailed
	DataLocationDeleted   = execution.DataLocationDeleted
)
