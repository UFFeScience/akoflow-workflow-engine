package execution

import (
	"context"
	"fmt"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type Controller struct {
	runtimes ports.RuntimeResolver
	handles  ports.ActivityExecutionStore
	data     ports.DataCatalog
}

func New(runtimes ports.RuntimeResolver, handles ports.ActivityExecutionStore, catalogs ...ports.DataCatalog) *Controller {
	controller := &Controller{runtimes: runtimes, handles: handles}
	if len(catalogs) > 0 {
		controller.data = catalogs[0]
	}
	return controller
}

func (s *Controller) Start(ctx context.Context, execution domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	if err := execution.Activity.Validate(); err != nil {
		return domain.ActivityHandle{}, err
	}
	capability, err := capabilityFor(execution.Run.Mode)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	if !execution.Activity.Supports(capability) {
		return domain.ActivityHandle{}, fmt.Errorf("activity %q does not support %q execution", execution.Activity.ID, execution.Run.Mode)
	}
	handleID := execution.Run.ID + ":" + execution.Activity.ID
	existing, err := s.handles.Find(ctx, handleID)
	if err != nil {
		return domain.ActivityHandle{}, fmt.Errorf("find activity handle: %w", err)
	}
	if existing != nil {
		return *existing, nil
	}
	if execution.RuntimeID == "" {
		return domain.ActivityHandle{}, fmt.Errorf("activity %q has no selected runtime", execution.Activity.ID)
	}
	runtime, err := s.runtimes.Resolve(execution.Run.Mode, execution.RuntimeID)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	handle, err := runtime.Start(ctx, execution)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	if handle.ActivityID != execution.Activity.ID || handle.RunID != execution.Run.ID {
		return domain.ActivityHandle{}, fmt.Errorf("runtime returned an invalid activity handle")
	}
	handle.ID = handleID
	if err := s.handles.Save(ctx, handle); err != nil {
		_ = runtime.Stop(ctx, handle)
		return domain.ActivityHandle{}, fmt.Errorf("persist activity handle: %w", err)
	}
	return handle, nil
}

func (s *Controller) Inspect(ctx context.Context, handleID string, mode domain.ExecutionMode) (*domain.ActivityHandle, error) {
	handle, err := s.handles.Find(ctx, handleID)
	if err != nil || handle == nil {
		return handle, err
	}
	runtime, err := s.runtimes.Resolve(mode, handle.RuntimeID)
	if err != nil {
		return nil, err
	}
	updated, err := runtime.Inspect(ctx, *handle)
	if err != nil {
		return nil, err
	}
	if s.data != nil && updated.Artifacts != nil &&
		(updated.Status == domain.HandleCompleted || updated.Status == domain.HandleFailed) {
		if catalogErr := s.data.CatalogArtifacts(ctx, updated); catalogErr != nil {
			if updated.Metadata == nil {
				updated.Metadata = make(map[string]any)
			}
			updated.Metadata["artifactCatalogError"] = catalogErr.Error()
		}
	}
	if err := s.handles.Save(ctx, updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Controller) Stop(ctx context.Context, handleID string, mode domain.ExecutionMode) error {
	handle, err := s.handles.Find(ctx, handleID)
	if err != nil {
		return err
	}
	if handle == nil {
		return fmt.Errorf("activity handle %q not found", handleID)
	}
	runtime, err := s.runtimes.Resolve(mode, handle.RuntimeID)
	if err != nil {
		return err
	}
	if err := runtime.Stop(ctx, *handle); err != nil {
		return err
	}
	handle.Status = domain.HandleStopped
	return s.handles.Save(ctx, *handle)
}

func capabilityFor(mode domain.ExecutionMode) (domain.ActivityCapability, error) {
	switch mode {
	case domain.ExecutionModeReal:
		return domain.ActivityCapabilityReal, nil
	case domain.ExecutionModeSimulation:
		return domain.ActivityCapabilitySimulation, nil
	case domain.ExecutionModeInteractive:
		return domain.ActivityCapabilityInteractive, nil
	default:
		return "", fmt.Errorf("unsupported execution mode %q", mode)
	}
}
