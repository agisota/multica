import { type Page } from "@playwright/test";
import { TestApiClient } from "./fixtures";

const DEFAULT_E2E_NAME = "E2E User";
const DEFAULT_E2E_EMAIL = "e2e@multica.ai";
const DEFAULT_E2E_WORKSPACE = "e2e-workspace";

interface DefaultSession {
  token: string;
  workspaceId: string | null;
}

let defaultSessionPromise: Promise<DefaultSession> | null = null;

async function getDefaultSession(): Promise<DefaultSession> {
  if (!defaultSessionPromise) {
    defaultSessionPromise = (async () => {
      const api = new TestApiClient();
      await api.login(DEFAULT_E2E_EMAIL, DEFAULT_E2E_NAME);
      await api.ensureWorkspace("E2E Workspace", DEFAULT_E2E_WORKSPACE);

      return {
        token: api.getToken() ?? "",
        workspaceId: api.getWorkspaceId(),
      };
    })().catch((error) => {
      defaultSessionPromise = null;
      throw error;
    });
  }

  return defaultSessionPromise;
}

/**
 * Log in as the default E2E user and ensure the workspace exists first.
 * Authenticates via API (send-code → DB read → verify-code), then injects
 * the token into localStorage so the browser session is authenticated.
 */
export async function loginAsDefault(page: Page) {
  const { token, workspaceId } = await getDefaultSession();

  await page.addInitScript(({ token: t, workspaceId: wsId }) => {
    localStorage.setItem("multica_token", t);
    document.cookie = "multica_logged_in=1; path=/; samesite=lax";
    if (wsId) {
      localStorage.setItem("multica_workspace_id", wsId);
    }
  }, { token, workspaceId });

  await page.goto("/issues");
  await page.waitForURL("**/issues", { timeout: 10000 });
}

/**
 * Create a TestApiClient logged in as the default E2E user.
 * Call api.cleanup() in afterEach to remove test data created during the test.
 */
export async function createTestApi(): Promise<TestApiClient> {
  const { token, workspaceId } = await getDefaultSession();
  const api = new TestApiClient();
  api.setAuth(token, workspaceId);
  return api;
}

export async function openWorkspaceMenu(page: Page) {
  const workspaceTrigger = page.locator('[data-slot="dropdown-menu-trigger"]').first();
  await workspaceTrigger.waitFor({ state: "visible" });
  await workspaceTrigger.click();
  await page.getByText("Workspaces", { exact: true }).waitFor({ state: "visible" });
}
