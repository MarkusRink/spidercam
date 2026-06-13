import {
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
} from "@playwright/test";

export const HOST_UI = "http://127.0.0.1:4175";
export const PARTICIPANT_UI = "http://127.0.0.1:4174";
export const MOCK_HOST = "http://127.0.0.1:1235";

export async function gotoHost(page: Page): Promise<void> {
  await page.goto(`${HOST_UI}/`);
  await expect(page.getByText("spidercam/host")).toBeVisible({
    timeout: 15_000,
  });
}

export async function connectParticipant(
  page: Page,
  name = "e2e-user",
): Promise<void> {
  await page.goto(`${PARTICIPANT_UI}/`);
  await expect(
    page.getByRole("textbox", { name: "Display name" }),
  ).toBeVisible();
  await page.getByRole("textbox", { name: "Display name" }).fill(name);
  await page.getByRole("button", { name: "Connect" }).click();
  await expect(page.getByText("connected · WebRTC active")).toBeVisible({
    timeout: 15_000,
  });
}

export async function participantClientId(page: Page): Promise<string> {
  const id = await page.locator("header p.truncate").textContent();
  if (!id?.trim()) {
    throw new Error("participant client id not visible");
  }
  return id.trim();
}

export async function devScenario(
  request: APIRequestContext,
  path: string,
  body?: object,
): Promise<void> {
  const res = await request.post(`${MOCK_HOST}/dev/scenario/${path}`, {
    data: body,
  });
  expect(res.ok()).toBeTruthy();
}

export function streamCard(page: Page, name: string): Locator {
  return page
    .locator('div[style*="--stream-card-w"]')
    .filter({ hasText: name });
}

export function mixerSlider(page: Page, label: string): Locator {
  return page
    .locator("label")
    .filter({ hasText: label })
    .locator('input[type="range"]');
}

export function waitForHostConfigMessage(
  page: Page,
  predicate: (config: Record<string, unknown>) => boolean,
): Promise<void> {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(
      () => reject(new Error("timed out waiting for config WS message")),
      10_000,
    );

    const onWebSocket = (ws: import("@playwright/test").WebSocket) => {
      if (!ws.url().includes("/api/v1/ws")) {
        return;
      }
      ws.on("framesent", (event) => {
        try {
          const msg = JSON.parse(String(event.payload)) as {
            type?: string;
            config?: Record<string, unknown>;
          };
          if (msg.type === "config" && msg.config && predicate(msg.config)) {
            clearTimeout(timeout);
            page.off("websocket", onWebSocket);
            resolve();
          }
        } catch {
          /* ignore non-json frames */
        }
      });
    };

    page.on("websocket", onWebSocket);
  });
}
