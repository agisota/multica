import { test, expect } from "@playwright/test";
import { loginAsDefault, openWorkspaceMenu } from "./helpers";

test.describe("Authentication", () => {
  test("login page renders correctly", async ({ page }) => {
    await page.goto("/login");

    await expect(page.getByText("Multica")).toBeVisible();
    await expect(
      page.getByText("Turn coding agents into real teammates"),
    ).toBeVisible();
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByRole("button", { name: "Continue" })).toBeVisible();
  });

  test("login and redirect to /issues", async ({ page }) => {
    await loginAsDefault(page);

    await expect(page).toHaveURL(/\/issues/);
    await expect(page.getByRole("link", { name: "Issues", exact: true })).toBeVisible();
  });

  test("unauthenticated user is redirected to /", async ({ page }) => {
    await page.goto("/login");
    await page.evaluate(() => {
      localStorage.removeItem("multica_token");
      localStorage.removeItem("multica_workspace_id");
      document.cookie = "multica_logged_in=; path=/; max-age=0; samesite=lax";
    });

    await page.goto("/issues");
    await page.waitForURL("**/", { timeout: 10000 });
  });

  test("logout redirects to /", async ({ page }) => {
    await loginAsDefault(page);

    // Open the workspace dropdown menu
    await openWorkspaceMenu(page);

    await page.getByText("Log out", { exact: true }).click();

    await page.waitForURL("**/", { timeout: 10000 });
    expect(page.url()).toMatch(/\/$/);
  });
});
