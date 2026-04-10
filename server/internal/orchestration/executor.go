package orchestration

import (
	"context"
	"fmt"
)

type Executor interface {
	Name() RuntimeBackend
	Mode() RuntimeExecutionMode
	Submit(ctx context.Context, req ManagedTaskRequest) (*ManagedTaskHandle, error)
	Poll(ctx context.Context, externalID string) (*ManagedTaskResult, error)
	Cancel(ctx context.Context, externalID string) error
}

type ExecutorRegistry struct {
	executors map[RuntimeBackend]Executor
}

func NewExecutorRegistry(executors ...Executor) *ExecutorRegistry {
	registry := &ExecutorRegistry{executors: map[RuntimeBackend]Executor{}}
	for _, executor := range executors {
		if executor == nil {
			continue
		}
		registry.executors[executor.Name()] = executor
	}
	return registry
}

func (r *ExecutorRegistry) Get(backend string) (Executor, error) {
	executor, ok := r.executors[RuntimeBackend(backend)]
	if !ok {
		return nil, fmt.Errorf("no executor registered for backend %q", backend)
	}
	return executor, nil
}

type localExecutor struct{}

func NewLocalExecutor() Executor {
	return &localExecutor{}
}

func (e *localExecutor) Name() RuntimeBackend {
	return RuntimeBackendLocal
}

func (e *localExecutor) Mode() RuntimeExecutionMode {
	return RuntimeExecutionModeLocal
}

func (e *localExecutor) Submit(context.Context, ManagedTaskRequest) (*ManagedTaskHandle, error) {
	return nil, fmt.Errorf("local execution is daemon-managed and does not support managed dispatch")
}

func (e *localExecutor) Poll(context.Context, string) (*ManagedTaskResult, error) {
	return nil, fmt.Errorf("local execution is daemon-managed and does not support managed polling")
}

func (e *localExecutor) Cancel(context.Context, string) error {
	return fmt.Errorf("local execution is daemon-managed and does not support managed cancellation")
}
