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

// フィルタは共有のデモグラフを読むだけなので、この spec は他と同じ
// サーバーに対して並列に実行する。
test("hides completed tasks and leaves their dependency visible", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("link", { name: /Delivery control showcase/ })
    .first()
    .click();
  const nodes = page.locator(".task-node");
  await expect(nodes).toHaveCount(13, { timeout: 25_000 });
  const finished = nodes.filter({ hasText: "Verify storage boundary" });
  await expect(finished).toHaveCount(1);
  await expect(page.locator(".node-hidden-dependency")).toHaveCount(0);

  const toggle = page.getByRole("switch", { name: "Hide completed" });
  await expect(toggle).not.toBeChecked();
  await toggle.check();

  await expect(finished).toHaveCount(0);
  await expect(nodes).toHaveCount(10, { timeout: 25_000 });
  // draft の pull request を塞いでいた完了タスクが消えるので、draft 側には
  // 何を待っているかを示すスタブが付く。
  const stub = nodes
    .filter({ hasText: "Draft WebUI shell" })
    .locator(".node-hidden-dependency-in");
  await expect(stub).toHaveAttribute(
    "aria-label",
    "Hidden completed blockers: Verify storage boundary",
  );

  // スイッチはブラウザローカルの状態なので、リロードしても絞り込みが残る。
  await page.reload();
  await expect(toggle).toBeChecked();
  await expect(nodes).toHaveCount(10, { timeout: 25_000 });

  await toggle.uncheck();
  await expect(nodes).toHaveCount(13, { timeout: 25_000 });
  await expect(page.locator(".node-hidden-dependency")).toHaveCount(0);
});
