CREATE TABLE runtime_policy (
    workspace_id UUID PRIMARY KEY REFERENCES workspace(id) ON DELETE CASCADE,
    shared_pool_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    private_runtime_scope TEXT NOT NULL DEFAULT 'user' CHECK (private_runtime_scope IN ('user', 'workspace', 'project')),
    default_runtime_placement TEXT NOT NULL DEFAULT 'shared' CHECK (default_runtime_placement IN ('shared', 'private', 'dedicated')),
    default_provider TEXT NOT NULL DEFAULT 'codex',
    allow_daytona BOOLEAN NOT NULL DEFAULT TRUE,
    allow_modal BOOLEAN NOT NULL DEFAULT TRUE,
    auto_stop_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    auto_delete_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    idle_ttl_minutes INTEGER NOT NULL DEFAULT 120,
    max_private_runtimes_per_user INTEGER NOT NULL DEFAULT 2,
    max_shared_runtimes INTEGER NOT NULL DEFAULT 8,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE runtime_billing_account (
    workspace_id UUID PRIMARY KEY REFERENCES workspace(id) ON DELETE CASCADE,
    billing_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    currency TEXT NOT NULL DEFAULT 'usd',
    hard_limit_cents BIGINT NOT NULL DEFAULT 0,
    soft_limit_cents BIGINT NOT NULL DEFAULT 0,
    shared_hourly_rate_cents INTEGER NOT NULL DEFAULT 0,
    private_hourly_rate_cents INTEGER NOT NULL DEFAULT 0,
    token_markup_percent INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE runtime_lease (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    runtime_id UUID REFERENCES agent_runtime(id) ON DELETE SET NULL,
    owner_user_id UUID REFERENCES "user"(id) ON DELETE SET NULL,
    project_id UUID REFERENCES project(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'workspace' CHECK (scope IN ('user', 'workspace', 'project')),
    placement TEXT NOT NULL DEFAULT 'shared' CHECK (placement IN ('shared', 'private', 'dedicated')),
    provider TEXT NOT NULL,
    backend TEXT NOT NULL DEFAULT 'local' CHECK (backend IN ('local', 'daytona', 'modal')),
    state TEXT NOT NULL DEFAULT 'requested' CHECK (state IN ('requested', 'provisioning', 'active', 'stopped', 'failed', 'deleted')),
    external_ref TEXT,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    billing JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_runtime_lease_workspace_id ON runtime_lease(workspace_id);
CREATE INDEX idx_runtime_lease_owner_user_id ON runtime_lease(owner_user_id);
CREATE INDEX idx_runtime_lease_project_id ON runtime_lease(project_id);
CREATE INDEX idx_runtime_lease_state ON runtime_lease(state);
CREATE INDEX idx_runtime_lease_runtime_id ON runtime_lease(runtime_id);
