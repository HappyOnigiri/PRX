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
// the only one that writes to it, and it puts the built-in templates back before
// it finishes. Serial mode orders the tests inside this file only: every other
// spec still runs against the same server in parallel, so a test that reads or
// writes a template belongs in this file rather than in a new one.
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

// The batch prompt covers several tasks at once, so this spec drives the whole
// path a reader takes: designing two tasks, selecting them, and reading the one
// prompt the server rendered from them.
test("copies one batch prompt for the tasks selected on a feature", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const title = `E2E batch ${crypto.randomUUID()}`;

  await page.goto("/");
  await page.getByRole("button", { name: "New feature" }).click();
  const featureDialog = page.getByRole("form", { name: "Create feature" });
  await featureDialog.getByLabel("Title").fill(title);
  await featureDialog.getByRole("button", { name: "Create feature" }).click();
  await expect(page.getByRole("heading", { name: title })).toBeVisible();

  const taskTitles = ["E2E batch first", "E2E batch second"];
  for (const taskTitle of taskTitles) {
    await page.getByRole("button", { name: "Add task" }).first().click();
    const taskDialog = page.getByRole("form", { name: "Create task" });
    await taskDialog.getByLabel("Title").fill(taskTitle);
    await taskDialog.getByRole("button", { name: "Add task" }).click();
    await expect(
      page.locator(".task-node").filter({ hasText: taskTitle }),
    ).toBeVisible();
  }

  // A task reaches the batch only once it has a plan, so each one is designed
  // through the reference dialog before the batch is opened.
  const taskIds: string[] = [];
  for (const taskTitle of taskTitles) {
    const node = page.locator(".task-node").filter({ hasText: taskTitle });
    taskIds.push(
      await node.locator(".copyable-identifier-value").first().innerText(),
    );
    await node
      .getByRole("button", { name: `Add reference to ${taskTitle}` })
      .click();
    const referenceDialog = page.getByRole("dialog", {
      name: "Add task reference",
    });
    await referenceDialog.getByRole("tab", { name: "Markdown" }).click();
    await referenceDialog
      .getByLabel("Reference title (optional)")
      .fill(`${taskTitle} plan`);
    await referenceDialog.getByLabel("Markdown content").fill("# Plan\n");
    await referenceDialog
      .getByLabel("Use as this task's implementation plan")
      .check();
    await referenceDialog
      .getByRole("button", { name: "Add reference" })
      .click();
    await expect(referenceDialog).toBeHidden();
  }

  await page.getByRole("button", { name: "Copy batch prompt" }).click();
  const batchDialog = page.getByRole("dialog", {
    name: "Copy batch implementation prompt",
  });
  await expect(batchDialog.getByText("0 of 2 selected")).toBeVisible();
  await batchDialog.getByRole("button", { name: "Select all" }).click();
  await batchDialog.getByRole("button", { name: "Copy prompt" }).click();
  await expect(
    batchDialog.getByText("Copied a prompt for 2 tasks."),
  ).toBeVisible();

  // The copied text is what the server rendered from the stored batch template:
  // it names every selected task and sends the agent back to PRX for each one.
  const copied = await page.evaluate(() => navigator.clipboard.readText());
  expect(copied).toContain(`- ${taskIds[0]}: ${taskTitles[0]}`);
  expect(copied).toContain(`- ${taskIds[1]}: ${taskTitles[1]}`);
  expect(copied).toContain("prx prompt TASK_ID");
  expect(copied).toContain("SubAgent");
  await batchDialog.getByRole("button", { name: "Close" }).click();
});
