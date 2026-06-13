import { expect, test } from "@playwright/test";
import { connectParticipant, gotoHost } from "./helpers.js";

test("participant join appears on host stream grid", async ({ browser }) => {
  const host = await browser.newPage();
  const participant = await browser.newPage();

  await gotoHost(host);
  const streamsBefore = await host.getByText(/\d+ streams/).textContent();

  await connectParticipant(participant, "room-flow");
  await expect(host.getByText("room-flow")).toBeVisible({ timeout: 15_000 });

  const streamsAfter = await host.getByText(/\d+ streams/).textContent();
  expect(streamsAfter).not.toBe(streamsBefore);

  await host.close();
  await participant.close();
});
