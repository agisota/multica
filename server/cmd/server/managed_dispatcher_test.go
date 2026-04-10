package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/orchestration"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/service"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

type fakeManagedExecutor struct {
	result *orchestration.ManagedTaskResult
}

func (f *fakeManagedExecutor) Name() orchestration.RuntimeBackend {
	return orchestration.RuntimeBackendModal
}

func (f *fakeManagedExecutor) Mode() orchestration.RuntimeExecutionMode {
	return orchestration.RuntimeExecutionModeManaged
}

func (f *fakeManagedExecutor) Submit(context.Context, orchestration.ManagedTaskRequest) (*orchestration.ManagedTaskHandle, error) {
	return nil, nil
}

func (f *fakeManagedExecutor) Poll(context.Context, string) (*orchestration.ManagedTaskResult, error) {
	return f.result, nil
}

func (f *fakeManagedExecutor) Cancel(context.Context, string) error {
	return nil
}

func TestRecoverManagedTasksReattachesInflightTasks(t *testing.T) {
	if testPool == nil {
		t.Skip("no database connection")
	}

	ctx := context.Background()
	issueID, agentID := setupManagedRecoveryFixture(t)
	t.Cleanup(func() { cleanupSweeperFixture(t, issueID, agentID) })

	var taskID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (
			agent_id,
			issue_id,
			status,
			priority,
			dispatched_at,
			started_at,
			execution_backend,
			dispatch_state,
			external_execution_id
		)
		VALUES ($1, $2, 'running', 0, now() - interval '10 minutes', now() - interval '10 minutes', 'modal', 'running', 'sandbox-recover-1')
		RETURNING id
	`, agentID, issueID).Scan(&taskID)
	if err != nil {
		t.Fatalf("failed to create managed task: %v", err)
	}

	var executionID string
	err = testPool.QueryRow(ctx, `
		INSERT INTO runtime_task_execution (
			task_id,
			backend,
			external_execution_id,
			status,
			metadata,
			submitted_at,
			started_at
		)
		VALUES ($1, 'modal', 'sandbox-recover-1', 'running', '{}'::jsonb, now() - interval '10 minutes', now() - interval '10 minutes')
		RETURNING id
	`, taskID).Scan(&executionID)
	if err != nil {
		t.Fatalf("failed to create runtime task execution: %v", err)
	}

	_, err = testPool.Exec(ctx, `UPDATE agent SET status = 'working' WHERE id = $1`, agentID)
	if err != nil {
		t.Fatalf("failed to set agent status: %v", err)
	}

	result, _ := json.Marshal(protocol.TaskCompletedPayload{
		TaskID: taskID,
		Output: "Recovered remote task completed.",
	})

	dispatcher := &managedDispatcher{
		queries:     db.New(testPool),
		taskService: service.NewTaskService(db.New(testPool), realtime.NewHub(), events.New()),
		executor: &fakeManagedExecutor{result: &orchestration.ManagedTaskResult{
			ExternalID: "sandbox-recover-1",
			Status:     orchestration.ManagedTaskStatusCompleted,
			Result:     result,
			Metadata: map[string]any{
				"work_dir": "/workspace/recovered-repo",
			},
		}},
		backend:      "modal",
		pollInterval: 10 * time.Millisecond,
		sem:          make(chan struct{}, 4),
	}

	dispatcher.recoverManagedTasks(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for {
		var status, dispatchState string
		err = testPool.QueryRow(ctx, `
			SELECT status, dispatch_state
			FROM agent_task_queue
			WHERE id = $1
		`, taskID).Scan(&status, &dispatchState)
		if err != nil {
			t.Fatalf("failed to query recovered task: %v", err)
		}
		if status == "completed" && dispatchState == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for recovered task completion; last status=%q dispatch_state=%q", status, dispatchState)
		}
		time.Sleep(20 * time.Millisecond)
	}

	var storedWorkDir string
	err = testPool.QueryRow(ctx, `
		SELECT COALESCE(work_dir, '')
		FROM agent_task_queue
		WHERE id = $1
	`, taskID).Scan(&storedWorkDir)
	if err != nil {
		t.Fatalf("failed to query recovered task workdir: %v", err)
	}
	if storedWorkDir != "/workspace/recovered-repo" {
		t.Fatalf("expected recovered task workdir to be persisted, got %q", storedWorkDir)
	}

	var executionStatus string
	err = testPool.QueryRow(ctx, `
		SELECT status
		FROM runtime_task_execution
		WHERE id = $1
	`, executionID).Scan(&executionStatus)
	if err != nil {
		t.Fatalf("failed to query runtime task execution: %v", err)
	}
	if executionStatus != "completed" {
		t.Fatalf("expected runtime task execution status 'completed', got %q", executionStatus)
	}

	var agentStatus string
	err = testPool.QueryRow(ctx, `SELECT status FROM agent WHERE id = $1`, agentID).Scan(&agentStatus)
	if err != nil {
		t.Fatalf("failed to query agent status: %v", err)
	}
	if agentStatus != "idle" {
		t.Fatalf("expected agent status 'idle', got %q", agentStatus)
	}
}

func setupManagedRecoveryFixture(t *testing.T) (string, string) {
	t.Helper()
	ctx := context.Background()

	var agentID string
	err := testPool.QueryRow(ctx, `
		SELECT a.id
		FROM agent a
		JOIN member m ON m.workspace_id = a.workspace_id
		JOIN "user" u ON u.id = m.user_id
		WHERE u.email = $1
		LIMIT 1
	`, integrationTestEmail).Scan(&agentID)
	if err != nil {
		t.Fatalf("failed to find test agent: %v", err)
	}

	var issueID string
	err = testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, status, priority, creator_type, creator_id, assignee_type, assignee_id)
		SELECT $1, 'Managed recovery issue', 'todo', 'none', 'member', m.user_id, 'agent', $2
		FROM member m WHERE m.workspace_id = $1 LIMIT 1
		RETURNING id
	`, testWorkspaceID, agentID).Scan(&issueID)
	if err != nil {
		t.Fatalf("failed to create managed recovery issue: %v", err)
	}

	return issueID, agentID
}
