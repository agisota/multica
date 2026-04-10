package orchestration

import (
	"encoding/json"
	"time"
)

type PrivateRuntimeScope string

const (
	PrivateRuntimeScopeUser      PrivateRuntimeScope = "user"
	PrivateRuntimeScopeWorkspace PrivateRuntimeScope = "workspace"
	PrivateRuntimeScopeProject   PrivateRuntimeScope = "project"
)

type RuntimePlacement string

const (
	RuntimePlacementShared    RuntimePlacement = "shared"
	RuntimePlacementPrivate   RuntimePlacement = "private"
	RuntimePlacementDedicated RuntimePlacement = "dedicated"
)

type RuntimeBackend string

const (
	RuntimeBackendLocal   RuntimeBackend = "local"
	RuntimeBackendDaytona RuntimeBackend = "daytona"
	RuntimeBackendModal   RuntimeBackend = "modal"
)

type RuntimeExecutionMode string

const (
	RuntimeExecutionModeLocal     RuntimeExecutionMode = "local"
	RuntimeExecutionModeDedicated RuntimeExecutionMode = "dedicated"
	RuntimeExecutionModeManaged   RuntimeExecutionMode = "managed"
)

type RuntimeLeaseState string

const (
	RuntimeLeaseStateRequested    RuntimeLeaseState = "requested"
	RuntimeLeaseStateProvisioning RuntimeLeaseState = "provisioning"
	RuntimeLeaseStateActive       RuntimeLeaseState = "active"
	RuntimeLeaseStateStopped      RuntimeLeaseState = "stopped"
	RuntimeLeaseStateFailed       RuntimeLeaseState = "failed"
	RuntimeLeaseStateDeleted      RuntimeLeaseState = "deleted"
)

type RuntimePolicy struct {
	WorkspaceID               string          `json:"workspace_id"`
	SharedPoolEnabled         bool            `json:"shared_pool_enabled"`
	PrivateRuntimeScope       string          `json:"private_runtime_scope"`
	DefaultRuntimePlacement   string          `json:"default_runtime_placement"`
	DefaultProvider           string          `json:"default_provider"`
	AllowDaytona              bool            `json:"allow_daytona"`
	AllowModal                bool            `json:"allow_modal"`
	AutoStopEnabled           bool            `json:"auto_stop_enabled"`
	AutoDeleteEnabled         bool            `json:"auto_delete_enabled"`
	IdleTTLMinutes            int             `json:"idle_ttl_minutes"`
	MaxPrivateRuntimesPerUser int             `json:"max_private_runtimes_per_user"`
	MaxSharedRuntimes         int             `json:"max_shared_runtimes"`
	Metadata                  json.RawMessage `json:"metadata"`
	CreatedAt                 time.Time       `json:"created_at"`
	UpdatedAt                 time.Time       `json:"updated_at"`
}

