DROP INDEX IF EXISTS idx_runtime_usage_event_lease_id;
DROP INDEX IF EXISTS idx_runtime_usage_event_workspace_created_at;
DROP INDEX IF EXISTS idx_runtime_task_execution_external_id;
DROP INDEX IF EXISTS idx_runtime_task_execution_lease_id;
DROP INDEX IF EXISTS idx_runtime_task_execution_task_id;
DROP INDEX IF EXISTS idx_agent_task_queue_external_execution_id;
DROP INDEX IF EXISTS idx_agent_task_queue_lease_id;
DROP INDEX IF EXISTS idx_agent_task_queue_managed_dispatch;
DROP INDEX IF EXISTS idx_agent_task_queue_runtime_pending_local;

DROP TABLE IF EXISTS runtime_usage_event;
DROP TABLE IF EXISTS runtime_task_execution;

ALTER TABLE agent_task_queue
    DROP COLUMN IF EXISTS external_execution_id,
    DROP COLUMN IF EXISTS dispatch_state,
    DROP COLUMN IF EXISTS execution_backend,
    DROP COLUMN IF EXISTS lease_id;

DELETE FROM agent_task_queue WHERE runtime_id IS NULL;
DELETE FROM agent WHERE runtime_id IS NULL;

ALTER TABLE agent_task_queue
    ALTER COLUMN runtime_id SET NOT NULL;

ALTER TABLE agent
    ALTER COLUMN runtime_id SET NOT NULL;
