import { test, expect, type Browser } from "@playwright/test";

async function startHost(browser: Browser) {
  const context = await browser.newContext({
    permissions: ["camera", "microphone"],
  });
  const page = await context.newPage();
  await page.goto("/host.html");
  await page.locator("#startBtn").click();
  await expect(page.getByText("spidercam / host")).toBeVisible({ timeout: 15_000 });
  return { context, page };
}

async function connectParticipant(browser: Browser, name: string, seat: string) {
  const context = await browser.newContext({
    permissions: ["camera", "microphone"],
  });
  const page = await context.newPage();
  await page.goto("/");
  await page.locator("#name").fill(name);
  await page.locator("#seat").selectOption(seat);
  await page.locator("#connectBtn").click();
  await expect(page.getByText(/connected/i)).toBeVisible({ timeout: 15_000 });
  return { context, page };
}

test.describe("room flow", () => {
  test("host sees participant join", async ({ browser }) => {
    const host = await startHost(browser);
    const participant = await connectParticipant(browser, "desk-1", "3");

    await expect(host.page.getByText("desk-1")).toBeVisible({ timeout: 10_000 });
    await expect(host.page.getByText("1 participants")).toBeVisible();

    await participant.context.close();
    await host.context.close();
  });

  test("participant sees selection state from host", async ({ browser }) => {
    const host = await startHost(browser);
    const participant = await connectParticipant(browser, "speaker-1", "4");

    await expect(host.page.getByText("speaker-1")).toBeVisible({ timeout: 10_000 });

    await expect(participant.page.locator("#metricsTable, table")).toBeVisible();
    await expect(participant.page.getByText("active video")).toBeVisible();

    await participant.context.close();
    await host.context.close();
  });
});
