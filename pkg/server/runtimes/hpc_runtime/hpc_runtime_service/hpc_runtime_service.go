package hpc_runtime_service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ovvesley/akoflow/pkg/server/config"
	"github.com/ovvesley/akoflow/pkg/server/connector/connector_hpc"
	"github.com/ovvesley/akoflow/pkg/server/database/repository/activity_repository"
	"github.com/ovvesley/akoflow/pkg/server/database/repository/runtime_repository"
	"github.com/ovvesley/akoflow/pkg/server/database/repository/workflow_repository"
	"github.com/ovvesley/akoflow/pkg/server/entities/workflow_activity_entity"
	"github.com/ovvesley/akoflow/pkg/server/entities/workflow_entity"
	"github.com/ovvesley/akoflow/pkg/server/runtimes/singularity_runtime/singularity_runtime_service"
)

type HPCRuntimeService struct {
	makeSingularityActivity      singularity_runtime_service.MakeSingularityActivityService
	makeSBatchHPCRuntimeActivity MakeSBatchHPCRuntimeActivityService

	activityRepository activity_repository.IActivityRepository
	workflowRepository workflow_repository.IWorkflowRepository
	runtimeRepository  runtime_repository.IRuntimeRepository

	connectorHPCRuntime connector_hpc.IConnectorHPCRuntime

	runtimeName string
	runtimeType string
}

func (s *HPCRuntimeService) SetRuntimeName(runtimeName string) *HPCRuntimeService {
	s.runtimeName = runtimeName
	return s
}

func (s *HPCRuntimeService) SetRuntimeType(runtimeType string) *HPCRuntimeService {
	s.runtimeType = runtimeType
	return s
}

func New() *HPCRuntimeService {
	return &HPCRuntimeService{
		makeSingularityActivity: singularity_runtime_service.NewMakeSingularityActivityService(),

		activityRepository: config.App().Repository.ActivityRepository,
		workflowRepository: config.App().Repository.WorkflowRepository,
		runtimeRepository:  config.App().Repository.RuntimeRepository,

		connectorHPCRuntime: config.App().Connector.HPCRuntimeConnector,
	}
}

func (s *HPCRuntimeService) ApplyJob(workflowID int, activityID int) string {
	wfa, err := s.activityRepository.Find(activityID)
	wf, _ := s.workflowRepository.Find(wfa.WorkflowId)

	if err != nil {
		config.App().Logger.Infof("WORKER: Activity not found %d", activityID)
		return ""
	}

	if wfa.Status != activity_repository.StatusCreated {
		config.App().Logger.Infof("WORKER: Activity not eligible for execution %d", activityID)
		return ""
	}

	if wf.Status == workflow_repository.StatusCreated {
		config.App().Logger.Infof("WORKER: Initial activity. Setup Data and Environment.")
		s.applyWorkflowInRuntime(wf, wfa)
	}

	runtime, err := s.runtimeRepository.GetByName(wf.GetRuntimeId()[0])
	if err != nil {
		config.App().Logger.Infof("WORKER: Error getting runtime from database.")
		return ""
	}

	singularitySystemCall := s.makeSingularityActivity.MakeContainerCommandActivityToHPC(wf, wfa)
	sBatchHPCRuntimeSystemCall := s.makeSBatchHPCRuntimeActivity.
		SetRuntime(*runtime).
		SetSingularityCommand(singularitySystemCall).
		Handle(wf, wfa)

	fmt.Println("PID: ", singularitySystemCall, sBatchHPCRuntimeSystemCall)

	connected, err := s.connectorHPCRuntime.IsVPNConnected()

	if err != nil {
		config.App().Logger.Error("WORKER: Error checking VPN connection to HPCRuntime.")
		return ""
	}

	if !connected {
		config.App().Logger.Error("WORKER: VPN is not connected to HPCRuntime. Continue to the next activity.")
		return ""
	}

	output, _ := s.connectorHPCRuntime.SetRuntime(*runtime).RunCommandWithOutputRemote(sBatchHPCRuntimeSystemCall)

	pid, err := s.extractJobID(output)

	if err != nil {
		config.App().Logger.Infof("WORKER: Error extracting job ID %s", strings.TrimSpace(wfa.GetProcId()))
		return ""
	}

	fmt.Println("PID: ", pid)

	err = s.workflowRepository.UpdateStatus(wfa.WorkflowId, workflow_repository.StatusRunning)

	if err != nil {
		config.App().Logger.Infof("WORKER: Error updating workflow status %d", wfa.WorkflowId)
		return ""
	}

	_ = s.activityRepository.UpdateStatus(wfa.Id, activity_repository.StatusRunning)
	err = s.activityRepository.UpdateProcID(wfa.Id, pid)

	if err != nil {
		config.App().Logger.Infof("WORKER: Error updating activity status %d", activityID)
		return ""
	}

	config.App().Logger.Infof("WORKER: Running singularity command %s", singularitySystemCall)

	return output

}

