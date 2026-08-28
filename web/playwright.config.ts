import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  outputDir: "../test-results/playwright",
  timeout: 45_000,
  expect: { timeout: 8_000 },
  fullyParallel: false,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI
    ? [
        ["github"],
        ["html", { outputFolder: "../playwright-report", open: "never" }],
      ]
    : "list",
  use: {
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    ...devices["Desktop Chrome"],
  },
  webServer: {
    command: "../scripts/run-e2e-server.sh",
    wait: {
      stderr: new RegExp(
        "PRX listening on http://127\\.0\\.0\\.1:(?<PRX_E2E_PORT>\\d+)",
      ),
    },
    timeout: 120_000,
  },
});
