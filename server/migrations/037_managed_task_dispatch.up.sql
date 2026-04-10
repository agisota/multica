ALTER TABLE agent
    ALTER COLUMN runtime_id DROP NOT NULL;

ALTER TABLE agent_task_queue
    ALTER COLUMN runtime_id DROP NOT NULL;

ALTER TABLE agent_task_queue
    ADD COLUMN lease_id UUID REFERENCES runtime_lease(id) ON DELETE SET NULL,
    ADD COLUMN execution_backend TEXT NOT NULL DEFAULT 'local'
        CHECK (execution_backend IN ('local', 'modal', 'daytona')),
    ADD COLUMN dispatch_state TEXT NOT NULL DEFAULT 'queued'
        CHECK (dispatch_state IN ('queued', 'dispatching', 'dispatched', 'running', 'completed', 'failed', 'cancelled')),
    ADD COLUMN external_execution_id TEXT;

CREATE TABLE runtime_task_execution (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES agent_task_queue(id) ON DELETE CASCADE,
    lease_id UUID REFERENCES runtime_lease(id) ON DELETE SET NULL,
    backend TEXT NOT NULL CHECK (backend IN ('local', 'modal', 'daytona')),
    external_execution_id TEXT,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'dispatching', 'running', 'completed', 'failed', 'cancelled', 'timed_out')),
    logs TEXT,
    result JSONB,
    error TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    submitted_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    last_polled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE runtime_usage_event (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    task_id UUID REFERENCES agent_task_queue(id) ON DELETE SET NULL,
    execution_id UUID REFERENCES runtime_task_execution(id) ON DELETE SET NULL,
    lease_id UUID REFERENCES runtime_lease(id) ON DELETE SET NULL,
    user_id UUID REFERENCES "user"(id) ON DELETE SET NULL,
    provider TEXT NOT NULL,
    backend TEXT NOT NULL CHECK (backend IN ('local', 'modal', 'daytona')),
    event_type TEXT NOT NULL
        CHECK (event_type IN ('task_started', 'task_completed', 'task_failed', 'runtime_allocated', 'runtime_released')),
    amount_cents BIGINT NOT NULL DEFAULT 0,
    quantity NUMERIC NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_agent_task_queue_runtime_pending_local
    ON agent_task_queue(runtime_id, priority DESC, created_at ASC)
    WHERE execution_backend = 'local' AND status IN ('queued', 'dispatched');

CREATE INDEX idx_agent_task_queue_managed_dispatch
    ON agent_task_queue(execution_backend, dispatch_state, created_at ASC)
    WHERE execution_backend <> 'local' AND dispatch_state IN ('queued', 'dispatching', 'dispatched', 'running');

CREATE INDEX idx_agent_task_queue_lease_id ON agent_task_queue(lease_id);

CREATE INDEX idx_agent_task_queue_external_execution_id
    ON agent_task_queue(external_execution_id)
    WHERE external_execution_id IS NOT NULL;

CREATE INDEX idx_runtime_task_execution_task_id ON runtime_task_execution(task_id);
CREATE INDEX idx_runtime_task_execution_lease_id ON runtime_task_execution(lease_id);
CREATE INDEX idx_runtime_task_execution_external_id
    ON runtime_task_execution(external_execution_id)
    WHERE external_execution_id IS NOT NULL;

CREATE INDEX idx_runtime_usage_event_workspace_created_at
    ON runtime_usage_event(workspace_id, created_at DESC);

CREATE INDEX idx_runtime_usage_event_lease_id ON runtime_usage_event(lease_id);
