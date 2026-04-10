package orchestration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	modal "github.com/modal-labs/modal-client/go"
	"github.com/multica-ai/multica/server/internal/daemon"
	"github.com/multica-ai/multica/server/internal/daemon/execenv"
	"github.com/multica-ai/multica/server/internal/daemon/repocache"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

const (
	defaultModalExecutorAppName = "multica-managed-executor"
	defaultModalSandboxTimeout  = 2 * time.Hour
	modalResultMarker           = "__MULTICA_RESULT__"
)

type ModalExecutorParams struct {
	WorkspacesRoot string
	ServerBaseURL  string
	Logger         *slog.Logger
	Timeout        time.Duration
}

type modalExecutor struct {
	workspacesRoot string
	serverBaseURL  string
	timeout        time.Duration
	logger         *slog.Logger
	tokenID        string
	tokenSecret    string
	environment    string
	appName        string
}

type modalManagedExecutionPayload struct {
	Task       daemon.Task
	Provider   string
	AuthUserID string
}

type modalResultEnvelope struct {
	Status     string `json:"status"`
	TaskID     string `json:"task_id"`
	Output     string `json:"output"`
	Error      string `json:"error"`
	BranchName string `json:"branch_name"`
	WorkDir    string `json:"work_dir"`
	ExitCode   int    `json:"exit_code"`
}

type modalWorkspaceLayout struct {
	Environment    *execenv.Environment
	RemoteRepoDirs []string
	DefaultRunDir  string
}

func NewModalExecutor() Executor {
	return NewModalExecutorWithParams(ModalExecutorParams{})
}

func NewModalExecutorWithParams(params ModalExecutorParams) Executor {
	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}
	timeout := params.Timeout
	if timeout <= 0 {
		timeout = defaultModalSandboxTimeout
	}
	return &modalExecutor{
		workspacesRoot: strings.TrimSpace(params.WorkspacesRoot),
		serverBaseURL:  strings.TrimSpace(params.ServerBaseURL),
		timeout:        timeout,
		logger:         logger,
		tokenID:        strings.TrimSpace(os.Getenv("MODAL_TOKEN_ID")),
		tokenSecret:    strings.TrimSpace(os.Getenv("MODAL_TOKEN_SECRET")),
		environment:    firstNonEmpty(os.Getenv("MODAL_ENVIRONMENT"), "main"),
		appName:        firstNonEmpty(os.Getenv("MULTICA_MODAL_APP_NAME"), defaultModalExecutorAppName),
	}
}

func (e *modalExecutor) Name() RuntimeBackend {
	return RuntimeBackendModal
}

func (e *modalExecutor) Mode() RuntimeExecutionMode {
	return RuntimeExecutionModeManaged
}

