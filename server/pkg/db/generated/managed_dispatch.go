package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const claimManagedTask = `
UPDATE agent_task_queue
SET
    status = 'dispatched',
    dispatched_at = now(),
    dispatch_state = 'dispatched',
    external_execution_id = COALESCE(agent_task_queue.external_execution_id, agent_task_queue.id::text)
WHERE id = (
    SELECT atq.id
    FROM agent_task_queue atq
    JOIN agent a ON a.id = atq.agent_id
    WHERE atq.execution_backend = $1
      AND atq.status = 'queued'
      AND atq.dispatch_state = 'queued'
      AND (
          SELECT count(*)
          FROM agent_task_queue active_agent
          WHERE active_agent.agent_id = atq.agent_id
            AND active_agent.status IN ('dispatched', 'running')
      ) < a.max_concurrent_tasks
      AND NOT EXISTS (
          SELECT 1
          FROM agent_task_queue active
          WHERE active.agent_id = atq.agent_id
            AND active.status IN ('dispatched', 'running')
            AND (
                (atq.issue_id IS NOT NULL AND active.issue_id = atq.issue_id)
                OR (atq.chat_session_id IS NOT NULL AND active.chat_session_id = atq.chat_session_id)
            )
      )
    ORDER BY atq.priority DESC, atq.created_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
RETURNING id, agent_id, issue_id, status, priority, dispatched_at, started_at, completed_at, result, error, created_at, context, runtime_id, session_id, work_dir, trigger_comment_id, chat_session_id, lease_id, execution_backend, dispatch_state, external_execution_id
`

func (q *Queries) ClaimManagedTask(ctx context.Context, backend string) (AgentTaskQueue, error) {
	row := q.db.QueryRow(ctx, claimManagedTask, backend)
	var i AgentTaskQueue
	err := row.Scan(
		&i.ID,
		&i.AgentID,
		&i.IssueID,
		&i.Status,
		&i.Priority,
		&i.DispatchedAt,
		&i.StartedAt,
		&i.CompletedAt,
		&i.Result,
		&i.Error,
		&i.CreatedAt,
		&i.Context,
		&i.RuntimeID,
		&i.SessionID,
		&i.WorkDir,
		&i.TriggerCommentID,
		&i.ChatSessionID,
		&i.LeaseID,
		&i.ExecutionBackend,
		&i.DispatchState,
		&i.ExternalExecutionID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AgentTaskQueue{}, err
		}
		return AgentTaskQueue{}, err
	}
	return i, nil
}

const listInflightManagedTasks = `
SELECT id, agent_id, issue_id, status, priority, dispatched_at, started_at, completed_at, result, error, created_at, context, runtime_id, session_id, work_dir, trigger_comment_id, chat_session_id, lease_id, execution_backend, dispatch_state, external_execution_id
FROM agent_task_queue
WHERE execution_backend = $1
  AND status IN ('dispatched', 'running')
ORDER BY created_at ASC
`

func (q *Queries) ListInflightManagedTasks(ctx context.Context, backend string) ([]AgentTaskQueue, error) {
	rows, err := q.db.Query(ctx, listInflightManagedTasks, backend)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AgentTaskQueue
	for rows.Next() {
		var i AgentTaskQueue
		if err := rows.Scan(
			&i.ID,
			&i.AgentID,
			&i.IssueID,
			&i.Status,
			&i.Priority,
			&i.DispatchedAt,
			&i.StartedAt,
			&i.CompletedAt,
			&i.Result,
			&i.Error,
			&i.CreatedAt,
			&i.Context,
			&i.RuntimeID,
			&i.SessionID,
			&i.WorkDir,
			&i.TriggerCommentID,
			&i.ChatSessionID,
			&i.LeaseID,
			&i.ExecutionBackend,
			&i.DispatchState,
			&i.ExternalExecutionID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
