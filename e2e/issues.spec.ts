import { test, expect } from "@playwright/test";
import { loginAsDefault, createTestApi } from "./helpers";
import type { TestApiClient } from "./fixtures";

const issuesViewToggle = 'button:has(svg.lucide-columns3), button:has(svg.lucide-list)';

test.describe("Issues", () => {
  let api: TestApiClient;

  test.beforeEach(async ({ page }) => {
    api = await createTestApi();
    await loginAsDefault(page);
  });

  test.afterEach(async () => {
    await api.cleanup();
  });

  test("issues page loads with board view", async ({ page }) => {
    await expect(page.locator("button:has(svg.lucide-columns3)")).toBeVisible();

    // Board columns should be visible
    await expect(page.locator("text=Backlog")).toBeVisible();
    await expect(page.locator("text=Todo")).toBeVisible();
    await expect(page.locator("text=In Progress")).toBeVisible();
  });

  test("can switch between board and list view", async ({ page }) => {
    await expect(page.locator("button:has(svg.lucide-columns3)")).toBeVisible();

    // Switch to list view
    await page.locator(issuesViewToggle).click();
    await page.getByRole("menuitem", { name: "List" }).click();
    await expect(page.locator("button:has(svg.lucide-list)")).toBeVisible();

    // Switch back to board view
    await page.locator(issuesViewToggle).click();
    await page.getByRole("menuitem", { name: "Board" }).click();
    await expect(page.locator("button:has(svg.lucide-columns3)")).toBeVisible();
  });

  test("can create a new issue", async ({ page }) => {
    await page.getByRole("button", { name: "New Issue" }).click();
    const dialog = page.getByRole("dialog", { name: "New Issue" });
    await expect(dialog).toBeVisible();

    const title = "E2E Created " + Date.now();
    await dialog.getByRole("textbox", { name: "Issue title" }).fill(title);
    await dialog.getByRole("button", { name: "Create Issue" }).click();

    // New issue should appear on the page
    await expect(page.locator(`text=${title}`).first()).toBeVisible({
      timeout: 10000,
    });
  });

  test("can navigate to issue detail page", async ({ page }) => {
    // Create a known issue via API so the test controls its own fixture
    const issue = await api.createIssue("E2E Detail Test " + Date.now());

    // Reload to see the new issue
    await page.reload();
    await expect(page.locator("button:has(svg.lucide-columns3)")).toBeVisible();

    // Navigate to the issue detail
    const issueLink = page.locator(`main a[href="/issues/${issue.id}"]`);
    await expect(issueLink).toBeVisible({ timeout: 5000 });
    await issueLink.click();

    await page.waitForURL(/\/issues\/[\w-]+/);

    // Should show Properties panel
    await expect(page.locator("text=Properties")).toBeVisible();
    // Should show breadcrumb link back to Issues
    await expect(
      page.locator("a", { hasText: "Issues" }).first(),
    ).toBeVisible();
  });

  test("can cancel issue creation", async ({ page }) => {
    await page.getByRole("button", { name: "New Issue" }).click();

    const dialog = page.getByRole("dialog", { name: "New Issue" });
    await expect(dialog.getByRole("textbox", { name: "Issue title" })).toBeVisible();

    await page.keyboard.press("Escape");

    await expect(dialog).not.toBeVisible();
    await expect(page.getByRole("button", { name: "New Issue" })).toBeVisible();
  });
});
