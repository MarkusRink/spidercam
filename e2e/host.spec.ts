import { expect, test } from "@playwright/test";
import {
  devScenario,
  gotoHost,
  mixerSlider,
  streamCard,
  waitForHostConfigMessage,
} from "./helpers.js";

test.describe("host console", () => {
  test("loads and connects to mock signaling", async ({ page }) => {
    await gotoHost(page);
    await expect(page.getByRole("button", { name: "copy URL" })).toBeVisible();
    await expect(page.getByText(/\d+ streams/)).toBeVisible();
  });

  test("UC-H3: shows OUT/REF meters, timeline, and on-air dot", async ({
    page,
  }) => {
    await gotoHost(page);

    await expect(page.getByText("OUT", { exact: true })).toBeVisible();
    await expect(page.getByText("REF", { exact: true })).toBeVisible();
    await expect(page.getByRole("img")).toBeVisible();

    const alice = streamCard(page, "Alice");
    await expect(alice).toBeVisible();
    await expect(
      alice.locator("span.rounded-full:not(.invisible)"),
    ).toBeVisible();

    const bob = streamCard(page, "Bob");
    await expect(
      bob.locator("span.rounded-full:not(.invisible)"),
    ).not.toBeVisible();
  });

  test("UC-H4: stream grid shows transport cells and score border", async ({
    page,
  }) => {
    await gotoHost(page);

    const alice = streamCard(page, "Alice");
    await expect(alice.getByText(/\d+ms/).first()).toBeVisible();
    await expect(alice.getByText(/fps/).first()).toBeVisible();
    await expect(alice.getByText("AV")).toBeVisible();

    const style = await alice.getAttribute("style");
    expect(style).toContain("color-mix");
    expect(style).toContain("var(--color-spider-accent)");
  });

  test("UC-H5: timeline grows when mixer state changes", async ({
    page,
    request,
  }) => {
    await gotoHost(page);

    const timeline = page.getByRole("img");
    await expect(timeline.locator("> div").first()).toBeVisible();
    const before = await timeline.locator("> div").count();

    await devScenario(request, "mixer-state/HOLD");
    await expect
      .poll(async () => timeline.locator("> div").count())
      .toBeGreaterThan(before);
  });

  test("UC-H6: settings sliders send config over WS", async ({ page }) => {
    const configPromise = waitForHostConfigMessage(
      page,
      (config) => config.referenceDuckDb === -6,
    );
    await gotoHost(page);

    const slider = mixerSlider(page, "Ducking");
    await slider.fill("-6");
    await configPromise;
  });

  test("UC-H6: stream card AEC toggle updates host state", async ({
    page,
    request,
  }) => {
    await gotoHost(page);

    await page.evaluate(() => {
      const card = [
        ...document.querySelectorAll('div[style*="--stream-card-w"]'),
      ].find((node) => node.textContent?.includes("Host"));
      const input = card?.querySelector(
        'input[type="checkbox"]',
      ) as HTMLInputElement | null;
      input?.click();
    });

    await expect
      .poll(async () => {
        const res = await request.get(
          "http://127.0.0.1:1235/api/v1/host/state",
        );
        const state = (await res.json()) as {
          metrics: { participantId: string; aecEnabled: boolean }[];
        };
        return state.metrics.find((m) => m.participantId === "host")
          ?.aecEnabled;
      })
      .toBe(true);
  });
});
