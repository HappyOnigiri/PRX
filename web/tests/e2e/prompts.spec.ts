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

// プロンプトテンプレートは共有のデモ設定にあるため、書き込むのはこの spec だけ
// にし、最後に組み込みテンプレートへ戻す。serial モードが効くのはこのファイル内
// だけなので、テンプレートに依存するテストはここに置く。
test.describe.configure({ mode: "serial" });

test("copies a task prompt built from the configured template", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const token = `e2e-prompt-${crypto.randomUUID()}`;
  const title = `E2E prompt ${token}`;

  // feature は project に属するので、rail からではなくデモの最初の project
  // から作る。
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

  // プロンプトは feature 画面に並ぶタスクからコピーできるので、エージェントに
  // 渡すためにタスクを開く必要はない。
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
  // 復元すると組み込みのテキストがすぐ表示されるので、空欄ではなく保存で
  // 書き込まれる内容が見える。
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

// バッチプロンプトは複数タスクをまとめて扱うので、この spec は 2 件を設計して
// 選択し、サーバーが生成した 1 つのプロンプトを読むまでの一連の流れをたどる。
test("copies one batch prompt for the tasks selected on a feature", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const title = `E2E batch ${crypto.randomUUID()}`;

  // feature は project に属するので、rail からではなくデモの最初の project
  // から作る。
  await page.goto("/projects/P-1?features=active");
  await page.getByRole("button", { name: "Create feature" }).click();
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

  // タスクは実装計画を持って初めてバッチに載るので、バッチを開く前に参照
  // ダイアログから各タスクを設計しておく。
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

  // コピーされるのは保存済みバッチテンプレートからサーバーが生成した文面で、
  // 選択した全タスクを列挙し、各タスクでエージェントを PRX に戻す。
  const copied = await page.evaluate(() => navigator.clipboard.readText());
  expect(copied).toContain(`- ${taskIds[0]}: ${taskTitles[0]}`);
  expect(copied).toContain(`- ${taskIds[1]}: ${taskTitles[1]}`);
  expect(copied).toContain("prx prompt TASK_ID");
  expect(copied).toContain("SubAgent");
  await batchDialog.getByRole("button", { name: "Close" }).click();
});
