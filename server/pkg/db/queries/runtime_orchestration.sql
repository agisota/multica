-- name: GetRuntimePolicy :one
SELECT * FROM runtime_policy
WHERE workspace_id = $1;

-- name: UpsertRuntimePolicy :one
INSERT INTO runtime_policy (
    workspace_id,
    shared_pool_enabled,
    private_runtime_scope,
    default_runtime_placement,
    default_provider,
    allow_daytona,
    allow_modal,
    auto_stop_enabled,
    auto_delete_enabled,
    idle_ttl_minutes,
    max_private_runtimes_per_user,
    max_shared_runtimes,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (workspace_id)
DO UPDATE SET
    shared_pool_enabled = EXCLUDED.shared_pool_enabled,
    private_runtime_scope = EXCLUDED.private_runtime_scope,
    default_runtime_placement = EXCLUDED.default_runtime_placement,
    default_provider = EXCLUDED.default_provider,
    allow_daytona = EXCLUDED.allow_daytona,
    allow_modal = EXCLUDED.allow_modal,
    auto_stop_enabled = EXCLUDED.auto_stop_enabled,
    auto_delete_enabled = EXCLUDED.auto_delete_enabled,
    idle_ttl_minutes = EXCLUDED.idle_ttl_minutes,
    max_private_runtimes_per_user = EXCLUDED.max_private_runtimes_per_user,
    max_shared_runtimes = EXCLUDED.max_shared_runtimes,
    metadata = EXCLUDED.metadata,
    updated_at = now()
RETURNING *;

-- name: GetRuntimeBillingAccount :one
SELECT * FROM runtime_billing_account
WHERE workspace_id = $1;

-- name: UpsertRuntimeBillingAccount :one
INSERT INTO runtime_billing_account (
    workspace_id,
    billing_enabled,
    currency,
    hard_limit_cents,
    soft_limit_cents,
    shared_hourly_rate_cents,
    private_hourly_rate_cents,
    token_markup_percent,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (workspace_id)
DO UPDATE SET
    billing_enabled = EXCLUDED.billing_enabled,
    currency = EXCLUDED.currency,
    hard_limit_cents = EXCLUDED.hard_limit_cents,
    soft_limit_cents = EXCLUDED.soft_limit_cents,
    shared_hourly_rate_cents = EXCLUDED.shared_hourly_rate_cents,
    private_hourly_rate_cents = EXCLUDED.private_hourly_rate_cents,
    token_markup_percent = EXCLUDED.token_markup_percent,
    metadata = EXCLUDED.metadata,
    updated_at = now()
RETURNING *;

-- name: ListRuntimeLeases :many
SELECT * FROM runtime_lease
WHERE workspace_id = $1 AND state <> 'deleted'
ORDER BY created_at DESC;

-- name: ListRuntimeLeasesByOwner :many
SELECT * FROM runtime_lease
WHERE workspace_id = $1
  AND owner_user_id = $2
  AND state <> 'deleted'
ORDER BY created_at DESC;

-- name: GetRuntimeLeaseInWorkspace :one
SELECT * FROM runtime_lease
WHERE id = $1 AND workspace_id = $2;

-- name: CreateRuntimeLease :one
INSERT INTO runtime_lease (
    workspace_id,
    runtime_id,
    owner_user_id,
    project_id,
    name,
    scope,
    placement,
    provider,
    backend,
    state,
    external_ref,
    config,
    billing,
    metadata,
    last_used_at,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING *;

-- name: UpdateRuntimeLease :one
UPDATE runtime_lease
SET
    runtime_id = COALESCE(sqlc.narg('runtime_id'), runtime_id),
    owner_user_id = COALESCE(sqlc.narg('owner_user_id'), owner_user_id),
    project_id = COALESCE(sqlc.narg('project_id'), project_id),
    name = COALESCE(sqlc.narg('name'), name),
    scope = COALESCE(sqlc.narg('scope'), scope),
    placement = COALESCE(sqlc.narg('placement'), placement),
    provider = COALESCE(sqlc.narg('provider'), provider),
    backend = COALESCE(sqlc.narg('backend'), backend),
    state = COALESCE(sqlc.narg('state'), state),
    external_ref = COALESCE(sqlc.narg('external_ref'), external_ref),
    config = COALESCE(sqlc.narg('config'), config),
    billing = COALESCE(sqlc.narg('billing'), billing),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    last_used_at = COALESCE(sqlc.narg('last_used_at'), last_used_at),
    expires_at = COALESCE(sqlc.narg('expires_at'), expires_at),
    updated_at = now()
WHERE id = $1 AND workspace_id = $2
RETURNING *;
