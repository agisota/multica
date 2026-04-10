package orchestration

import "encoding/json"

type Manager struct {
	provisioners *ProvisionerRegistry
	executors    *ExecutorRegistry
	scheduler    Scheduler
}

func NewManager() *Manager {
	return &Manager{
		provisioners: NewProvisionerRegistry(
			NewLocalProvisioner(),
			NewDaytonaProvisionerFromEnv(),
			NewModalProvisionerFromEnv(),
		),
		executors: NewExecutorRegistry(
			NewLocalExecutor(),
			NewModalExecutor(),
		),
		scheduler: NewDefaultScheduler(),
	}
}

func (m *Manager) Registry() *ProvisionerRegistry {
	return m.provisioners
}

func (m *Manager) Provisioners() *ProvisionerRegistry {
	return m.provisioners
}

func (m *Manager) Executors() *ExecutorRegistry {
	return m.executors
}

func (m *Manager) Scheduler() Scheduler {
	return m.scheduler
}

func DefaultLeaseBilling(policy RuntimeBillingAccount, placement string) json.RawMessage {
	hourly := policy.SharedHourlyRateCents
	if placement == string(RuntimePlacementPrivate) || placement == string(RuntimePlacementDedicated) {
		hourly = policy.PrivateHourlyRateCents
	}

	payload, _ := json.Marshal(map[string]any{
		"currency":             policy.Currency,
		"billing_enabled":      policy.BillingEnabled,
		"hourly_rate_cents":    hourly,
		"token_markup_percent": policy.TokenMarkupPercent,
		"soft_limit_cents":     policy.SoftLimitCents,
		"hard_limit_cents":     policy.HardLimitCents,
	})
	return payload
}
