package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type httpProvisioner struct {
	backend    RuntimeBackend
	baseURL    string
	token      string
	orgID      string
	httpClient *http.Client
}

func newHTTPProvisioner(backend RuntimeBackend, baseURL, token, orgID string) *httpProvisioner {
	return &httpProvisioner{
		backend:    backend,
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:      strings.TrimSpace(token),
		orgID:      strings.TrimSpace(orgID),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *httpProvisioner) configured() bool {
	return p.baseURL != "" && p.token != ""
}

func (p *httpProvisioner) postJSON(ctx context.Context, path string, payload any) (map[string]any, error) {
	if !p.configured() {
		return nil, fmt.Errorf("%s is not configured", p.backend)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	if p.orgID != "" {
		req.Header.Set("X-Daytona-Organization-ID", p.orgID)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s request failed: %s", p.backend, strings.TrimSpace(string(respBody)))
	}

	if len(respBody) == 0 {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type daytonaProvisioner struct {
	*httpProvisioner
	target             string
	snapshot           string
	class              string
	autoStopInterval   int
	autoDeleteInterval int
}

func NewDaytonaProvisionerFromEnv() Provisioner {
	return &daytonaProvisioner{
		httpProvisioner: newHTTPProvisioner(
			RuntimeBackendDaytona,
			os.Getenv("DAYTONA_API_URL"),
			os.Getenv("DAYTONA_API_KEY"),
			os.Getenv("DAYTONA_ORGANIZATION_ID"),
		),
		target:             firstNonEmpty(os.Getenv("DAYTONA_TARGET"), "us"),
		snapshot:           firstNonEmpty(os.Getenv("DAYTONA_SNAPSHOT"), "daytona-small"),
		class:              firstNonEmpty(os.Getenv("DAYTONA_CLASS"), "small"),
		autoStopInterval:   firstNonZeroInt(os.Getenv("DAYTONA_AUTO_STOP_INTERVAL"), 30),
		autoDeleteInterval: firstNonZeroInt(os.Getenv("DAYTONA_AUTO_DELETE_INTERVAL"), 120),
	}
}

func (p *daytonaProvisioner) Name() RuntimeBackend { return RuntimeBackendDaytona }

func (p *daytonaProvisioner) Provision(ctx context.Context, spec RuntimeSpec) (*ProvisionedRuntime, error) {
	payload := map[string]any{
		"name":     spec.Name,
		"snapshot": p.snapshot,
		"class":    p.class,
		"labels": map[string]any{
			"multica_lease_id": spec.LeaseID,
			"workspace_id":     spec.WorkspaceID,
			"provider":         spec.Provider,
			"placement":        spec.Placement,
		},
		"env": map[string]any{
			"MULTICA_WORKSPACE_ID": spec.WorkspaceID,
			"MULTICA_LEASE_ID":     spec.LeaseID,
			"MULTICA_PROVIDER":     spec.Provider,
		},
		"target":             p.target,
		"autoStopInterval":   p.autoStopInterval,
		"autoDeleteInterval": p.autoDeleteInterval,
	}
	resp, err := p.postJSON(ctx, "/sandbox", payload)
	if err != nil {
		return nil, err
	}
	externalRef, _ := stringFromMap(resp, "id")
	return &ProvisionedRuntime{
		ExternalRef: externalRef,
		State:       inferLeaseState(resp, RuntimeLeaseStateProvisioning),
		Metadata:    resp,
	}, nil
}

func (p *daytonaProvisioner) Start(ctx context.Context, lease RuntimeLease) (*ProvisionedRuntime, error) {
	if lease.ExternalRef == nil || *lease.ExternalRef == "" {
		return p.Provision(ctx, RuntimeSpec{
			LeaseID:     lease.ID,
			WorkspaceID: lease.WorkspaceID,
			Name:        lease.Name,
			Scope:       lease.Scope,
			Placement:   lease.Placement,
			Provider:    lease.Provider,
			Backend:     lease.Backend,
			Config:      lease.Config,
			Billing:     lease.Billing,
		})
	}
	resp, err := p.postJSON(ctx, "/sandbox/"+*lease.ExternalRef+"/start", map[string]any{})
	if err != nil {
		return nil, err
	}
	return &ProvisionedRuntime{
		ExternalRef: *lease.ExternalRef,
		State:       inferLeaseState(resp, RuntimeLeaseStateProvisioning),
		Metadata:    resp,
	}, nil
}

func (p *daytonaProvisioner) Stop(ctx context.Context, lease RuntimeLease) error {
	if lease.ExternalRef == nil || *lease.ExternalRef == "" {
		return nil
	}
	_, err := p.postJSON(ctx, "/sandbox/"+*lease.ExternalRef+"/stop", map[string]any{})
	return err
}

func (p *daytonaProvisioner) Delete(ctx context.Context, lease RuntimeLease) error {
	if lease.ExternalRef == nil || *lease.ExternalRef == "" {
		return nil
	}
	_, err := p.postJSON(ctx, "/sandbox/"+*lease.ExternalRef+"/delete", map[string]any{})
	return err
}

type modalProvisioner struct {
	*httpProvisioner
	environment string
}

func NewModalProvisionerFromEnv() Provisioner {
	tokenID := strings.TrimSpace(os.Getenv("MODAL_TOKEN_ID"))
	tokenSecret := strings.TrimSpace(os.Getenv("MODAL_TOKEN_SECRET"))
	token := ""
	if tokenID != "" && tokenSecret != "" {
		token = tokenID + ":" + tokenSecret
	}

	return &modalProvisioner{
		httpProvisioner: newHTTPProvisioner(
			RuntimeBackendModal,
			firstNonEmpty(os.Getenv("MODAL_API_URL"), "https://api.modal.com"),
			token,
			"",
		),
		environment: firstNonEmpty(os.Getenv("MODAL_ENVIRONMENT"), "main"),
	}
}

func (p *modalProvisioner) Name() RuntimeBackend { return RuntimeBackendModal }

func (p *modalProvisioner) Provision(ctx context.Context, spec RuntimeSpec) (*ProvisionedRuntime, error) {
	payload := map[string]any{
		"name":             spec.Name,
		"environment_name": p.environment,
		"labels": map[string]any{
			"multica_lease_id": spec.LeaseID,
			"workspace_id":     spec.WorkspaceID,
			"provider":         spec.Provider,
			"placement":        spec.Placement,
		},
	}
	resp, err := p.postJSON(ctx, "/api/v1/sandbox/create", payload)
	if err != nil {
		return nil, err
	}
	externalRef, _ := stringFromMap(resp, "sandbox_id")
	return &ProvisionedRuntime{
		ExternalRef: externalRef,
		State:       inferLeaseState(resp, RuntimeLeaseStateProvisioning),
		Metadata:    resp,
	}, nil
}

func (p *modalProvisioner) Start(ctx context.Context, lease RuntimeLease) (*ProvisionedRuntime, error) {
	if lease.ExternalRef == nil || *lease.ExternalRef == "" {
		return p.Provision(ctx, RuntimeSpec{
			LeaseID:     lease.ID,
			WorkspaceID: lease.WorkspaceID,
			Name:        lease.Name,
			Scope:       lease.Scope,
			Placement:   lease.Placement,
			Provider:    lease.Provider,
			Backend:     lease.Backend,
			Config:      lease.Config,
			Billing:     lease.Billing,
		})
	}
	resp, err := p.postJSON(ctx, "/api/v1/sandbox/"+*lease.ExternalRef+"/resume", map[string]any{})
	if err != nil {
		return nil, err
	}
	return &ProvisionedRuntime{
		ExternalRef: *lease.ExternalRef,
		State:       inferLeaseState(resp, RuntimeLeaseStateProvisioning),
		Metadata:    resp,
	}, nil
}

func (p *modalProvisioner) Stop(ctx context.Context, lease RuntimeLease) error {
	if lease.ExternalRef == nil || *lease.ExternalRef == "" {
		return nil
	}
	_, err := p.postJSON(ctx, "/api/v1/sandbox/"+*lease.ExternalRef+"/terminate", map[string]any{})
	return err
}

func (p *modalProvisioner) Delete(ctx context.Context, lease RuntimeLease) error {
	return p.Stop(ctx, lease)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringFromMap(data map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		s, ok := value.(string)
		if ok && s != "" {
			return s, true
		}
	}
	return "", false
}

func firstNonZeroInt(raw string, fallback int) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value == 0 {
		return fallback
	}
	return value
}

func inferLeaseState(payload map[string]any, fallback RuntimeLeaseState) RuntimeLeaseState {
	for _, key := range []string{"state", "status", "desiredState"} {
		raw, ok := stringFromMap(payload, key)
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "started", "running", "active", "ready", "resumed":
			return RuntimeLeaseStateActive
		case "stopped", "paused", "suspended", "archived", "terminated":
			return RuntimeLeaseStateStopped
		case "failed", "error":
			return RuntimeLeaseStateFailed
		case "creating", "pending", "provisioning", "starting":
			return RuntimeLeaseStateProvisioning
		}
	}
	return fallback
}
