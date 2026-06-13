import { expect, test } from "@playwright/test";
import {
  connectParticipant,
  devScenario,
  participantClientId,
} from "./helpers.js";

test.describe("participant monitor", () => {
  test("UC-P1: single-screen layout with device pickers and connect toggle", async ({
    page,
  }) => {
    await page.goto("http://127.0.0.1:4174/");
    await expect(
      page.getByRole("textbox", { name: "Display name" }),
    ).toBeVisible();
    await expect(page.getByText("Microphone")).toBeVisible();
    await expect(page.getByText("Camera")).toBeVisible();
    await expect(page.getByRole("button", { name: "Connect" })).toBeVisible();
    await expect(page.getByRole("combobox").nth(0)).toBeVisible();
    await expect(page.getByRole("combobox").nth(1)).toBeVisible();
  });

  test("participant connects to mock room", async ({ page }) => {
    await connectParticipant(page, "e2e-user");
    await expect(
      page.getByRole("button", { name: "Disconnect" }),
    ).toBeVisible();
  });

  test("UC-P2: shows loop delay when connected", async ({ page }) => {
    await connectParticipant(page, "delay-user");
    await expect(page.getByText(/Loop delay/)).toBeVisible();
    await expect(page.getByText(/~\d+ ms/)).toBeVisible({ timeout: 15_000 });
  });

  test("UC-P3: on-air label and header routed dot when selected", async ({
    page,
    request,
  }) => {
    await connectParticipant(page, "on-air-user");
    const clientId = await participantClientId(page);

    await devScenario(request, `route-to/${encodeURIComponent(clientId)}`);

    await expect(page.getByText("you", { exact: true })).toBeVisible({
      timeout: 10_000,
    });
    await expect(
      page.locator('header span[title="Routed to Teams output"]'),
    ).not.toHaveClass(/invisible/);
  });
});
