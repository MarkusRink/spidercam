import { defineConfig, devices } from "@playwright/test";

const reuseExistingServer = !process.env.CI;

export default defineConfig({
  testDir: ".",
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? "github" : "list",
  use: {
    trace: "on-first-retry",
    launchOptions: {
      args: [
        "--use-fake-ui-for-media-stream",
        "--use-fake-device-for-media-stream",
      ],
    },
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
      },
    },
  ],
  webServer: [
    {
      command:
        "npm run build -w @spidercam/mock-server && npm run start -w @spidercam/mock-server",
      url: "http://127.0.0.1:1235/api/health",
      reuseExistingServer,
      timeout: 120_000,
    },
    {
      command: "npm run preview -w @spidercam/host",
      url: "http://127.0.0.1:4175",
      reuseExistingServer,
      timeout: 60_000,
    },
    {
      command: "npm run preview -w @spidercam/participant",
      url: "http://127.0.0.1:4174",
      reuseExistingServer,
      timeout: 60_000,
    },
  ],
});
