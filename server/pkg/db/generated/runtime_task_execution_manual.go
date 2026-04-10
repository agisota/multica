package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type UpdateManagedTaskDispatchRefParams struct {
	ID                  pgtype.UUID
	DispatchState       string
	ExternalExecutionID pgtype.Text
}

type CreateRuntimeTaskExecutionParams struct {
	TaskID              pgtype.UUID
	LeaseID             pgtype.UUID
	Backend             string
	ExternalExecutionID pgtype.Text
	Status              string
	Metadata            []byte
}

type UpdateRuntimeTaskExecutionParams struct {
	ID           pgtype.UUID
	Status       string
	Logs         pgtype.Text
	Result       []byte
	Error        pgtype.Text
	Metadata     []byte
	StartedAt    pgtype.Timestamptz
	CompletedAt  pgtype.Timestamptz
	LastPolledAt pgtype.Timestamptz
}

const updateManagedTaskDispatchRef = `
UPDATE agent_task_queue
SET
    dispatch_state = $2,
    external_execution_id = COALESCE($3, external_execution_id)
WHERE id = $1
`

func (q *Queries) UpdateManagedTaskDispatchRef(ctx context.Context, arg UpdateManagedTaskDispatchRefParams) error {
	_, err := q.db.Exec(ctx, updateManagedTaskDispatchRef, arg.ID, arg.DispatchState, arg.ExternalExecutionID)
	return err
}

const createRuntimeTaskExecution = `
INSERT INTO runtime_task_execution (
    task_id,
    lease_id,
    backend,
    external_execution_id,
    status,
    metadata,
    submitted_at
)
VALUES ($1, $2, $3, $4, $5, COALESCE($6, '{}'::jsonb), now())
RETURNING id, task_id, lease_id, backend, external_execution_id, status, logs, result, error, metadata, submitted_at, started_at, completed_at, last_polled_at, created_at, updated_at
`

func (q *Queries) CreateRuntimeTaskExecution(ctx context.Context, arg CreateRuntimeTaskExecutionParams) (RuntimeTaskExecution, error) {
	row := q.db.QueryRow(ctx, createRuntimeTaskExecution, arg.TaskID, arg.LeaseID, arg.Backend, arg.ExternalExecutionID, arg.Status, arg.Metadata)
	return scanRuntimeTaskExecution(row)
}

const updateRuntimeTaskExecution = `
UPDATE runtime_task_execution
SET
    status = $2,
    logs = COALESCE($3, logs),
    result = COALESCE($4, result),
    error = COALESCE($5, error),
    metadata = COALESCE($6, metadata),
    started_at = COALESCE($7, started_at),
    completed_at = COALESCE($8, completed_at),
    last_polled_at = COALESCE($9, last_polled_at),
    updated_at = now()
WHERE id = $1
RETURNING id, task_id, lease_id, backend, external_execution_id, status, logs, result, error, metadata, submitted_at, started_at, completed_at, last_polled_at, created_at, updated_at
`

func (q *Queries) UpdateRuntimeTaskExecution(ctx context.Context, arg UpdateRuntimeTaskExecutionParams) (RuntimeTaskExecution, error) {
	row := q.db.QueryRow(ctx, updateRuntimeTaskExecution, arg.ID, arg.Status, arg.Logs, arg.Result, arg.Error, arg.Metadata, arg.StartedAt, arg.CompletedAt, arg.LastPolledAt)
	return scanRuntimeTaskExecution(row)
}

const getLatestRuntimeTaskExecutionByTaskID = `
SELECT id, task_id, lease_id, backend, external_execution_id, status, logs, result, error, metadata, submitted_at, started_at, completed_at, last_polled_at, created_at, updated_at
FROM runtime_task_execution
WHERE task_id = $1
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLatestRuntimeTaskExecutionByTaskID(ctx context.Context, taskID pgtype.UUID) (RuntimeTaskExecution, error) {
	row := q.db.QueryRow(ctx, getLatestRuntimeTaskExecutionByTaskID, taskID)
	return scanRuntimeTaskExecution(row)
}

func scanRuntimeTaskExecution(row pgx.Row) (RuntimeTaskExecution, error) {
	var item RuntimeTaskExecution
	err := row.Scan(
		&item.ID,
		&item.TaskID,
		&item.LeaseID,
		&item.Backend,
		&item.ExternalExecutionID,
		&item.Status,
		&item.Logs,
		&item.Result,
		&item.Error,
		&item.Metadata,
		&item.SubmittedAt,
		&item.StartedAt,
		&item.CompletedAt,
		&item.LastPolledAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}
