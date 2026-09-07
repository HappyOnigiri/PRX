import { expect, test } from "@playwright/test";

const browserErrors: string[] = [];
const e2ePort = process.env["PRX_E2E_PORT"];
if (!e2ePort) throw new Error("Playwright did not capture the E2E server port");

test.use({
  baseURL: `http://127.0.0.1:${e2ePort}`,
});

test.beforeEach(({ page }) => {
  browserErrors.length = 0;
  page.on("console", (message) => {
    if (message.type() === "error" || message.type() === "warning")
      browserErrors.push(`console ${message.type()}: ${message.text()}`);
  });
  page.on("pageerror", (error) =>
    browserErrors.push(`pageerror: ${error.message}`),
  );
  page.on("requestfailed", (request) =>
    browserErrors.push(
      `requestfailed: ${request.method()} ${request.url()} ${request.failure()?.errorText}`,
    ),
  );
});

test.afterEach(() => {
  expect(browserErrors, browserErrors.join("\n")).toEqual([]);
});

// The prompt templates live in the shared demo configuration, so this spec is
// the only one that writes to it and it restores the built-in templates. Serial
// mode orders this file only, so template-sensitive tests belong here.
test.describe.configure({ mode: "serial" });

test("copies a task prompt built from the configured template", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const token = `e2e-prompt-${crypto.randomUUID()}`;
  const title = `E2E prompt ${token}`;

  // A feature belongs to a project, so it is created from the demo's first
  // project rather than from the rail.
  await page.goto("/projects/P-1?features=active");
  await page.getByRole("button", { name: "Create feature" }).click();
  const featureDialog = page.getByRole("form", { name: "Create feature" });
  await featureDialog.getByLabel("Title").fill(title);
  await featureDialog.getByRole("button", { name: "Create feature" }).click();
  await expect(page.getByRole("heading", { name: title })).toBeVisible();

  await page.getByRole("button", { name: "Add task" }).first().click();
  const taskDialog = page.getByRole("form", { name: "Create task" });
  await taskDialog.getByLabel("Title").fill("E2E prompt task");
  await taskDialog.getByLabel("Scope").fill("Prompt boundary");
  await taskDialog.getByRole("button", { name: "Add task" }).click();
  await expect(
    page.locator(".task-node").filter({ hasText: "E2E prompt task" }),
  ).toBeVisible();

  await page.getByRole("button", { name: "Settings" }).click();
  const settings = page.getByRole("dialog", { name: "Settings" });
  await settings.getByRole("tab", { name: "Prompts" }).click();
  const promptPanel = settings.getByRole("tabpanel", { name: "Prompts" });
  await promptPanel
    .getByLabel("Design prompt")
    .fill(`${token} designs {{task_id}}: {{task_title}}`);
  await promptPanel.getByRole("button", { name: "Save" }).click();
  await expect(promptPanel.getByText("Prompt templates saved.")).toBeVisible();
  await settings.getByRole("button", { name: "Done" }).click();

  // The prompt is copied from the task listed on the feature screen, so the
  // reader never has to open the task to hand it to an agent.
  const node = page
    .locator(".task-node")
    .filter({ hasText: "E2E prompt task" });
  const taskId = await node
    .locator(".copyable-identifier-value")
    .first()
    .innerText();
  await node.getByRole("button", { name: "Copy design prompt" }).click();
  await expect(node.getByText("Design prompt copied.")).toBeVisible();
  const copied = await page.evaluate(() => navigator.clipboard.readText());
  expect(copied).toBe(`${token} designs ${taskId}: E2E prompt task`);

  await page.getByRole("button", { name: "Settings" }).click();
  await settings.getByRole("tab", { name: "Prompts" }).click();
  await promptPanel
    .getByRole("button", { name: "Restore built-in templates" })
    .click();
  // Restoring shows the built-in text right away, so the reader sees what the
  // save is about to write instead of an empty field.
  await expect(promptPanel.getByLabel("Design prompt")).toContainText(
    "Design PRX task {{task_id}}",
  );
  await promptPanel.getByRole("button", { name: "Save" }).click();
  await expect(promptPanel.getByText("Prompt templates saved.")).toBeVisible();
  await expect(promptPanel.getByLabel("Design prompt")).toContainText(
    "Design PRX task {{task_id}}",
  );
  await settings.getByRole("button", { name: "Done" }).click();
});
