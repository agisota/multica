package orchestration

import "context"

type Scheduler interface {
	Name() string
	Select(ctx context.Context, request PlacementRequest) (*PlacementDecision, error)
}

type DefaultScheduler struct{}

func NewDefaultScheduler() Scheduler {
	return &DefaultScheduler{}
}

func (s *DefaultScheduler) Name() string {
	return "default"
}

func (s *DefaultScheduler) Select(_ context.Context, request PlacementRequest) (*PlacementDecision, error) {
	if request.PreferredLeaseID == "" {
		return nil, nil
	}

	placement := RuntimePlacementShared
	scope := PrivateRuntimeScopeWorkspace
	if request.RequirePrivate {
		placement = RuntimePlacementPrivate
		scope = PrivateRuntimeScopeUser
	}

	backend := request.PreferredBackend
	if backend == "" {
		backend = RuntimeBackendModal
	}

	return &PlacementDecision{
		LeaseID:   request.PreferredLeaseID,
		Backend:   backend,
		Placement: placement,
		Scope:     scope,
		Reason:    "preferred lease override",
	}, nil
}