type RuntimeBillingAccount struct {
	WorkspaceID            string          `json:"workspace_id"`
	BillingEnabled         bool            `json:"billing_enabled"`
	Currency               string          `json:"currency"`
	HardLimitCents         int64           `json:"hard_limit_cents"`
	SoftLimitCents         int64           `json:"soft_limit_cents"`
	SharedHourlyRateCents  int             `json:"shared_hourly_rate_cents"`
	PrivateHourlyRateCents int             `json:"private_hourly_rate_cents"`
	TokenMarkupPercent     int             `json:"token_markup_percent"`
	Metadata               json.RawMessage `json:"metadata"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

type RuntimeLease struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	RuntimeID   *string         `json:"runtime_id"`
	OwnerUserID *string         `json:"owner_user_id"`
	ProjectID   *string         `json:"project_id"`
	Name        string          `json:"name"`
	Scope       string          `json:"scope"`
	Placement   string          `json:"placement"`
	Provider    string          `json:"provider"`
	Backend     string          `json:"backend"`
	State       string          `json:"state"`
	ExternalRef *string         `json:"external_ref"`
	Config      json.RawMessage `json:"config"`
	Billing     json.RawMessage `json:"billing"`
	Metadata    json.RawMessage `json:"metadata"`
	LastUsedAt  *time.Time      `json:"last_used_at"`
	ExpiresAt   *time.Time      `json:"expires_at"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type RuntimeSpec struct {
	LeaseID     string
	WorkspaceID string
	OwnerUserID string
	ProjectID   string
	Name        string
	Scope       string
	Placement   string
	Provider    string
	Backend     string
	Config      json.RawMessage
	Billing     json.RawMessage
}

type ProvisionedRuntime struct {
	ExternalRef string
	State       RuntimeLeaseState
	Metadata    map[string]any
}

type RuntimeBackendProfile struct {
	Backend                  RuntimeBackend
	ExecutionMode            RuntimeExecutionMode
	SupportsProvisioning     bool
	SupportsManagedExecution bool
	Experimental             bool
}

func BackendProfile(backend RuntimeBackend) RuntimeBackendProfile {
	switch backend {
	case RuntimeBackendModal:
		return RuntimeBackendProfile{
			Backend:                  backend,
			ExecutionMode:            RuntimeExecutionModeManaged,
			SupportsProvisioning:     true,
			SupportsManagedExecution: true,
		}
	case RuntimeBackendDaytona:
		return RuntimeBackendProfile{
			Backend:              backend,
			ExecutionMode:        RuntimeExecutionModeDedicated,
			SupportsProvisioning: true,
			Experimental:         true,
		}
	default:
		return RuntimeBackendProfile{
			Backend:              RuntimeBackendLocal,
			ExecutionMode:        RuntimeExecutionModeLocal,
			SupportsProvisioning: false,
		}
	}
}

type ManagedTaskStatus string

const (
	ManagedTaskStatusQueued    ManagedTaskStatus = "queued"
	ManagedTaskStatusRunning   ManagedTaskStatus = "running"
	ManagedTaskStatusCompleted ManagedTaskStatus = "completed"
	ManagedTaskStatusFailed    ManagedTaskStatus = "failed"
	ManagedTaskStatusCancelled ManagedTaskStatus = "cancelled"
	ManagedTaskStatusTimedOut  ManagedTaskStatus = "timed_out"
)

type ManagedTaskRequest struct {
	TaskID      string
	WorkspaceID string
	LeaseID     string
	AgentID     string
	Provider    string
	RepoURL     string
	Workdir     string
	Payload     json.RawMessage
	Metadata    map[string]any
}

type ManagedTaskHandle struct {
	ExternalID string
	Status     ManagedTaskStatus
	Metadata   map[string]any
}

type ManagedTaskResult struct {
	ExternalID string
	Status     ManagedTaskStatus
	Logs       string
	Result     json.RawMessage
	Error      string
	Metadata   map[string]any
}

type PlacementRequest struct {
	WorkspaceID      string
	UserID           string
	ProjectID        string
	Provider         string
	RequirePrivate   bool
	PreferredLeaseID string
	PreferredBackend RuntimeBackend
}

type PlacementDecision struct {
	LeaseID   string
	Backend   RuntimeBackend
	Placement RuntimePlacement
	Scope     PrivateRuntimeScope
	Reason    string
}

func DefaultRuntimePolicy(workspaceID string) RuntimePolicy {
	metadata, _ := json.Marshal(map[string]any{
		"primary_remote_backend": "modal",
		"fallback_backend":       "local",
		"daytona_status":         "experimental_disabled",
	})

	return RuntimePolicy{
		WorkspaceID:               workspaceID,
		SharedPoolEnabled:         true,
		PrivateRuntimeScope:       string(PrivateRuntimeScopeUser),
		DefaultRuntimePlacement:   string(RuntimePlacementShared),
		DefaultProvider:           "codex",
		AllowDaytona:              false,
		AllowModal:                true,
		AutoStopEnabled:           true,
		AutoDeleteEnabled:         false,
		IdleTTLMinutes:            120,
		MaxPrivateRuntimesPerUser: 2,
		MaxSharedRuntimes:         8,
		Metadata:                  metadata,
	}
}
