package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalRuntimeLeaseStartRegistersRuntime(t *testing.T) {
	createReq := newRequest("POST", "/api/runtimes/leases?workspace_id="+testWorkspaceID, map[string]any{
		"name":      "Handler Test Shared Lease",
		"backend":   "local",
		"placement": "shared",
		"scope":     "workspace",
	})

	createResp := httptest.NewRecorder()
	testHandler.CreateRuntimeLease(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("CreateRuntimeLease: expected 201, got %d: %s", createResp.Code, createResp.Body.String())
	}

	var lease RuntimeLeaseResponse
	if err := json.NewDecoder(createResp.Body).Decode(&lease); err != nil {
		t.Fatalf("CreateRuntimeLease: decode response: %v", err)
	}

	t.Cleanup(func() {
		if lease.ID != "" {
			_, _ = testPool.Exec(context.Background(), `DELETE FROM runtime_lease WHERE id = $1`, lease.ID)
		}
		if lease.RuntimeID != "" {
			_, _ = testPool.Exec(context.Background(), `DELETE FROM agent_runtime WHERE id = $1`, lease.RuntimeID)
		}
	})

	startReq := newRequest("POST", "/api/runtimes/leases/"+lease.ID+"/start?workspace_id="+testWorkspaceID, nil)
	startReq = withURLParam(startReq, "leaseId", lease.ID)

	startResp := httptest.NewRecorder()
	testHandler.StartRuntimeLease(startResp, startReq)
	if startResp.Code != http.StatusOK {
		t.Fatalf("StartRuntimeLease: expected 200, got %d: %s", startResp.Code, startResp.Body.String())
	}

	var started RuntimeLeaseResponse
	if err := json.NewDecoder(startResp.Body).Decode(&started); err != nil {
		t.Fatalf("StartRuntimeLease: decode response: %v", err)
	}

	if started.State != "active" {
		t.Fatalf("StartRuntimeLease: expected active state, got %q", started.State)
	}
	if started.RuntimeID == "" {
		t.Fatal("StartRuntimeLease: expected runtime_id to be assigned")
	}

	lease.RuntimeID = started.RuntimeID

	runtime, err := testHandler.Queries.GetAgentRuntime(context.Background(), parseUUID(started.RuntimeID))
	if err != nil {
		t.Fatalf("GetAgentRuntime: %v", err)
	}
	if runtime.Status != "online" {
		t.Fatalf("expected runtime status online, got %q", runtime.Status)
	}
	if runtime.RuntimeMode != "local" {
		t.Fatalf("expected runtime mode local, got %q", runtime.RuntimeMode)
	}
	if got := textToPtr(runtime.DaemonID); got == nil || *got != "lease:"+lease.ID {
		t.Fatalf("expected daemon_id %q, got %v", "lease:"+lease.ID, got)
	}

	listReq := newRequest("GET", "/api/runtimes?workspace_id="+testWorkspaceID, nil)
	listResp := httptest.NewRecorder()
	testHandler.ListAgentRuntimes(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("ListAgentRuntimes: expected 200, got %d: %s", listResp.Code, listResp.Body.String())
	}

	var runtimes []AgentRuntimeResponse
	if err := json.NewDecoder(listResp.Body).Decode(&runtimes); err != nil {
		t.Fatalf("ListAgentRuntimes: decode response: %v", err)
	}

	found := false
	for _, runtime := range runtimes {
		if runtime.ID == started.RuntimeID && runtime.Status == "online" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected started runtime %q to be listed as online", started.RuntimeID)
	}
}

func TestStopLocalRuntimeLeaseMarksRuntimeOffline(t *testing.T) {
	createReq := newRequest("POST", "/api/runtimes/leases?workspace_id="+testWorkspaceID, map[string]any{
		"name":      "Handler Test Private Lease",
		"backend":   "local",
		"placement": "private",
		"scope":     "user",
	})

	createResp := httptest.NewRecorder()
	testHandler.CreateRuntimeLease(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("CreateRuntimeLease: expected 201, got %d: %s", createResp.Code, createResp.Body.String())
	}

	var lease RuntimeLeaseResponse
	if err := json.NewDecoder(createResp.Body).Decode(&lease); err != nil {
		t.Fatalf("CreateRuntimeLease: decode response: %v", err)
	}

	t.Cleanup(func() {
		if lease.ID != "" {
			_, _ = testPool.Exec(context.Background(), `DELETE FROM runtime_lease WHERE id = $1`, lease.ID)
		}
		if lease.RuntimeID != "" {
			_, _ = testPool.Exec(context.Background(), `DELETE FROM agent_runtime WHERE id = $1`, lease.RuntimeID)
		}
	})

	startReq := newRequest("POST", "/api/runtimes/leases/"+lease.ID+"/start?workspace_id="+testWorkspaceID, nil)
	startReq = withURLParam(startReq, "leaseId", lease.ID)
	startResp := httptest.NewRecorder()
	testHandler.StartRuntimeLease(startResp, startReq)
	if startResp.Code != http.StatusOK {
		t.Fatalf("StartRuntimeLease: expected 200, got %d: %s", startResp.Code, startResp.Body.String())
	}

	var started RuntimeLeaseResponse
	if err := json.NewDecoder(startResp.Body).Decode(&started); err != nil {
		t.Fatalf("StartRuntimeLease: decode response: %v", err)
	}
	lease.RuntimeID = started.RuntimeID

	stopReq := newRequest("POST", "/api/runtimes/leases/"+lease.ID+"/stop?workspace_id="+testWorkspaceID, nil)
	stopReq = withURLParam(stopReq, "leaseId", lease.ID)
	stopResp := httptest.NewRecorder()
	testHandler.StopRuntimeLease(stopResp, stopReq)
	if stopResp.Code != http.StatusOK {
		t.Fatalf("StopRuntimeLease: expected 200, got %d: %s", stopResp.Code, stopResp.Body.String())
	}

	runtime, err := testHandler.Queries.GetAgentRuntime(context.Background(), parseUUID(started.RuntimeID))
	if err != nil {
		t.Fatalf("GetAgentRuntime: %v", err)
	}
	if runtime.Status != "offline" {
		t.Fatalf("expected runtime status offline, got %q", runtime.Status)
	}
}

func TestCreateRuntimeLeaseAllowsModalBackend(t *testing.T) {
	req := newRequest("POST", "/api/runtimes/leases?workspace_id="+testWorkspaceID, map[string]any{
		"name":      "Handler Test Modal Lease",
		"backend":   "modal",
		"placement": "private",
		"scope":     "user",
	})

	resp := httptest.NewRecorder()
	testHandler.CreateRuntimeLease(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("CreateRuntimeLease: expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var lease RuntimeLeaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&lease); err != nil {
		t.Fatalf("CreateRuntimeLease: decode response: %v", err)
	}

	t.Cleanup(func() {
		if lease.ID != "" {
			_, _ = testPool.Exec(context.Background(), `DELETE FROM runtime_lease WHERE id = $1`, lease.ID)
		}
	})
}