func (s *HPCRuntimeService) applyWorkflowInRuntime(wf workflow_entity.Workflow, wfa workflow_activity_entity.WorkflowActivities) {
	config.App().Logger.Infof("WORKER: Apply workflow in HPCRuntime")

	s.updateWorkflowAndActivityStatus(wfa)

	s.syncWorkflowVolumes(wf)
}

func (s *HPCRuntimeService) updateWorkflowAndActivityStatus(wfa workflow_activity_entity.WorkflowActivities) {
	_ = s.activityRepository.UpdateStatus(wfa.Id, activity_repository.StatusSyncing)
}

func (s *HPCRuntimeService) syncWorkflowVolumes(wf workflow_entity.Workflow) {
	volumes := wf.GetVolumes()
	commands := []string{}

	runtime, err := s.runtimeRepository.GetByName(wf.GetRuntimeId()[0])
	if err != nil {
		config.App().Logger.Infof("WORKER: Error getting runtime from database.")
		return
	}

	for _, volume := range volumes {
		command1, err := s.connectorHPCRuntime.BuildRemoteCommand(*runtime, fmt.Sprintf("mkdir -p %s", volume.GetRemotePath()))
		if err != nil {
			config.App().Logger.Infof("WORKER: Error building remote command.")
			return
		}

		var command2, command3 string

		command2 = fmt.Sprintf("rsync -ah --progress %s %s@%s:%s",
			volume.GetLocalPath(),
			runtime.GetCurrentRuntimeMetadata("USER"),
			runtime.GetCurrentRuntimeMetadata("HOST_CLUSTER"),
			volume.GetRemotePath(),
		)

		command3 = fmt.Sprintf("rsync -ah --progress %s@%s:%s %s",
			runtime.GetCurrentRuntimeMetadata("USER"),
			runtime.GetCurrentRuntimeMetadata("HOST_CLUSTER"),
			volume.GetRemotePath(),
			volume.GetLocalPath(),
		)

		fullCommands := fmt.Sprintf("%s && %s && %s", command1, command2, command3)

		commands = append(commands, fullCommands)
	}

	s.connectorHPCRuntime.ExecuteMultiplesCommand(commands)

}

func (s *HPCRuntimeService) extractJobID(outputCommand string) (string, error) {
	reOutput := regexp.MustCompile(`(?m)(\d+)`)

	matchOutput := reOutput.FindStringSubmatch(outputCommand)

	var logsOutput string

	if len(matchOutput) > 1 {
		logsOutput = strings.TrimSpace(matchOutput[1])
	}

	return logsOutput, nil
}

func (s *HPCRuntimeService) VerifyActivitiesWasFinished(workflow workflow_entity.Workflow) bool {
	config.App().Logger.Infof("WORKER: Verify activities was finished in HPCRuntime")

	finishedActivities := s.getFinishedActivities(workflow)
	if len(finishedActivities) == 0 {
		return true
	}

	syncNeeded := false
	for _, activity := range workflow.Spec.Activities {
		if activity.GetRuntimeId() != s.runtimeName {
			continue
		}

		if _, ok := finishedActivities[activity.Id]; !ok {
			continue
		}

		if s.handleVerifyActivityWasFinished(activity) {
			syncNeeded = true
		}
	}

	if syncNeeded {
		s.syncWorkflowVolumes(workflow)
	}

	return true
}

