export type AgentStatus = "idle" | "working" | "blocked" | "error" | "offline";

export type AgentRuntimeMode = "local" | "cloud";

export type AgentVisibility = "workspace" | "private";

export interface RuntimeDevice {
  id: string;
  workspace_id: string;
  daemon_id: string | null;
  name: string;
  runtime_mode: AgentRuntimeMode;
  provider: string;
  status: "online" | "offline";
  device_info: string;
  metadata: Record<string, unknown>;
  owner_id: string | null;
  last_seen_at: string | null;
  created_at: string;
  updated_at: string;
}

export type AgentRuntime = RuntimeDevice;

export interface AgentTask {
  id: string;
  agent_id: string;
  runtime_id: string;
  lease_id?: string;
  execution_backend?: RuntimeBackend;
  dispatch_state?: string;
  external_execution_id?: string | null;
  issue_id: string;
  status: "queued" | "dispatched" | "running" | "completed" | "failed" | "cancelled";
  priority: number;
  dispatched_at: string | null;
  started_at: string | null;
  completed_at: string | null;
  result: unknown;
  error: string | null;
  created_at: string;
}

export interface Agent {
  id: string;
  workspace_id: string;
  runtime_id: string;
  name: string;
  description: string;
  instructions: string;
  avatar_url: string | null;
  runtime_mode: AgentRuntimeMode;
  runtime_config: Record<string, unknown>;
  visibility: AgentVisibility;
  status: AgentStatus;
  max_concurrent_tasks: number;
  owner_id: string | null;
  skills: Skill[];
  created_at: string;
  updated_at: string;
  archived_at: string | null;
  archived_by: string | null;
}

export interface CreateAgentRequest {
  name: string;
  description?: string;
  instructions?: string;
  avatar_url?: string;
  runtime_id?: string;
  runtime_config?: Record<string, unknown>;
  visibility?: AgentVisibility;
  max_concurrent_tasks?: number;
}

export interface UpdateAgentRequest {
  name?: string;
  description?: string;
  instructions?: string;
  avatar_url?: string;
  runtime_id?: string;
  runtime_config?: Record<string, unknown>;
  visibility?: AgentVisibility;
  status?: AgentStatus;
  max_concurrent_tasks?: number;
}

// Skills

export interface Skill {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  content: string;
  config: Record<string, unknown>;
  files: SkillFile[];
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface SkillFile {
  id: string;
  skill_id: string;
  path: string;
  content: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSkillRequest {
  name: string;
  description?: string;
  content?: string;
  config?: Record<string, unknown>;
  files?: { path: string; content: string }[];
}

export interface UpdateSkillRequest {
  name?: string;
  description?: string;
  content?: string;
  config?: Record<string, unknown>;
  files?: { path: string; content: string }[];
}

export interface SetAgentSkillsRequest {
  skill_ids: string[];
}

export type RuntimePingStatus = "pending" | "running" | "completed" | "failed" | "timeout";

export interface RuntimePing {
  id: string;
  runtime_id: string;
  status: RuntimePingStatus;
  output?: string;
  error?: string;
  duration_ms?: number;
  created_at: string;
  updated_at: string;
}

export interface RuntimeUsage {
  runtime_id: string;
  date: string;
  provider: string;
  model: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

export interface RuntimeHourlyActivity {
  hour: number;
  count: number;
}

export type RuntimePlacement = "shared" | "private" | "dedicated";

export type RuntimeBackend = "local" | "daytona" | "modal";

export type RuntimeExecutionMode = "local" | "dedicated" | "managed";

export type PrivateRuntimeScope = "user" | "workspace" | "project";

export type RuntimeLeaseState =
  | "requested"
  | "provisioning"
  | "active"
  | "stopped"
  | "failed"
  | "deleted";

export interface RuntimePolicy {
  workspace_id: string;
  shared_pool_enabled: boolean;
  private_runtime_scope: PrivateRuntimeScope;
  default_runtime_placement: RuntimePlacement;
  default_provider: string;
  allow_daytona: boolean;
  allow_modal: boolean;
  auto_stop_enabled: boolean;
  auto_delete_enabled: boolean;
  idle_ttl_minutes: number;
  max_private_runtimes_per_user: number;
  max_shared_runtimes: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface RuntimeBillingAccount {
  workspace_id: string;
  billing_enabled: boolean;
  currency: string;
  hard_limit_cents: number;
  soft_limit_cents: number;
  shared_hourly_rate_cents: number;
  private_hourly_rate_cents: number;
  token_markup_percent: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface RuntimeLease {
  id: string;
  workspace_id: string;
  runtime_id: string;
  owner_user_id: string;
  project_id: string;
  name: string;
  scope: PrivateRuntimeScope;
  placement: RuntimePlacement;
  provider: string;
  backend: RuntimeBackend;
  state: RuntimeLeaseState;
  external_ref: string | null;
  config: Record<string, unknown>;
  billing: Record<string, unknown>;
  metadata: Record<string, unknown>;
  last_used_at: string | null;
  expires_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateRuntimeLeaseRequest {
  runtime_id?: string;
  owner_user_id?: string;
  project_id?: string;
  name: string;
  scope?: PrivateRuntimeScope;
  placement?: RuntimePlacement;
  provider?: string;
  backend?: RuntimeBackend;
  state?: RuntimeLeaseState;
  external_ref?: string;
  config?: Record<string, unknown>;
  billing?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  expires_at?: string;
}

export interface UpdateRuntimePolicyRequest {
  shared_pool_enabled?: boolean;
  private_runtime_scope?: PrivateRuntimeScope;
  default_runtime_placement?: RuntimePlacement;
  default_provider?: string;
  allow_daytona?: boolean;
  allow_modal?: boolean;
  auto_stop_enabled?: boolean;
  auto_delete_enabled?: boolean;
  idle_ttl_minutes?: number;
  max_private_runtimes_per_user?: number;
  max_shared_runtimes?: number;
  metadata?: Record<string, unknown>;
}

export interface UpdateRuntimeBillingRequest {
  billing_enabled?: boolean;
  currency?: string;
  hard_limit_cents?: number;
  soft_limit_cents?: number;
  shared_hourly_rate_cents?: number;
  private_hourly_rate_cents?: number;
  token_markup_percent?: number;
  metadata?: Record<string, unknown>;
}

export type RuntimeUpdateStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "timeout";

export interface RuntimeUpdate {
  id: string;
  runtime_id: string;
  status: RuntimeUpdateStatus;
  target_version: string;
  output?: string;
  error?: string;
  created_at: string;
  updated_at: string;
}
