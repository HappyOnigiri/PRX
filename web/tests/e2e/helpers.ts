import { expect, test, type Locator, type Page } from "@playwright/test";

const e2ePort = process.env["PRX_E2E_PORT"];
if (!e2ePort) throw new Error("Playwright did not capture the E2E server port");

export const e2eBaseURL = `http://127.0.0.1:${e2ePort}`;

// ブラウザ側のエラーはテスト本体のアサーションに出ないので、spec ごとに集めて
// 終了時にまとめて突き合わせる。意図した失敗を見込むテストは、返した配列から
// その分を取り除く。
export function guardBrowserErrors() {
  const browserErrors: string[] = [];
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
  return browserErrors;
}

// feature は必ず project に属するので、rail からではなく project のページから
// 作る。これらのテストはデモの最初の project の中に作る。
const demoProjectPath = "/projects/P-1?features=active";

export async function createFeature(
  page: Page,
  title: string,
  description?: string,
) {
  await page.goto(demoProjectPath);
  await page.getByRole("button", { name: "Create feature" }).click();
  const dialog = page.getByRole("form", { name: "Create feature" });
  await dialog.getByLabel("Title").fill(title);
  if (description !== undefined)
    await dialog.getByLabel("Description").fill(description);
  await dialog.getByRole("button", { name: "Create feature" }).click();
  await expect(page.getByRole("heading", { name: title })).toBeVisible();
}

export async function addTask(page: Page, title: string, trigger?: Locator) {
  await (
    trigger ?? page.getByRole("button", { name: "Add task" }).first()
  ).click();
  await submitCreateTask(page, title);
}

export async function submitCreateTask(
  page: Page,
  title: string,
  relation?: string,
) {
  const dialog = page.getByRole("form", { name: "Create task" });
  if (relation !== undefined) await expect(dialog).toContainText(relation);
  await dialog.getByLabel("Title").fill(title);
  await dialog.getByLabel("Scope").fill(`Acceptance boundary for ${title}`);
  await dialog.getByLabel("Assignee").fill("Bob");
  await dialog.getByRole("button", { name: "Add task" }).click();
  await expect(dialog).toBeHidden();
  await expect(
    page.locator(".task-node").filter({ hasText: title }),
  ).toBeVisible();
}

export async function taskNodeId(page: Page, title: string) {
  const id = await page
    .locator(".task-node")
    .filter({ hasText: title })
    .evaluate((element) =>
      element.closest(".react-flow__node")?.getAttribute("data-id"),
    );
  if (!id) throw new Error("task node id missing");
  return id;
}

// レイアウトは非同期に走るので、ノードとビューポートの transform が数サンプル
// 変わらなくなるまで待つ。座標を読む操作はすべてこの後に置く。
export async function settleGraph(page: Page) {
  const viewport = page.locator(".react-flow__viewport");
  const nodes = page.locator(".react-flow__node");
  await expect(page.locator(".graph-stage")).toHaveAttribute(
    "aria-busy",
    "false",
  );
  let previousState: string | null = null;
  let stableSamples = 0;
  await expect
    .poll(
      async () => {
        const viewportStyle = await viewport.getAttribute("style");
        const nodeStyles = await nodes.evaluateAll((elements) =>
          elements.map((element) => element.getAttribute("style")),
        );
        const currentState = JSON.stringify({ nodeStyles, viewportStyle });
        stableSamples = currentState === previousState ? stableSamples + 1 : 0;
        previousState = currentState;
        return stableSamples;
      },
      { intervals: [100], timeout: 3000 },
    )
    .toBeGreaterThanOrEqual(3);
}
