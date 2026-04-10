package daemon

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/cli"
)

func TestNormalizeServerBaseURL(t *testing.T) {
	t.Parallel()

	got, err := NormalizeServerBaseURL("ws://localhost:8080/ws")
	if err != nil {
		t.Fatalf("NormalizeServerBaseURL returned error: %v", err)
	}
	if got != "http://localhost:8080" {
		t.Fatalf("expected http://localhost:8080, got %s", got)
	}
}

func TestBuildPromptContainsIssueID(t *testing.T) {
	t.Parallel()

	issueID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	prompt := BuildPrompt(Task{
		IssueID: issueID,
		Agent: &AgentData{
			Name: "Local Codex",
			Skills: []SkillData{
				{Name: "Concise", Content: "Be concise."},
			},
		},
	})

	// Prompt should contain the issue ID and CLI hint.
	for _, want := range []string{
		issueID,
		"multica issue get",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}

	// Skills should NOT be inlined in the prompt (they're in runtime config).
	for _, absent := range []string{"## Agent Skills", "Be concise."} {
		if strings.Contains(prompt, absent) {
			t.Fatalf("prompt should NOT contain %q (skills are in runtime config)", absent)
		}
	}
}

func TestBuildPromptNoIssueDetails(t *testing.T) {
	t.Parallel()

	prompt := BuildPrompt(Task{
		IssueID: "test-id",
		Agent:   &AgentData{Name: "Test"},
	})

	// Prompt should not contain issue title/description (agent fetches via CLI).
	for _, absent := range []string{"**Issue:**", "**Summary:**"} {
		if strings.Contains(prompt, absent) {
			t.Fatalf("prompt should NOT contain %q — agent fetches details via CLI", absent)
		}
	}
}

func TestIsWorkspaceNotFoundError(t *testing.T) {
	t.Parallel()

	err := &requestError{
		Method:     http.MethodPost,
		Path:       "/api/daemon/register",
		StatusCode: http.StatusNotFound,
		Body:       `{"error":"workspace not found"}`,
	}
	if !isWorkspaceNotFoundError(err) {
		t.Fatal("expected workspace not found error to be recognized")
	}

	if isWorkspaceNotFoundError(&requestError{StatusCode: http.StatusInternalServerError, Body: `{"error":"workspace not found"}`}) {
		t.Fatal("did not expect 500 to be treated as workspace not found")
	}
}

func TestLoadWatchedWorkspacesRecoversStaleConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	const (
		staleWorkspaceID = "stale-workspace"
		validWorkspaceID = "valid-workspace"
	)

	cfg := cli.CLIConfig{
		ServerURL:   "http://example.test",
		WorkspaceID: staleWorkspaceID,
		WatchedWorkspaces: []cli.WatchedWorkspace{
			{ID: staleWorkspaceID, Name: "Stale"},
		},
	}
	if err := cli.SaveCLIConfig(cfg); err != nil {
		t.Fatalf("SaveCLIConfig returned error: %v", err)
	}

	var registerAttempts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/daemon/register":
			var req struct {
				WorkspaceID string `json:"workspace_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode register request: %v", err)
			}
			registerAttempts = append(registerAttempts, req.WorkspaceID)
			if req.WorkspaceID == staleWorkspaceID {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"workspace not found"}`))
				return
			}
			if req.WorkspaceID != validWorkspaceID {
				t.Fatalf("unexpected workspace registration attempt: %s", req.WorkspaceID)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"runtimes":[{"id":"runtime-1","name":"Codex","provider":"codex","status":"online"}],"repos":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/workspaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"valid-workspace","name":"Valid"}]`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("LookPath(bash) returned error: %v", err)
	}

	d := New(Config{
		ServerBaseURL: server.URL,
		DaemonID:      "daemon-test",
		DeviceName:    "test-device",
		CLIVersion:    "1.0.0",
		Agents: map[string]AgentEntry{
			"codex": {Path: bashPath},
		},
		WorkspacesRoot: filepath.Join(home, "workspaces"),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	d.client.SetToken("token")
	d.authToken = "token"

	if err := d.loadWatchedWorkspaces(context.Background()); err != nil {
		t.Fatalf("loadWatchedWorkspaces returned error: %v", err)
	}

	if !slices.Equal(registerAttempts, []string{staleWorkspaceID, validWorkspaceID}) {
		t.Fatalf("unexpected register attempts: %v", registerAttempts)
	}

	if got := d.allRuntimeIDs(); !slices.Equal(got, []string{"runtime-1"}) {
		t.Fatalf("unexpected runtime IDs: %v", got)
	}

	saved, err := cli.LoadCLIConfig()
	if err != nil {
		t.Fatalf("LoadCLIConfig returned error: %v", err)
	}
	if saved.WorkspaceID != validWorkspaceID {
		t.Fatalf("expected workspace_id %q, got %q", validWorkspaceID, saved.WorkspaceID)
	}
	if !slices.Equal(saved.WatchedWorkspaces, []cli.WatchedWorkspace{{ID: validWorkspaceID, Name: "Valid"}}) {
		t.Fatalf("unexpected watched workspaces: %+v", saved.WatchedWorkspaces)
	}
}
