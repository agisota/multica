package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/daemon"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/orchestration"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

const managedDispatchInterval = 2 * time.Second
const managedRecoveryMissingExternalID = "managed recovery: missing external execution id"

type managedDispatcher struct {
	queries       *db.Queries
	taskService   *service.TaskService
	executor      orchestration.Executor
	backend       string
	serverBaseURL string
	pollInterval  time.Duration
	sem           chan struct{}
}

func runManagedDispatcher(ctx context.Context, queries *db.Queries, hub *realtime.Hub, bus *events.Bus, serverBaseURL string) {
	cfg, err := daemon.LoadConfig(daemon.Overrides{ServerURL: serverBaseURL})
	if err != nil {
		slog.Warn("managed dispatcher disabled", "error", err)
		return
	}

	taskService := service.NewTaskService(queries, hub, bus)
	executor := orchestration.NewModalExecutorWithParams(orchestration.ModalExecutorParams{
		WorkspacesRoot: cfg.WorkspacesRoot,
		ServerBaseURL:  serverBaseURL,
		Logger:         slog.Default(),
		Timeout:        cfg.AgentTimeout,
	})

	maxConcurrent := cfg.MaxConcurrentTasks
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	if maxConcurrent > 4 {
		maxConcurrent = 4
	}

	dispatcher := &managedDispatcher{
		queries:       queries,
		taskService:   taskService,
		executor:      executor,
		backend:       string(orchestration.RuntimeBackendModal),
		serverBaseURL: serverBaseURL,
		pollInterval:  managedDispatchInterval,
		sem:           make(chan struct{}, maxConcurrent),
	}

	dispatcher.recoverManagedTasks(ctx)

	ticker := time.NewTicker(managedDispatchInterval)
	defer ticker.Stop()

	slog.Info("managed dispatcher started", "backend", dispatcher.backend, "concurrency", maxConcurrent)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		dispatched := false
		for len(dispatcher.sem) < cap(dispatcher.sem) {
			task, err := taskService.ClaimManagedTask(ctx, dispatcher.backend)
			if err != nil {
				slog.Warn("managed dispatcher: claim failed", "backend", dispatcher.backend, "error", err)
				break
			}
			if task == nil {
				break
			}

			dispatched = true
			dispatcher.launchDispatch(ctx, *task)
		}

		if dispatched {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *managedDispatcher) recoverManagedTasks(ctx context.Context) {
	tasks, err := d.queries.ListInflightManagedTasks(ctx, d.backend)
	if err != nil {
		slog.Warn("managed dispatcher: failed to list inflight tasks for recovery", "backend", d.backend, "error", err)
		return
	}
	if len(tasks) == 0 {
		return
	}

	recovered := 0
	for _, task := range tasks {
		executionID := pgtype.UUID{}
		externalID := stringsFromText(task.ExternalExecutionID)
		execRow, execErr := d.queries.GetLatestRuntimeTaskExecutionByTaskID(ctx, task.ID)
		if execErr == nil {
			executionID = execRow.ID
			if externalID == "" {
				externalID = stringsFromText(execRow.ExternalExecutionID)
			}
		}
		if externalID == "" {
			if _, failErr := d.taskService.FailTask(context.Background(), task.ID, managedRecoveryMissingExternalID); failErr != nil {
				slog.Warn("managed dispatcher: failed to mark unrecoverable task", "task_id", util.UUIDToString(task.ID), "error", failErr)
			}
			continue
		}

		if task.Status == "dispatched" {
			if _, err := d.taskService.StartTask(context.Background(), task.ID); err != nil {
				slog.Warn("managed dispatcher: failed to move recovered task to running", "task_id", util.UUIDToString(task.ID), "error", err)
				continue
			}
			task.Status = "running"
		}

		d.launchWatch(ctx, task, externalID, executionID)
		recovered++
	}

	slog.Info("managed dispatcher reattached inflight tasks", "backend", d.backend, "count", recovered)
}

func (d *managedDispatcher) launchDispatch(ctx context.Context, task db.AgentTaskQueue) {
	d.sem <- struct{}{}
	go func() {
		defer func() { <-d.sem }()
		d.dispatchTask(ctx, task)
	}()
}

func (d *managedDispatcher) launchWatch(ctx context.Context, task db.AgentTaskQueue, externalID string, executionID pgtype.UUID) {
	d.sem <- struct{}{}
	go func() {
		defer func() { <-d.sem }()
		d.watchManagedTask(ctx, task, externalID, executionID)
	}()
}

func (d *managedDispatcher) dispatchTask(ctx context.Context, task db.AgentTaskQueue) {
	payload, err := d.taskService.BuildManagedExecutionTask(ctx, task)
	if err != nil {
		_, _ = d.taskService.FailTask(context.Background(), task.ID, fmt.Sprintf("prepare managed task: %s", err.Error()))
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		_, _ = d.taskService.FailTask(context.Background(), task.ID, fmt.Sprintf("marshal managed task payload: %s", err.Error()))
		return
	}

	handle, err := d.executor.Submit(ctx, orchestration.ManagedTaskRequest{
		TaskID:      util.UUIDToString(task.ID),
		WorkspaceID: payload.Task.WorkspaceID,
		LeaseID:     util.UUIDToString(task.LeaseID),
		AgentID:     util.UUIDToString(task.AgentID),
		Provider:    payload.Provider,
		Payload:     payloadJSON,
		Metadata: map[string]any{
			"server_base_url": d.serverBaseURL,
		},
	})
	if err != nil {
		_, _ = d.taskService.FailTask(context.Background(), task.ID, fmt.Sprintf("submit managed task: %s", err.Error()))
		return
	}

	externalID := handle.ExternalID
	if externalID == "" {
		_, _ = d.taskService.FailTask(context.Background(), task.ID, "submit managed task: missing external execution id")
		return
	}

	if err := d.queries.UpdateManagedTaskDispatchRef(ctx, db.UpdateManagedTaskDispatchRefParams{
		ID:                  task.ID,
		DispatchState:       "dispatched",
		ExternalExecutionID: util.StrToText(externalID),
	}); err != nil {
		_, _ = d.taskService.FailTask(context.Background(), task.ID, fmt.Sprintf("persist external execution id: %s", err.Error()))
		return
	}

	metadataJSON, _ := json.Marshal(handle.Metadata)
	execution, err := d.queries.CreateRuntimeTaskExecution(ctx, db.CreateRuntimeTaskExecutionParams{
		TaskID:              task.ID,
		LeaseID:             task.LeaseID,
		Backend:             d.backend,
		ExternalExecutionID: util.StrToText(externalID),
		Status:              string(handle.Status),
		Metadata:            metadataJSON,
	})
	if err != nil {
		_, _ = d.taskService.FailTask(context.Background(), task.ID, fmt.Sprintf("create runtime task execution: %s", err.Error()))
		return
	}

	if _, err := d.taskService.StartTask(ctx, task.ID); err != nil {
		_, _ = d.taskService.FailTask(context.Background(), task.ID, fmt.Sprintf("start managed task: %s", err.Error()))
		return
	}

	startedAt := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	if _, err := d.queries.UpdateRuntimeTaskExecution(ctx, db.UpdateRuntimeTaskExecutionParams{
		ID:           execution.ID,
		Status:       string(orchestration.ManagedTaskStatusRunning),
		StartedAt:    startedAt,
		LastPolledAt: startedAt,
	}); err != nil {
		slog.Warn("managed dispatcher: failed to mark runtime execution running", "task_id", util.UUIDToString(task.ID), "execution_id", util.UUIDToString(execution.ID), "error", err)
	}

	task.Status = "running"
	d.watchManagedTask(ctx, task, externalID, execution.ID)
}

func (d *managedDispatcher) watchManagedTask(ctx context.Context, task db.AgentTaskQueue, externalID string, executionID pgtype.UUID) {
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		result, err := d.executor.Poll(ctx, externalID)
		if err != nil {
			slog.Warn("managed dispatcher: poll failed", "task_id", util.UUIDToString(task.ID), "external_id", externalID, "error", err)
			if executionID.Valid {
				now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
				if _, updateErr := d.queries.UpdateRuntimeTaskExecution(context.Background(), db.UpdateRuntimeTaskExecutionParams{
					ID:           executionID,
					Status:       string(orchestration.ManagedTaskStatusRunning),
					Error:        util.StrToText(err.Error()),
					LastPolledAt: now,
				}); updateErr != nil {
					slog.Warn("managed dispatcher: failed to persist poll error", "task_id", util.UUIDToString(task.ID), "error", updateErr)
				}
			}
		} else {
			if persistErr := d.persistExecutionPoll(executionID, result); persistErr != nil {
				slog.Warn("managed dispatcher: failed to persist poll state", "task_id", util.UUIDToString(task.ID), "error", persistErr)
			}
			if d.handleTerminalResult(task, result) {
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *managedDispatcher) persistExecutionPoll(executionID pgtype.UUID, result *orchestration.ManagedTaskResult) error {
	if !executionID.Valid || result == nil {
		return nil
	}
	now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	params := db.UpdateRuntimeTaskExecutionParams{
		ID:           executionID,
		Status:       string(result.Status),
		Logs:         util.StrToText(strings.TrimSpace(result.Logs)),
		Result:       result.Result,
		Error:        util.StrToText(strings.TrimSpace(result.Error)),
		LastPolledAt: now,
	}
	if result.Metadata != nil {
		params.Metadata, _ = json.Marshal(result.Metadata)
	}
	switch result.Status {
	case orchestration.ManagedTaskStatusRunning:
		params.StartedAt = now
	case orchestration.ManagedTaskStatusCompleted, orchestration.ManagedTaskStatusFailed, orchestration.ManagedTaskStatusCancelled, orchestration.ManagedTaskStatusTimedOut:
		params.CompletedAt = now
	}
	_, err := d.queries.UpdateRuntimeTaskExecution(context.Background(), params)
	return err
}

func (d *managedDispatcher) handleTerminalResult(task db.AgentTaskQueue, result *orchestration.ManagedTaskResult) bool {
	if result == nil {
		return false
	}

	switch result.Status {
	case orchestration.ManagedTaskStatusQueued, orchestration.ManagedTaskStatusRunning:
		return false
	case orchestration.ManagedTaskStatusCompleted:
		sessionID, workDir := metadataStrings(result.Metadata)
		if _, err := d.taskService.CompleteTask(context.Background(), task.ID, normalizeTaskResult(task.ID, result.Result, result.Metadata), sessionID, workDir); err != nil {
			slog.Warn("managed dispatcher: complete task failed", "task_id", util.UUIDToString(task.ID), "error", err)
		}
		return true
	case orchestration.ManagedTaskStatusCancelled:
		if _, err := d.taskService.FailTask(context.Background(), task.ID, "managed execution cancelled"); err != nil {
			slog.Warn("managed dispatcher: cancel result failed", "task_id", util.UUIDToString(task.ID), "error", err)
		}
		return true
	case orchestration.ManagedTaskStatusTimedOut:
		if _, err := d.taskService.FailTask(context.Background(), task.ID, "managed execution timed out"); err != nil {
			slog.Warn("managed dispatcher: timeout result failed", "task_id", util.UUIDToString(task.ID), "error", err)
		}
		return true
	default:
		errMsg := strings.TrimSpace(result.Error)
		if errMsg == "" {
			errMsg = "managed execution failed"
		}
		if _, err := d.taskService.FailTask(context.Background(), task.ID, errMsg); err != nil {
			slog.Warn("managed dispatcher: fail task failed", "task_id", util.UUIDToString(task.ID), "error", err)
		}
		return true
	}
}

func normalizeTaskResult(taskID pgtype.UUID, raw json.RawMessage, metadata map[string]any) []byte {
	if len(raw) == 0 {
		payload, _ := json.Marshal(protocol.TaskCompletedPayload{
			TaskID: util.UUIDToString(taskID),
		})
		return payload
	}

	var payload protocol.TaskCompletedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw
	}
	if payload.TaskID == "" {
		payload.TaskID = util.UUIDToString(taskID)
	}
	if payload.Output == "" && metadata != nil {
		if output, ok := metadata["output"].(string); ok {
			payload.Output = output
		}
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return normalized
}

func metadataStrings(metadata map[string]any) (sessionID string, workDir string) {
	if metadata == nil {
		return "", ""
	}
	if raw, ok := metadata["session_id"].(string); ok {
		sessionID = strings.TrimSpace(raw)
	}
	if raw, ok := metadata["work_dir"].(string); ok {
		workDir = strings.TrimSpace(raw)
	}
	return sessionID, workDir
}

func stringsFromText(text pgtype.Text) string {
	if !text.Valid {
		return ""
	}
	return strings.TrimSpace(text.String)
}
