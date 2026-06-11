import { test, expect } from "@playwright/test";

test.describe("participant page", () => {
  test("loads connect form", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("participant / connect")).toBeVisible();
    await expect(page.locator("#name")).toBeVisible();
    await expect(page.locator("#connectBtn")).toBeVisible();
  });

  test("connects with fake media devices", async ({ page }) => {
    await page.goto("/");
    await page.locator("#name").fill("e2e-user");
    await page.locator("#seat").selectOption("2");
    await page.locator("#connectBtn").click();

    await expect(page.getByText(/connected/i)).toBeVisible({ timeout: 15_000 });
    await expect(page.locator("#preview")).toBeVisible();
    await expect(page.getByText("seat 2")).toBeVisible();
    await expect(page.getByText("signal latency")).toBeVisible();
  });

  test("disconnects cleanly", async ({ page }) => {
    await page.goto("/");
    await page.locator("#name").fill("leaver");
    await page.locator("#connectBtn").click();
    await expect(page.getByText(/connected/i)).toBeVisible({ timeout: 15_000 });

    await page.locator("#disconnectBtn").click();
    await expect(page.getByText("participant / connect")).toBeVisible();
    await expect(page.locator("#connectBtn")).toBeVisible();
  });
});
