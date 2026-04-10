package orchestration

import (
	"context"
	"fmt"
)

type Provisioner interface {
	Name() RuntimeBackend
	Provision(ctx context.Context, spec RuntimeSpec) (*ProvisionedRuntime, error)
	Start(ctx context.Context, lease RuntimeLease) (*ProvisionedRuntime, error)
	Stop(ctx context.Context, lease RuntimeLease) error
	Delete(ctx context.Context, lease RuntimeLease) error
}

type ProvisionerRegistry struct {
	provisioners map[RuntimeBackend]Provisioner
}

func NewProvisionerRegistry(provisioners ...Provisioner) *ProvisionerRegistry {
	r := &ProvisionerRegistry{provisioners: map[RuntimeBackend]Provisioner{}}
	for _, provisioner := range provisioners {
		if provisioner == nil {
			continue
		}
		r.provisioners[provisioner.Name()] = provisioner
	}
	return r
}

func (r *ProvisionerRegistry) Get(backend string) (Provisioner, error) {
	p, ok := r.provisioners[RuntimeBackend(backend)]
	if !ok {
		return nil, fmt.Errorf("no provisioner registered for backend %q", backend)
	}
	return p, nil
}
