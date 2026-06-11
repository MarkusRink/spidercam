import { test, expect } from "@playwright/test";

test.describe("host dashboard", () => {
  test("loads host start form", async ({ page }) => {
    await page.goto("/host.html");
    await expect(page.getByText("host / mixer")).toBeVisible();
    await expect(page.locator("#startBtn")).toBeVisible();
  });

  test("starts host mixer with local media", async ({ page }) => {
    await page.goto("/host.html");
    await page.locator("#seatCount").fill("8");
    await page.locator("#hostSeat").fill("0");
    await page.locator("#startBtn").click();

    await expect(page.getByText("spidercam / host")).toBeVisible({ timeout: 15_000 });
    await expect(page.locator("#hostPreview")).toBeVisible();
    await expect(page.locator("#outputCanvas")).toBeVisible();
    await expect(page.getByText("OUT")).toBeVisible();
    await expect(page.getByText("STREAMS")).toBeVisible();
    await expect(page.locator("#metricsTable")).toBeVisible();
  });
});