func (e *modalExecutor) Submit(ctx context.Context, req ManagedTaskRequest) (*ManagedTaskHandle, error) {
	payload, err := e.parsePayload(req.Payload)
	if err != nil {
		return nil, err
	}
	if payload.Provider != "codex" {
		return nil, fmt.Errorf("modal executor only supports codex managed tasks, got %q", payload.Provider)
	}

	layout, err := e.prepareWorkspace(payload)
	if err != nil {
		return nil, err
	}
	defer func() {
		if layout != nil && layout.Environment != nil {
			_ = layout.Environment.Cleanup(true)
		}
	}()

	client, err := e.newClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	app, err := client.Apps.FromName(ctx, e.appName, &modal.AppFromNameParams{
		Environment:     e.environment,
		CreateIfMissing: true,
	})
	if err != nil {
		return nil, fmt.Errorf("load modal app: %w", err)
	}

	image := client.Images.FromRegistry("node:22-bookworm", nil).DockerfileCommands([]string{
		"RUN apt-get update && apt-get install -y git ca-certificates openssh-client && rm -rf /var/lib/apt/lists/*",
		"RUN npm install -g @openai/codex",
	}, nil)

	sandbox, err := client.Sandboxes.Create(ctx, app, image, &modal.SandboxCreateParams{
		Timeout: e.timeout,
		Workdir: "/workspace",
		Command: []string{
			"/bin/bash",
			"-lc",
			"while [ ! -f /workspace/.bootstrap-ready ]; do sleep 0.2; done; exec /bin/bash /workspace/run-task.sh",
		},
		Env: map[string]string{
			"HOME":                "/workspace",
			"CODEX_HOME":          "/workspace/.codex",
			"MULTICA_TASK_ID":     payload.Task.ID,
			"MULTICA_CODEX_MODEL": strings.TrimSpace(os.Getenv("MULTICA_CODEX_MODEL")),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create modal sandbox: %w", err)
	}

	if err := e.bootstrapSandbox(ctx, sandbox, payload, layout); err != nil {
		_, _ = sandbox.Terminate(context.Background(), &modal.SandboxTerminateParams{Wait: false})
		return nil, err
	}

	metadata := map[string]any{
		"backend":          string(RuntimeBackendModal),
		"environment":      e.environment,
		"default_run_dir":  layout.DefaultRunDir,
		"remote_repo_dirs": layout.RemoteRepoDirs,
		"workspace_id":     payload.Task.WorkspaceID,
		"task_provider":    payload.Provider,
		"modal_app_name":   e.appName,
		"server_base_url":  e.serverBaseURL,
	}

	return &ManagedTaskHandle{
		ExternalID: sandbox.SandboxID,
		Status:     ManagedTaskStatusQueued,
		Metadata:   metadata,
	}, nil
}

func (e *modalExecutor) Poll(ctx context.Context, externalID string) (*ManagedTaskResult, error) {
	client, err := e.newClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	sandbox, err := client.Sandboxes.FromID(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("load modal sandbox: %w", err)
	}

	exitCode, err := sandbox.Poll(ctx)
	if err != nil {
		return nil, fmt.Errorf("poll modal sandbox: %w", err)
	}
	if exitCode == nil {
		return &ManagedTaskResult{
			ExternalID: externalID,
			Status:     ManagedTaskStatusRunning,
		}, nil
	}

	stdoutBytes, stdoutErr := io.ReadAll(sandbox.Stdout)
	stderrBytes, stderrErr := io.ReadAll(sandbox.Stderr)
	if stdoutErr != nil {
		return nil, fmt.Errorf("read sandbox stdout: %w", stdoutErr)
	}
	if stderrErr != nil {
		return nil, fmt.Errorf("read sandbox stderr: %w", stderrErr)
	}

	envelope, cleanedStdout, parseErr := parseModalResult(stdoutBytes)
	if parseErr != nil {
		return nil, fmt.Errorf("parse sandbox result: %w", parseErr)
	}

	logs := strings.TrimSpace(joinLogs(cleanedStdout, stderrBytes))
	if envelope == nil {
		status := ManagedTaskStatusCompleted
		errMsg := ""
		if *exitCode != 0 {
			status = ManagedTaskStatusFailed
			errMsg = strings.TrimSpace(string(stderrBytes))
			if errMsg == "" {
				errMsg = strings.TrimSpace(string(stdoutBytes))
			}
			if errMsg == "" {
				errMsg = fmt.Sprintf("modal sandbox exited with code %d", *exitCode)
			}
		}

		payload, _ := json.Marshal(protocol.TaskCompletedPayload{
			Output: strings.TrimSpace(string(cleanedStdout)),
		})
		return &ManagedTaskResult{
			ExternalID: externalID,
			Status:     status,
			Logs:       logs,
			Result:     payload,
			Error:      errMsg,
			Metadata: map[string]any{
				"exit_code": *exitCode,
			},
		}, nil
	}

	status := ManagedTaskStatus(envelope.Status)
	if status == "" {
		if envelope.ExitCode == 0 {
			status = ManagedTaskStatusCompleted
		} else {
			status = ManagedTaskStatusFailed
		}
	}

	var result json.RawMessage
	if envelope.TaskID != "" || envelope.Output != "" {
		completed, _ := json.Marshal(protocol.TaskCompletedPayload{
			TaskID: envelope.TaskID,
			Output: envelope.Output,
		})
		result = completed
	}

	metadata := map[string]any{
		"branch_name": envelope.BranchName,
		"work_dir":    envelope.WorkDir,
		"exit_code":   envelope.ExitCode,
	}

	return &ManagedTaskResult{
		ExternalID: externalID,
		Status:     status,
		Logs:       logs,
		Result:     result,
		Error:      strings.TrimSpace(envelope.Error),
		Metadata:   metadata,
	}, nil
}

func (e *modalExecutor) Cancel(ctx context.Context, externalID string) error {
	client, err := e.newClient()
	if err != nil {
		return err
	}
	defer client.Close()

	sandbox, err := client.Sandboxes.FromID(ctx, externalID)
	if err != nil {
		return fmt.Errorf("load modal sandbox: %w", err)
	}
	if _, err := sandbox.Terminate(ctx, &modal.SandboxTerminateParams{Wait: false}); err != nil {
		return fmt.Errorf("terminate modal sandbox: %w", err)
	}
	return nil
}

func (e *modalExecutor) parsePayload(raw json.RawMessage) (*modalManagedExecutionPayload, error) {
	var payload modalManagedExecutionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode modal task payload: %w", err)
	}
	if payload.Task.ID == "" {
		return nil, fmt.Errorf("decode modal task payload: missing task id")
	}
	if payload.Task.WorkspaceID == "" {
		return nil, fmt.Errorf("decode modal task payload: missing workspace id")
	}
	if payload.Task.Agent == nil {
		return nil, fmt.Errorf("decode modal task payload: missing agent data")
	}
	return &payload, nil
}

func (e *modalExecutor) prepareWorkspace(payload *modalManagedExecutionPayload) (*modalWorkspaceLayout, error) {
	if e.workspacesRoot == "" {
		return nil, fmt.Errorf("modal executor requires a workspaces root")
	}

	taskContext := execenv.TaskContextForEnv{
		IssueID:           payload.Task.IssueID,
		IssueTitle:        payload.Task.IssueTitle,
		IssueDescription:  payload.Task.IssueDescription,
		TriggerCommentID:  payload.Task.TriggerCommentID,
		ChatSessionID:     payload.Task.ChatSessionID,
		AgentName:         payload.Task.Agent.Name,
		AgentInstructions: payload.Task.Agent.Instructions,
		AgentSkills:       toExecEnvSkills(payload.Task.Agent.Skills),
		Repos:             toExecEnvRepos(payload.Task.Repos),
	}

	env, err := execenv.Prepare(execenv.PrepareParams{
		WorkspacesRoot: e.workspacesRoot,
		WorkspaceID:    payload.Task.WorkspaceID,
		TaskID:         payload.Task.ID,
		AgentName:      payload.Task.Agent.Name,
		Provider:       payload.Provider,
		Task:           taskContext,
	}, e.logger)
	if err != nil {
		return nil, fmt.Errorf("prepare task workspace: %w", err)
	}

	layout := &modalWorkspaceLayout{
		Environment:   env,
		DefaultRunDir: "/workspace",
	}

	if len(payload.Task.Repos) == 0 {
		return layout, nil
	}

	cache := repocache.New(filepath.Join(e.workspacesRoot, ".repos"), e.logger)
	repos := toRepoCacheRepos(payload.Task.Repos)
	if err := cache.Sync(payload.Task.WorkspaceID, repos); err != nil {
		return nil, fmt.Errorf("sync repo cache: %w", err)
	}

	for _, repo := range payload.Task.Repos {
		worktree, err := cache.CreateWorktree(repocache.WorktreeParams{
			WorkspaceID: payload.Task.WorkspaceID,
			RepoURL:     repo.URL,
			WorkDir:     env.WorkDir,
			AgentName:   payload.Task.Agent.Name,
			TaskID:      payload.Task.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("create worktree for %s: %w", repo.URL, err)
		}
		layout.RemoteRepoDirs = append(layout.RemoteRepoDirs, "/workspace/"+filepath.Base(worktree.Path))
	}

	if len(layout.RemoteRepoDirs) == 1 {
		layout.DefaultRunDir = layout.RemoteRepoDirs[0]
	}
	return layout, nil
}

func (e *modalExecutor) bootstrapSandbox(ctx context.Context, sandbox *modal.Sandbox, payload *modalManagedExecutionPayload, layout *modalWorkspaceLayout) error {
	if _, err := sandbox.Exec(ctx, []string{"/bin/bash", "-lc", "mkdir -p /workspace /workspace/.codex"}, nil); err != nil {
		return fmt.Errorf("prepare sandbox directories: %w", err)
	}

	if err := uploadTree(ctx, sandbox, layout.Environment.WorkDir, "/workspace"); err != nil {
		return fmt.Errorf("upload workdir: %w", err)
	}
	if layout.Environment.CodexHome != "" {
		if err := uploadTree(ctx, sandbox, layout.Environment.CodexHome, "/workspace/.codex"); err != nil {
			return fmt.Errorf("upload codex home: %w", err)
		}
	}

	files := map[string]string{
		"/workspace/AGENTS.md":   buildRemoteAgents(payload, layout),
		"/workspace/prompt.txt":  buildRemotePrompt(payload.Task, layout),
		"/workspace/run-task.sh": buildRemoteRunnerScript(),
	}
	if len(layout.RemoteRepoDirs) == 1 {
		files["/workspace/.single-repo-path"] = layout.RemoteRepoDirs[0] + "\n"
	}

	for remotePath, content := range files {
		if err := writeRemoteFile(ctx, sandbox, remotePath, []byte(content)); err != nil {
			return fmt.Errorf("upload %s: %w", remotePath, err)
		}
	}

	if _, err := sandbox.Exec(ctx, []string{"/bin/bash", "-lc", "touch /workspace/.bootstrap-ready"}, nil); err != nil {
		return fmt.Errorf("finalize sandbox bootstrap: %w", err)
	}
	return nil
}

func (e *modalExecutor) newClient() (*modal.Client, error) {
	if e.tokenID == "" || e.tokenSecret == "" {
		return nil, fmt.Errorf("modal executor requires MODAL_TOKEN_ID and MODAL_TOKEN_SECRET")
	}
	return modal.NewClientWithOptions(&modal.ClientParams{
		TokenID:     e.tokenID,
		TokenSecret: e.tokenSecret,
		Environment: e.environment,
		Logger:      e.logger,
	})
}

func toExecEnvRepos(repos []daemon.RepoData) []execenv.RepoContextForEnv {
	if len(repos) == 0 {
		return nil
	}
	result := make([]execenv.RepoContextForEnv, 0, len(repos))
	for _, repo := range repos {
		result = append(result, execenv.RepoContextForEnv{
			URL:         repo.URL,
			Description: repo.Description,
		})
	}
	return result
}

func toExecEnvSkills(skills []daemon.SkillData) []execenv.SkillContextForEnv {
	if len(skills) == 0 {
		return nil
	}
	result := make([]execenv.SkillContextForEnv, 0, len(skills))
	for _, skill := range skills {
		item := execenv.SkillContextForEnv{
			Name:    skill.Name,
			Content: skill.Content,
		}
		for _, file := range skill.Files {
			item.Files = append(item.Files, execenv.SkillFileContextForEnv{
				Path:    file.Path,
				Content: file.Content,
			})
		}
		result = append(result, item)
	}
	return result
}

func toRepoCacheRepos(repos []daemon.RepoData) []repocache.RepoInfo {
	if len(repos) == 0 {
		return nil
	}
	result := make([]repocache.RepoInfo, 0, len(repos))
	for _, repo := range repos {
		result = append(result, repocache.RepoInfo{
			URL:         repo.URL,
			Description: repo.Description,
		})
	}
	return result
}

func buildRemoteAgents(payload *modalManagedExecutionPayload, layout *modalWorkspaceLayout) string {
	var b strings.Builder
	b.WriteString("# Remote Modal Sandbox\n\n")
	b.WriteString("You are running inside a remote Modal sandbox for a managed Multica task.\n")
	b.WriteString("Repositories are already checked out locally. Do not run `multica repo checkout`.\n")
	b.WriteString("Use the checked-out repositories under `/workspace` as the source of truth.\n")
	b.WriteString("Read `/workspace/.agent_context/issue_context.md` first.\n")
	b.WriteString("Complete the work end-to-end and run targeted verification before finishing.\n")
	if payload.Task.Agent != nil && strings.TrimSpace(payload.Task.Agent.Instructions) != "" {
		b.WriteString("\n## Agent Instructions\n\n")
		b.WriteString(strings.TrimSpace(payload.Task.Agent.Instructions))
		b.WriteString("\n")
	}
	if len(layout.RemoteRepoDirs) > 0 {
		b.WriteString("\n## Checked Out Repositories\n\n")
		for _, repoDir := range layout.RemoteRepoDirs {
			b.WriteString("- " + repoDir + "\n")
		}
	}
	return b.String()
}

func buildRemotePrompt(task daemon.Task, layout *modalWorkspaceLayout) string {
	var b strings.Builder
	b.WriteString("You are running as a remote coding agent inside a Modal sandbox for a Multica workspace.\n\n")
	b.WriteString(daemon.BuildPrompt(task))
	b.WriteString("\n")
	b.WriteString("Primary issue context: `/workspace/.agent_context/issue_context.md`.\n")
	b.WriteString("Repositories are already checked out under `/workspace`.\n")
	b.WriteString("Do not use `multica repo checkout`.\n")
	if len(layout.RemoteRepoDirs) > 0 {
		b.WriteString("Available repositories:\n")
		for _, repoDir := range layout.RemoteRepoDirs {
			b.WriteString("- " + repoDir + "\n")
		}
	}
	b.WriteString("Run the smallest verification that proves the task is complete.\n")
	b.WriteString("Your final answer must be concise and concrete.\n")
	return b.String()
}

func buildRemoteRunnerScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

export HOME=/workspace
export CODEX_HOME=/workspace/.codex
export PATH="/usr/local/bin:/usr/bin:/bin:${PATH}"

RUN_DIR="/workspace"
if [[ -f /workspace/.single-repo-path ]]; then
  RUN_DIR="$(tr -d '\n' < /workspace/.single-repo-path)"
fi

OUTPUT_FILE="/workspace/last-message.txt"
STATUS="completed"
EXIT_CODE=0

MODEL_ARGS=()
if [[ -n "${MULTICA_CODEX_MODEL:-}" ]]; then
  MODEL_ARGS=(-m "${MULTICA_CODEX_MODEL}")
fi

codex exec --skip-git-repo-check -C "${RUN_DIR}" -s danger-full-access -o "${OUTPUT_FILE}" "${MODEL_ARGS[@]}" < /workspace/prompt.txt || EXIT_CODE=$?
if [[ "${EXIT_CODE}" -ne 0 ]]; then
  STATUS="failed"
fi

BRANCH_NAME=""
if git -C "${RUN_DIR}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  BRANCH_NAME="$(git -C "${RUN_DIR}" branch --show-current || true)"
fi

export MULTICA_STATUS="${STATUS}"
export MULTICA_RUN_DIR="${RUN_DIR}"
export MULTICA_BRANCH="${BRANCH_NAME}"
export MULTICA_EXIT_CODE="${EXIT_CODE}"

node <<'EOF'
const fs = require("fs");

const outputPath = "/workspace/last-message.txt";
let output = "";
try {
  output = fs.readFileSync(outputPath, "utf8").trim();
} catch {}

const status = process.env.MULTICA_STATUS || "failed";
const exitCode = Number(process.env.MULTICA_EXIT_CODE || "1");
const payload = {
  status,
  task_id: process.env.MULTICA_TASK_ID || "",
  output,
  branch_name: process.env.MULTICA_BRANCH || "",
  work_dir: process.env.MULTICA_RUN_DIR || "/workspace",
  exit_code: exitCode,
};

if (status !== "completed") {
  payload.error = output || ("codex exited with code " + exitCode);
}

console.log("__MULTICA_RESULT__" + JSON.stringify(payload));
EOF

exit "${EXIT_CODE}"
`
}

func uploadTree(ctx context.Context, sandbox *modal.Sandbox, localRoot, remoteRoot string) error {
	return filepath.WalkDir(localRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(localRoot, path)
		if err != nil {
			return err
		}
		remotePath := remoteRoot
		if rel != "." {
			remotePath = filepath.ToSlash(filepath.Join(remoteRoot, rel))
		}
		if d.IsDir() {
			if _, err := sandbox.Exec(ctx, []string{"/bin/bash", "-lc", "mkdir -p " + shellQuote(remotePath)}, nil); err != nil {
				return err
			}
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			targetBytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return writeRemoteFile(ctx, sandbox, remotePath, targetBytes)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeRemoteFile(ctx, sandbox, remotePath, content)
	})
}

func writeRemoteFile(ctx context.Context, sandbox *modal.Sandbox, remotePath string, content []byte) error {
	dir := filepath.ToSlash(filepath.Dir(remotePath))
	if _, err := sandbox.Exec(ctx, []string{"/bin/bash", "-lc", "mkdir -p " + shellQuote(dir)}, nil); err != nil {
		return err
	}
	file, err := sandbox.Open(ctx, remotePath, "w")
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, bytes.NewReader(content)); err != nil {
		return err
	}
	return nil
}

func parseModalResult(stdout []byte) (*modalResultEnvelope, []byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	var cleaned []string
	var envelope *modalResultEnvelope
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, modalResultMarker) {
			raw := strings.TrimPrefix(line, modalResultMarker)
			var parsed modalResultEnvelope
			if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
				return nil, nil, err
			}
			envelope = &parsed
			continue
		}
		cleaned = append(cleaned, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return envelope, []byte(strings.Join(cleaned, "\n")), nil
}

func joinLogs(parts ...[]byte) string {
	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		text := strings.TrimSpace(string(part))
		if text != "" {
			nonEmpty = append(nonEmpty, text)
		}
	}
	return strings.Join(nonEmpty, "\n\n")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
