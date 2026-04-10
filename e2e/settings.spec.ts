import { test, expect } from "@playwright/test";
import { loginAsDefault } from "./helpers";

test.describe("Settings", () => {
  test("updating workspace name reflects in sidebar immediately", async ({
    page,
  }) => {
    await loginAsDefault(page);

    const workspaceTrigger = page.locator('[data-slot="dropdown-menu-trigger"]').first();

    // Navigate to settings
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await page.waitForURL("**/settings");
    await page.getByRole("tab", { name: "General" }).click();
    await expect(page.getByRole("heading", { name: "General" })).toBeVisible();

    // Change workspace name
    const nameInput = page
      .getByRole("tabpanel", { name: "General" })
      .getByRole("textbox")
      .first();
    const originalName = await nameInput.inputValue();
    const newName = "Renamed WS " + Date.now();
    await nameInput.fill(newName);

    // Save
    await page.getByRole("button", { name: "Save" }).click();

    // Sidebar should reflect the new name WITHOUT page refresh
    await expect(workspaceTrigger).toContainText(newName);

    // Restore original name so other tests aren't affected
    await nameInput.fill(originalName.trim());
    await page.getByRole("button", { name: "Save" }).click();
    await expect(workspaceTrigger).toContainText(originalName.trim());
  });
});
