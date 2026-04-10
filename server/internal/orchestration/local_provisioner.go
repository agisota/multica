package orchestration

import "context"

type localProvisioner struct{}

func NewLocalProvisioner() Provisioner {
	return &localProvisioner{}
}

func (p *localProvisioner) Name() RuntimeBackend {
	return RuntimeBackendLocal
}

func (p *localProvisioner) Provision(_ context.Context, spec RuntimeSpec) (*ProvisionedRuntime, error) {
	return &ProvisionedRuntime{
		State: RuntimeLeaseStateActive,
		Metadata: map[string]any{
			"backend":           RuntimeBackendLocal,
			"execution_mode":    RuntimeExecutionModeLocal,
			"orchestration":     "daemon_claim_loop",
			"placement":         spec.Placement,
			"shared_pool":       spec.Placement == string(RuntimePlacementShared),
			"managed_execution": false,
		},
	}, nil
}

func (p *localProvisioner) Start(ctx context.Context, lease RuntimeLease) (*ProvisionedRuntime, error) {
	return p.Provision(ctx, RuntimeSpec{
		LeaseID:     lease.ID,
		WorkspaceID: lease.WorkspaceID,
		Name:        lease.Name,
		Scope:       lease.Scope,
		Placement:   lease.Placement,
		Provider:    lease.Provider,
		Backend:     lease.Backend,
		Config:      lease.Config,
		Billing:     lease.Billing,
	})
}

func (p *localProvisioner) Stop(context.Context, RuntimeLease) error {
	return nil
}

func (p *localProvisioner) Delete(context.Context, RuntimeLease) error {
	return nil
}