func (s *HPCRuntimeService) getFinishedActivities(workflow workflow_entity.Workflow) map[int]struct{} {
	finishedActivities := make(map[int]struct{})

	runtimeName := s.runtimeName
	if runtimeName == "" && len(workflow.GetRuntimeId()) > 0 {
		runtimeName = workflow.GetRuntimeId()[0]
	}

	if runtimeName == "" {
		return finishedActivities
	}

	runtime, err := s.runtimeRepository.GetByName(runtimeName)
	if err != nil {
		config.App().Logger.Infof("WORKER: Error getting runtime from database.")
		return finishedActivities
	}

	sharedMountPath := workflow.GetMountPath()
	if sharedMountPath == "" {
		sharedMountPath = runtime.GetCurrentRuntimeMetadata("MOUNT_PATH")
	}

	commandVerify := fmt.Sprintf("find %s -maxdepth 1 -type f -name 'akoflow_finished_%d_*.txt' -print", sharedMountPath, workflow.GetId())
	output, err := s.connectorHPCRuntime.SetRuntime(*runtime).RunCommandWithOutputRemote(commandVerify)
	if err != nil {
		config.App().Logger.Infof("WORKER: Error listing finished activities for workflow %d", workflow.GetId())
		return finishedActivities
	}

	re := regexp.MustCompile(fmt.Sprintf(`akoflow_finished_%d_(\d+)\.txt`, workflow.GetId()))
	for _, match := range re.FindAllStringSubmatch(output, -1) {
		if len(match) < 2 {
			continue
		}

		activityID, err := strconv.Atoi(strings.TrimSpace(match[1]))
		if err != nil {
			continue
		}

		finishedActivities[activityID] = struct{}{}
	}

	return finishedActivities
}

func (s *HPCRuntimeService) handleVerifyActivityWasFinished(activity workflow_activity_entity.WorkflowActivities) bool {
	println("Verifying activity: ", activity.Name, " with id: ", activity.Id)

	wfaDatabase, _ := s.activityRepository.Find(activity.Id)

	if wfaDatabase.Status == activity_repository.StatusFinished {
		return false
	}

	if wfaDatabase.Status == activity_repository.StatusCreated {
		return false
	}

	if wfaDatabase.GetProcId() == "" {
		config.App().Logger.Infof("WORKER: Activity %d has no process ID", activity.Id)
		return false
	}

	config.App().Logger.Infof("WORKER: Activity %d finished", activity.Id)
	_ = s.activityRepository.UpdateStatus(activity.Id, activity_repository.StatusFinished)
	return true

}

func extractField(pattern, text string) (string, error) {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return "", fmt.Errorf("field not found: %s", pattern)
	}
	return match[1], nil
}

func (s *HPCRuntimeService) HealthCheck(runtimeName string) bool {
	config.App().Logger.Infof("WORKER: Health check HPCRuntime")

	connected, err := s.connectorHPCRuntime.IsVPNConnected()
	if err != nil {
		config.App().Logger.Error("WORKER: Error checking VPN connection to HPCRuntime.")
		return false
	}
	if !connected {
		config.App().Logger.Error("WORKER: VPN is not connected to HPCRuntime.")
		return false
	}

	runtime, err := s.runtimeRepository.GetByName(runtimeName)

	if err != nil {
		config.App().Logger.Error("WORKER: Error getting runtime from database.")
		return false
	}

	if runtime == nil {
		config.App().Logger.Error("WORKER: Runtime not found in database.")
		return false
	}

	command := fmt.Sprintf("sinfo -p %s", runtime.GetCurrentRuntimeMetadata("QUEUE"))
	output, err := s.connectorHPCRuntime.SetRuntime(*runtime).RunCommandWithOutputRemote(command)

	if err != nil {
		runtime.Status = runtime_repository.STATUS_NOT_READY
		config.App().Logger.Error("WORKER: Error running command in HPCRuntime.")
		return false
	}

	if strings.Contains(output, "No nodes available") {
		s.runtimeRepository.UpdateStatus(runtime, runtime_repository.STATUS_NOT_READY)
		config.App().Logger.Error("WORKER: No nodes available in HPCRuntime.")
		return false
	}

	s.runtimeRepository.UpdateStatus(runtime, runtime_repository.STATUS_READY)

	return true

}
