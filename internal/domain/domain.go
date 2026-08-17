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
type Resource = resource.Resource
type ResourceSnapshot = resource.ResourceSnapshot
type NetworkLink = resource.NetworkLink

const (
	ResourceCluster             = resource.ResourceCluster
	ResourceNodePool            = resource.ResourceNodePool
	ResourceKubernetesMachine   = resource.ResourceKubernetesMachine
	ResourceHPCPartition        = resource.ResourceHPCPartition
	ResourceHPCMachine          = resource.ResourceHPCMachine
	ResourceCloudVM             = resource.ResourceCloudVM
	ResourceFogDevice           = resource.ResourceFogDevice
	ResourceLocalMachine        = resource.ResourceLocalMachine
	ResourceServerlessPlatform  = resource.ResourceServerlessPlatform
	ResourceServerlessFunction  = resource.ResourceServerlessFunction
	ResourceBatchQueue          = resource.ResourceBatchQueue
	ResourceKubernetesNamespace = resource.ResourceKubernetesNamespace
	ResourceSlurmReservation    = resource.ResourceSlurmReservation
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
)
