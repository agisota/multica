package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/orchestration"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type RuntimePolicyResponse struct {
	WorkspaceID               string         `json:"workspace_id"`
	SharedPoolEnabled         bool           `json:"shared_pool_enabled"`
	PrivateRuntimeScope       string         `json:"private_runtime_scope"`
	DefaultRuntimePlacement   string         `json:"default_runtime_placement"`
	DefaultProvider           string         `json:"default_provider"`
	AllowDaytona              bool           `json:"allow_daytona"`
	AllowModal                bool           `json:"allow_modal"`
	AutoStopEnabled           bool           `json:"auto_stop_enabled"`
	AutoDeleteEnabled         bool           `json:"auto_delete_enabled"`
	IdleTTLMinutes            int32          `json:"idle_ttl_minutes"`
	MaxPrivateRuntimesPerUser int32          `json:"max_private_runtimes_per_user"`
	MaxSharedRuntimes         int32          `json:"max_shared_runtimes"`
	Metadata                  map[string]any `json:"metadata"`
	CreatedAt                 string         `json:"created_at"`
	UpdatedAt                 string         `json:"updated_at"`
}

type RuntimeBillingResponse struct {
	WorkspaceID            string         `json:"workspace_id"`
	BillingEnabled         bool           `json:"billing_enabled"`
	Currency               string         `json:"currency"`
	HardLimitCents         int64          `json:"hard_limit_cents"`
	SoftLimitCents         int64          `json:"soft_limit_cents"`
	SharedHourlyRateCents  int32          `json:"shared_hourly_rate_cents"`
	PrivateHourlyRateCents int32          `json:"private_hourly_rate_cents"`
	TokenMarkupPercent     int32          `json:"token_markup_percent"`
	Metadata               map[string]any `json:"metadata"`
	CreatedAt              string         `json:"created_at"`
	UpdatedAt              string         `json:"updated_at"`
}

