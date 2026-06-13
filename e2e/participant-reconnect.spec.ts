import { expect, test } from "@playwright/test";
import { connectParticipant, devScenario } from "./helpers.js";

test("UC-P5: WS drop shows lost-host banner then auto-rejoins", async ({
  page,
  request,
}) => {
  await connectParticipant(page, "reconnect-user");

  await devScenario(request, "drop-participant-ws");

  await expect(page.getByText("reconnecting…")).toBeVisible({
    timeout: 10_000,
  });
  await expect(page.getByText("Lost host connection")).toBeVisible();
  await expect(page.getByText(/Retrying in/)).toBeVisible();

  await expect(page.getByText("connected · WebRTC active")).toBeVisible({
    timeout: 20_000,
  });
  await expect(page.getByText("Lost host connection")).not.toBeVisible();
});

test("UC-P5: retry now reconnects immediately", async ({ page, request }) => {
  await connectParticipant(page, "retry-user");

  await devScenario(request, "drop-participant-ws");
  await expect(page.getByText("Lost host connection")).toBeVisible({
    timeout: 5_000,
  });

  await page.getByRole("button", { name: "Retry now" }).click();
  await expect(page.getByText("connected · WebRTC active")).toBeVisible({
    timeout: 15_000,
  });
});