type RuntimeLeaseResponse struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	RuntimeID   string         `json:"runtime_id"`
	OwnerUserID string         `json:"owner_user_id"`
	ProjectID   string         `json:"project_id"`
	Name        string         `json:"name"`
	Scope       string         `json:"scope"`
	Placement   string         `json:"placement"`
	Provider    string         `json:"provider"`
	Backend     string         `json:"backend"`
	State       string         `json:"state"`
	ExternalRef *string        `json:"external_ref"`
	Config      map[string]any `json:"config"`
	Billing     map[string]any `json:"billing"`
	Metadata    map[string]any `json:"metadata"`
	LastUsedAt  *string        `json:"last_used_at"`
	ExpiresAt   *string        `json:"expires_at"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

type UpdateRuntimePolicyRequest struct {
	SharedPoolEnabled         *bool          `json:"shared_pool_enabled"`
	PrivateRuntimeScope       *string        `json:"private_runtime_scope"`
	DefaultRuntimePlacement   *string        `json:"default_runtime_placement"`
	DefaultProvider           *string        `json:"default_provider"`
	AllowDaytona              *bool          `json:"allow_daytona"`
	AllowModal                *bool          `json:"allow_modal"`
	AutoStopEnabled           *bool          `json:"auto_stop_enabled"`
	AutoDeleteEnabled         *bool          `json:"auto_delete_enabled"`
	IdleTTLMinutes            *int32         `json:"idle_ttl_minutes"`
	MaxPrivateRuntimesPerUser *int32         `json:"max_private_runtimes_per_user"`
	MaxSharedRuntimes         *int32         `json:"max_shared_runtimes"`
	Metadata                  map[string]any `json:"metadata"`
}

type UpdateRuntimeBillingRequest struct {
	BillingEnabled         *bool          `json:"billing_enabled"`
	Currency               *string        `json:"currency"`
	HardLimitCents         *int64         `json:"hard_limit_cents"`
	SoftLimitCents         *int64         `json:"soft_limit_cents"`
	SharedHourlyRateCents  *int32         `json:"shared_hourly_rate_cents"`
	PrivateHourlyRateCents *int32         `json:"private_hourly_rate_cents"`
	TokenMarkupPercent     *int32         `json:"token_markup_percent"`
	Metadata               map[string]any `json:"metadata"`
}

type CreateRuntimeLeaseRequest struct {
	RuntimeID   *string        `json:"runtime_id"`
	OwnerUserID *string        `json:"owner_user_id"`
	ProjectID   *string        `json:"project_id"`
	Name        string         `json:"name"`
	Scope       string         `json:"scope"`
	Placement   string         `json:"placement"`
	Provider    string         `json:"provider"`
	Backend     string         `json:"backend"`
	State       string         `json:"state"`
	ExternalRef *string        `json:"external_ref"`
	Config      map[string]any `json:"config"`
	Billing     map[string]any `json:"billing"`
	Metadata    map[string]any `json:"metadata"`
	ExpiresAt   *string        `json:"expires_at"`
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil || data == nil {
		return map[string]any{}
	}
	return data
}

func encodeJSONMap(data map[string]any) []byte {
	if data == nil {
		return []byte("{}")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func runtimePolicyToResponse(policy db.RuntimePolicy) RuntimePolicyResponse {
	return RuntimePolicyResponse{
		WorkspaceID:               uuidToString(policy.WorkspaceID),
		SharedPoolEnabled:         policy.SharedPoolEnabled,
		PrivateRuntimeScope:       policy.PrivateRuntimeScope,
		DefaultRuntimePlacement:   policy.DefaultRuntimePlacement,
		DefaultProvider:           policy.DefaultProvider,
		AllowDaytona:              policy.AllowDaytona,
		AllowModal:                policy.AllowModal,
		AutoStopEnabled:           policy.AutoStopEnabled,
		AutoDeleteEnabled:         policy.AutoDeleteEnabled,
		IdleTTLMinutes:            policy.IdleTtlMinutes,
		MaxPrivateRuntimesPerUser: policy.MaxPrivateRuntimesPerUser,
		MaxSharedRuntimes:         policy.MaxSharedRuntimes,
		Metadata:                  decodeJSONMap(policy.Metadata),
		CreatedAt:                 timestampToString(policy.CreatedAt),
		UpdatedAt:                 timestampToString(policy.UpdatedAt),
	}
}

func runtimeBillingToResponse(billing db.RuntimeBillingAccount) RuntimeBillingResponse {
	return RuntimeBillingResponse{
		WorkspaceID:            uuidToString(billing.WorkspaceID),
		BillingEnabled:         billing.BillingEnabled,
		Currency:               billing.Currency,
		HardLimitCents:         billing.HardLimitCents,
		SoftLimitCents:         billing.SoftLimitCents,
		SharedHourlyRateCents:  billing.SharedHourlyRateCents,
		PrivateHourlyRateCents: billing.PrivateHourlyRateCents,
		TokenMarkupPercent:     billing.TokenMarkupPercent,
		Metadata:               decodeJSONMap(billing.Metadata),
		CreatedAt:              timestampToString(billing.CreatedAt),
		UpdatedAt:              timestampToString(billing.UpdatedAt),
	}
}

func runtimeLeaseToResponse(lease db.RuntimeLease) RuntimeLeaseResponse {
	return RuntimeLeaseResponse{
		ID:          uuidToString(lease.ID),
		WorkspaceID: uuidToString(lease.WorkspaceID),
		RuntimeID:   uuidToString(lease.RuntimeID),
		OwnerUserID: uuidToString(lease.OwnerUserID),
		ProjectID:   uuidToString(lease.ProjectID),
		Name:        lease.Name,
		Scope:       lease.Scope,
		Placement:   lease.Placement,
		Provider:    lease.Provider,
		Backend:     lease.Backend,
		State:       lease.State,
		ExternalRef: textToPtr(lease.ExternalRef),
		Config:      decodeJSONMap(lease.Config),
		Billing:     decodeJSONMap(lease.Billing),
		Metadata:    decodeJSONMap(lease.Metadata),
		LastUsedAt:  timestampToPtr(lease.LastUsedAt),
		ExpiresAt:   timestampToPtr(lease.ExpiresAt),
		CreatedAt:   timestampToString(lease.CreatedAt),
		UpdatedAt:   timestampToString(lease.UpdatedAt),
	}
}

func parseOptionalTimestamp(value *string) (pgtype.Timestamptz, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return pgtype.Timestamptz{}, nil
	}

	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: parsed, Valid: true}, nil
}

func (h *Handler) ensureRuntimePolicy(ctx context.Context, workspaceID pgtype.UUID) (db.RuntimePolicy, error) {
	policy, err := h.Queries.GetRuntimePolicy(ctx, workspaceID)
	if err == nil {
		return policy, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.RuntimePolicy{}, err
	}

	defaults := orchestration.DefaultRuntimePolicy(uuidToString(workspaceID))
	return h.Queries.UpsertRuntimePolicy(ctx, db.UpsertRuntimePolicyParams{
		WorkspaceID:               workspaceID,
		SharedPoolEnabled:         defaults.SharedPoolEnabled,
		PrivateRuntimeScope:       defaults.PrivateRuntimeScope,
		DefaultRuntimePlacement:   defaults.DefaultRuntimePlacement,
		DefaultProvider:           defaults.DefaultProvider,
		AllowDaytona:              defaults.AllowDaytona,
		AllowModal:                defaults.AllowModal,
		AutoStopEnabled:           defaults.AutoStopEnabled,
		AutoDeleteEnabled:         defaults.AutoDeleteEnabled,
		IdleTtlMinutes:            int32(defaults.IdleTTLMinutes),
		MaxPrivateRuntimesPerUser: int32(defaults.MaxPrivateRuntimesPerUser),
		MaxSharedRuntimes:         int32(defaults.MaxSharedRuntimes),
		Metadata:                  defaults.Metadata,
	})
}

func (h *Handler) ensureRuntimeBilling(ctx context.Context, workspaceID pgtype.UUID) (db.RuntimeBillingAccount, error) {
	billing, err := h.Queries.GetRuntimeBillingAccount(ctx, workspaceID)
	if err == nil {
		return billing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.RuntimeBillingAccount{}, err
	}

	return h.Queries.UpsertRuntimeBillingAccount(ctx, db.UpsertRuntimeBillingAccountParams{
		WorkspaceID:            workspaceID,
		BillingEnabled:         false,
		Currency:               "usd",
		HardLimitCents:         0,
		SoftLimitCents:         0,
		SharedHourlyRateCents:  0,
		PrivateHourlyRateCents: 0,
		TokenMarkupPercent:     0,
		Metadata:               []byte("{}"),
	})
}

func validateLeasePlacement(placement string) bool {
	switch placement {
	case string(orchestration.RuntimePlacementShared), string(orchestration.RuntimePlacementPrivate), string(orchestration.RuntimePlacementDedicated):
		return true
	default:
		return false
	}
}

func validateLeaseScope(scope string) bool {
	switch scope {
	case string(orchestration.PrivateRuntimeScopeUser), string(orchestration.PrivateRuntimeScopeWorkspace), string(orchestration.PrivateRuntimeScopeProject):
		return true
	default:
		return false
	}
}

func validateLeaseBackend(backend string) bool {
	switch backend {
	case string(orchestration.RuntimeBackendLocal), string(orchestration.RuntimeBackendModal), string(orchestration.RuntimeBackendDaytona):
		return true
	default:
		return false
	}
}

func runtimeLeaseBackendSupported(backend string) bool {
	switch backend {
	case string(orchestration.RuntimeBackendLocal), string(orchestration.RuntimeBackendModal):
		return true
	default:
		return false
	}
}

func localLeaseDaemonID(lease db.RuntimeLease) string {
	return "lease:" + uuidToString(lease.ID)
}

func localLeaseDeviceInfo(lease db.RuntimeLease) string {
	switch lease.Placement {
	case string(orchestration.RuntimePlacementShared):
		return "Shared control-plane runtime"
	case string(orchestration.RuntimePlacementDedicated):
		return "Dedicated control-plane runtime"
	default:
		return "Private control-plane runtime"
	}
}

func mergeRuntimeMetadata(lease db.RuntimeLease, provisioned *orchestration.ProvisionedRuntime) []byte {
	metadata := decodeJSONMap(lease.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}

	if provisioned != nil {
		for key, value := range provisioned.Metadata {
			metadata[key] = value
		}
		metadata["lease_state"] = string(provisioned.State)
	}

	metadata["lease_id"] = uuidToString(lease.ID)
	metadata["lease_backend"] = lease.Backend
	metadata["lease_scope"] = lease.Scope
	metadata["lease_placement"] = lease.Placement
	metadata["source"] = "runtime_lease"

	return encodeJSONMap(metadata)
}

func (h *Handler) upsertLocalLeaseRuntime(ctx context.Context, lease db.RuntimeLease, provisioned *orchestration.ProvisionedRuntime) (db.AgentRuntime, error) {
	return h.Queries.UpsertAgentRuntime(ctx, db.UpsertAgentRuntimeParams{
		WorkspaceID: lease.WorkspaceID,
		DaemonID:    strToText(localLeaseDaemonID(lease)),
		Name:        lease.Name,
		RuntimeMode: "local",
		Provider:    lease.Provider,
		Status:      "online",
		DeviceInfo:  localLeaseDeviceInfo(lease),
		Metadata:    mergeRuntimeMetadata(lease, provisioned),
		OwnerID:     lease.OwnerUserID,
	})
}

func (h *Handler) setLocalLeaseRuntimeOffline(ctx context.Context, lease db.RuntimeLease) error {
	if !lease.RuntimeID.Valid {
		return nil
	}
	return h.Queries.SetAgentRuntimeOffline(ctx, lease.RuntimeID)
}

func (h *Handler) GetRuntimePolicy(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.workspaceMember(w, r, workspaceID); !ok {
		return
	}

	policy, err := h.ensureRuntimePolicy(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load runtime policy")
		return
	}

	writeJSON(w, http.StatusOK, runtimePolicyToResponse(policy))
}

func (h *Handler) UpdateRuntimePolicy(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin"); !ok {
		return
	}

	current, err := h.ensureRuntimePolicy(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load runtime policy")
		return
	}

	var req UpdateRuntimePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.PrivateRuntimeScope != nil && !validateLeaseScope(*req.PrivateRuntimeScope) {
		writeError(w, http.StatusBadRequest, "invalid private_runtime_scope")
		return
	}
	if req.DefaultRuntimePlacement != nil && !validateLeasePlacement(*req.DefaultRuntimePlacement) {
		writeError(w, http.StatusBadRequest, "invalid default_runtime_placement")
		return
	}

	params := db.UpsertRuntimePolicyParams{
		WorkspaceID:               current.WorkspaceID,
		SharedPoolEnabled:         current.SharedPoolEnabled,
		PrivateRuntimeScope:       current.PrivateRuntimeScope,
		DefaultRuntimePlacement:   current.DefaultRuntimePlacement,
		DefaultProvider:           current.DefaultProvider,
		AllowDaytona:              current.AllowDaytona,
		AllowModal:                current.AllowModal,
		AutoStopEnabled:           current.AutoStopEnabled,
		AutoDeleteEnabled:         current.AutoDeleteEnabled,
		IdleTtlMinutes:            current.IdleTtlMinutes,
		MaxPrivateRuntimesPerUser: current.MaxPrivateRuntimesPerUser,
		MaxSharedRuntimes:         current.MaxSharedRuntimes,
		Metadata:                  current.Metadata,
	}

	if req.SharedPoolEnabled != nil {
		params.SharedPoolEnabled = *req.SharedPoolEnabled
	}
	if req.PrivateRuntimeScope != nil {
		params.PrivateRuntimeScope = strings.TrimSpace(*req.PrivateRuntimeScope)
	}
	if req.DefaultRuntimePlacement != nil {
		params.DefaultRuntimePlacement = strings.TrimSpace(*req.DefaultRuntimePlacement)
	}
	if req.DefaultProvider != nil && strings.TrimSpace(*req.DefaultProvider) != "" {
		params.DefaultProvider = strings.TrimSpace(*req.DefaultProvider)
	}
	if req.AllowDaytona != nil {
		params.AllowDaytona = *req.AllowDaytona
	}
	if req.AllowModal != nil {
		params.AllowModal = *req.AllowModal
	}
	if req.AutoStopEnabled != nil {
		params.AutoStopEnabled = *req.AutoStopEnabled
	}
	if req.AutoDeleteEnabled != nil {
		params.AutoDeleteEnabled = *req.AutoDeleteEnabled
	}
	if req.IdleTTLMinutes != nil {
		params.IdleTtlMinutes = *req.IdleTTLMinutes
	}
	if req.MaxPrivateRuntimesPerUser != nil {
		params.MaxPrivateRuntimesPerUser = *req.MaxPrivateRuntimesPerUser
	}
	if req.MaxSharedRuntimes != nil {
		params.MaxSharedRuntimes = *req.MaxSharedRuntimes
	}
	if req.Metadata != nil {
		params.Metadata = encodeJSONMap(req.Metadata)
	}

	policy, err := h.Queries.UpsertRuntimePolicy(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update runtime policy")
		return
	}

	writeJSON(w, http.StatusOK, runtimePolicyToResponse(policy))
}

func (h *Handler) GetRuntimeBilling(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.workspaceMember(w, r, workspaceID); !ok {
		return
	}

	billing, err := h.ensureRuntimeBilling(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load runtime billing")
		return
	}

	writeJSON(w, http.StatusOK, runtimeBillingToResponse(billing))
}

func (h *Handler) UpdateRuntimeBilling(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin"); !ok {
		return
	}

	current, err := h.ensureRuntimeBilling(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load runtime billing")
		return
	}

	var req UpdateRuntimeBillingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	params := db.UpsertRuntimeBillingAccountParams{
		WorkspaceID:            current.WorkspaceID,
		BillingEnabled:         current.BillingEnabled,
		Currency:               current.Currency,
		HardLimitCents:         current.HardLimitCents,
		SoftLimitCents:         current.SoftLimitCents,
		SharedHourlyRateCents:  current.SharedHourlyRateCents,
		PrivateHourlyRateCents: current.PrivateHourlyRateCents,
		TokenMarkupPercent:     current.TokenMarkupPercent,
		Metadata:               current.Metadata,
	}

	if req.BillingEnabled != nil {
		params.BillingEnabled = *req.BillingEnabled
	}
	if req.Currency != nil && strings.TrimSpace(*req.Currency) != "" {
		params.Currency = strings.ToLower(strings.TrimSpace(*req.Currency))
	}
	if req.HardLimitCents != nil {
		params.HardLimitCents = *req.HardLimitCents
	}
	if req.SoftLimitCents != nil {
		params.SoftLimitCents = *req.SoftLimitCents
	}
	if req.SharedHourlyRateCents != nil {
		params.SharedHourlyRateCents = *req.SharedHourlyRateCents
	}
	if req.PrivateHourlyRateCents != nil {
		params.PrivateHourlyRateCents = *req.PrivateHourlyRateCents
	}
	if req.TokenMarkupPercent != nil {
		params.TokenMarkupPercent = *req.TokenMarkupPercent
	}
	if req.Metadata != nil {
		params.Metadata = encodeJSONMap(req.Metadata)
	}

	billing, err := h.Queries.UpsertRuntimeBillingAccount(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update runtime billing")
		return
	}

	writeJSON(w, http.StatusOK, runtimeBillingToResponse(billing))
}

func (h *Handler) ListRuntimeLeases(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	member, ok := h.workspaceMember(w, r, workspaceID)
	if !ok {
		return
	}

	var (
		leases []db.RuntimeLease
		err    error
	)
	if r.URL.Query().Get("owner") == "me" {
		leases, err = h.Queries.ListRuntimeLeasesByOwner(r.Context(), db.ListRuntimeLeasesByOwnerParams{
			WorkspaceID: parseUUID(workspaceID),
			OwnerUserID: member.UserID,
		})
	} else {
		leases, err = h.Queries.ListRuntimeLeases(r.Context(), parseUUID(workspaceID))
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runtime leases")
		return
	}

	resp := make([]RuntimeLeaseResponse, 0, len(leases))
	for _, lease := range leases {
		resp = append(resp, runtimeLeaseToResponse(lease))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetRuntimeLease(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.workspaceMember(w, r, workspaceID); !ok {
		return
	}

	lease, err := h.Queries.GetRuntimeLeaseInWorkspace(r.Context(), db.GetRuntimeLeaseInWorkspaceParams{
		ID:          parseUUID(chi.URLParam(r, "leaseId")),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "runtime lease not found")
		return
	}

	writeJSON(w, http.StatusOK, runtimeLeaseToResponse(lease))
}

func (h *Handler) CreateRuntimeLease(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	member, ok := h.workspaceMember(w, r, workspaceID)
	if !ok {
		return
	}

	policy, err := h.ensureRuntimePolicy(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load runtime policy")
		return
	}
	billing, err := h.ensureRuntimeBilling(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load runtime billing")
		return
	}

	var req CreateRuntimeLeaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = policy.PrivateRuntimeScope
	}
	if !validateLeaseScope(scope) {
		writeError(w, http.StatusBadRequest, "invalid scope")
		return
	}

	placement := strings.TrimSpace(req.Placement)
	if placement == "" {
		placement = policy.DefaultRuntimePlacement
	}
	if !validateLeasePlacement(placement) {
		writeError(w, http.StatusBadRequest, "invalid placement")
		return
	}

	backend := strings.TrimSpace(req.Backend)
	if backend == "" {
		backend = string(orchestration.RuntimeBackendLocal)
	}
	if !validateLeaseBackend(backend) {
		writeError(w, http.StatusBadRequest, "invalid backend")
		return
	}

	if backend == string(orchestration.RuntimeBackendModal) && !policy.AllowModal {
		writeError(w, http.StatusConflict, "modal runtimes are disabled")
		return
	}
	if backend == string(orchestration.RuntimeBackendDaytona) && !policy.AllowDaytona {
		writeError(w, http.StatusConflict, "daytona runtimes are disabled")
		return
	}
	if !runtimeLeaseBackendSupported(backend) {
		writeError(w, http.StatusConflict, "managed runtime provisioning is not wired yet; use local backend")
		return
	}

	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = policy.DefaultProvider
	}

	state := strings.TrimSpace(req.State)
	if state == "" {
		state = string(orchestration.RuntimeLeaseStateRequested)
	}

	expiresAt, err := parseOptionalTimestamp(req.ExpiresAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid expires_at")
		return
	}

	ownerUserID := member.UserID
	if req.OwnerUserID != nil && strings.TrimSpace(*req.OwnerUserID) != "" {
		ownerUserID = parseUUID(*req.OwnerUserID)
	}

	lease, err := h.Queries.CreateRuntimeLease(r.Context(), db.CreateRuntimeLeaseParams{
		WorkspaceID: parseUUID(workspaceID),
		RuntimeID:   parseUUID(strings.TrimSpace(textValue(req.RuntimeID))),
		OwnerUserID: ownerUserID,
		ProjectID:   parseUUID(strings.TrimSpace(textValue(req.ProjectID))),
		Name:        strings.TrimSpace(req.Name),
		Scope:       scope,
		Placement:   placement,
		Provider:    provider,
		Backend:     backend,
		State:       state,
		ExternalRef: ptrToText(req.ExternalRef),
		Config:      encodeJSONMap(req.Config),
		Billing:     encodeJSONMap(req.Billing),
		Metadata:    encodeJSONMap(req.Metadata),
		LastUsedAt:  pgtype.Timestamptz{},
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create runtime lease")
		return
	}

	if len(lease.Billing) == 0 || string(lease.Billing) == "{}" {
		lease, err = h.Queries.UpdateRuntimeLease(r.Context(), db.UpdateRuntimeLeaseParams{
			ID:          lease.ID,
			WorkspaceID: lease.WorkspaceID,
			Billing: orchestration.DefaultLeaseBilling(orchestration.RuntimeBillingAccount{
				WorkspaceID:            uuidToString(billing.WorkspaceID),
				BillingEnabled:         billing.BillingEnabled,
				Currency:               billing.Currency,
				HardLimitCents:         billing.HardLimitCents,
				SoftLimitCents:         billing.SoftLimitCents,
				SharedHourlyRateCents:  int(billing.SharedHourlyRateCents),
				PrivateHourlyRateCents: int(billing.PrivateHourlyRateCents),
				TokenMarkupPercent:     int(billing.TokenMarkupPercent),
				Metadata:               billing.Metadata,
			}, placement),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to finalize runtime lease")
			return
		}
	}

	writeJSON(w, http.StatusCreated, runtimeLeaseToResponse(lease))
}

func (h *Handler) StartRuntimeLease(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.workspaceMember(w, r, workspaceID); !ok {
		return
	}

	lease, err := h.Queries.GetRuntimeLeaseInWorkspace(r.Context(), db.GetRuntimeLeaseInWorkspaceParams{
		ID:          parseUUID(chi.URLParam(r, "leaseId")),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "runtime lease not found")
		return
	}

	provisioner, err := h.Orchestration.Provisioners().Get(lease.Backend)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !runtimeLeaseBackendSupported(lease.Backend) {
		writeError(w, http.StatusConflict, "managed runtime provisioning is not wired yet; use local backend")
		return
	}

	provisioned, err := provisioner.Start(r.Context(), orchestration.RuntimeLease{
		ID:          uuidToString(lease.ID),
		WorkspaceID: uuidToString(lease.WorkspaceID),
		RuntimeID:   uuidToPtr(lease.RuntimeID),
		OwnerUserID: uuidToPtr(lease.OwnerUserID),
		ProjectID:   uuidToPtr(lease.ProjectID),
		Name:        lease.Name,
		Scope:       lease.Scope,
		Placement:   lease.Placement,
		Provider:    lease.Provider,
		Backend:     lease.Backend,
		State:       lease.State,
		ExternalRef: textToPtr(lease.ExternalRef),
		Config:      lease.Config,
		Billing:     lease.Billing,
		Metadata:    lease.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	runtimeID := pgtype.UUID{}
	if lease.Backend == string(orchestration.RuntimeBackendLocal) {
		runtime, err := h.upsertLocalLeaseRuntime(r.Context(), lease, provisioned)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to register local runtime lease")
			return
		}
		runtimeID = runtime.ID
	}

	updated, err := h.Queries.UpdateRuntimeLease(r.Context(), db.UpdateRuntimeLeaseParams{
		ID:          lease.ID,
		WorkspaceID: lease.WorkspaceID,
		RuntimeID:   runtimeID,
		State:       strToText(string(provisioned.State)),
		ExternalRef: strToText(provisioned.ExternalRef),
		Metadata:    mergeRuntimeMetadata(lease, provisioned),
		LastUsedAt:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update runtime lease")
		return
	}

	writeJSON(w, http.StatusOK, runtimeLeaseToResponse(updated))
}

func (h *Handler) StopRuntimeLease(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.workspaceMember(w, r, workspaceID); !ok {
		return
	}

	lease, err := h.Queries.GetRuntimeLeaseInWorkspace(r.Context(), db.GetRuntimeLeaseInWorkspaceParams{
		ID:          parseUUID(chi.URLParam(r, "leaseId")),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "runtime lease not found")
		return
	}

	provisioner, err := h.Orchestration.Provisioners().Get(lease.Backend)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := provisioner.Stop(r.Context(), orchestration.RuntimeLease{
		ID:          uuidToString(lease.ID),
		WorkspaceID: uuidToString(lease.WorkspaceID),
		RuntimeID:   uuidToPtr(lease.RuntimeID),
		OwnerUserID: uuidToPtr(lease.OwnerUserID),
		ProjectID:   uuidToPtr(lease.ProjectID),
		Name:        lease.Name,
		Scope:       lease.Scope,
		Placement:   lease.Placement,
		Provider:    lease.Provider,
		Backend:     lease.Backend,
		State:       lease.State,
		ExternalRef: textToPtr(lease.ExternalRef),
		Config:      lease.Config,
		Billing:     lease.Billing,
		Metadata:    lease.Metadata,
	}); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if lease.Backend == string(orchestration.RuntimeBackendLocal) {
		if err := h.setLocalLeaseRuntimeOffline(r.Context(), lease); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update local runtime lease")
			return
		}
	}

	updated, err := h.Queries.UpdateRuntimeLease(r.Context(), db.UpdateRuntimeLeaseParams{
		ID:          lease.ID,
		WorkspaceID: lease.WorkspaceID,
		State:       strToText(string(orchestration.RuntimeLeaseStateStopped)),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update runtime lease")
		return
	}

	writeJSON(w, http.StatusOK, runtimeLeaseToResponse(updated))
}

func (h *Handler) DeleteRuntimeLease(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin"); !ok {
		return
	}

	lease, err := h.Queries.GetRuntimeLeaseInWorkspace(r.Context(), db.GetRuntimeLeaseInWorkspaceParams{
		ID:          parseUUID(chi.URLParam(r, "leaseId")),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "runtime lease not found")
		return
	}

	provisioner, err := h.Orchestration.Provisioners().Get(lease.Backend)
	if err == nil {
		_ = provisioner.Delete(r.Context(), orchestration.RuntimeLease{
			ID:          uuidToString(lease.ID),
			WorkspaceID: uuidToString(lease.WorkspaceID),
			RuntimeID:   uuidToPtr(lease.RuntimeID),
			OwnerUserID: uuidToPtr(lease.OwnerUserID),
			ProjectID:   uuidToPtr(lease.ProjectID),
			Name:        lease.Name,
			Scope:       lease.Scope,
			Placement:   lease.Placement,
			Provider:    lease.Provider,
			Backend:     lease.Backend,
			State:       lease.State,
			ExternalRef: textToPtr(lease.ExternalRef),
			Config:      lease.Config,
			Billing:     lease.Billing,
			Metadata:    lease.Metadata,
		})
	}
	if lease.Backend == string(orchestration.RuntimeBackendLocal) {
		_ = h.setLocalLeaseRuntimeOffline(r.Context(), lease)
	}

	updated, err := h.Queries.UpdateRuntimeLease(r.Context(), db.UpdateRuntimeLeaseParams{
		ID:          lease.ID,
		WorkspaceID: lease.WorkspaceID,
		State:       strToText(string(orchestration.RuntimeLeaseStateDeleted)),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete runtime lease")
		return
	}

	writeJSON(w, http.StatusOK, runtimeLeaseToResponse(updated))
}

func textValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
