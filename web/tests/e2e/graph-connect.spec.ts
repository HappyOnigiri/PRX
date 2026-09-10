import { expect, test, type Page } from "@playwright/test";

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

test("creates a dependent task by dropping a handle on empty space", async ({
  page,
}) => {
  await createFeature(page, `Handle drop ${crypto.randomUUID()}`);
  await addTask(page, "Drop origin");

  await dropHandleOnEmptySpace(page, "Drop origin", "source");
  await submitCreateTask(page, "Drop follower", "will be blocked by");
  await expectDependencyEdge(page, "Drop origin", "Drop follower", 1);

  await dropHandleOnEmptySpace(page, "Drop origin", "target");
  await submitCreateTask(page, "Drop blocker", "will block");
  await expectDependencyEdge(page, "Drop blocker", "Drop origin", 2);
});

// feature は必ず project に属するので、デモの最初の project の中に作る。
async function createFeature(page: Page, title: string) {
  await page.goto("/projects/P-1?features=active");
  await page.getByRole("button", { name: "Create feature" }).click();
  const dialog = page.getByRole("form", { name: "Create feature" });
  await dialog.getByLabel("Title").fill(title);
  await dialog.getByRole("button", { name: "Create feature" }).click();
  await expect(page.getByRole("heading", { name: title })).toBeVisible();
}

async function addTask(page: Page, title: string) {
  await page.getByRole("button", { name: "Add task" }).first().click();
  await submitCreateTask(page, title);
}

async function submitCreateTask(page: Page, title: string, relation?: string) {
  const dialog = page.getByRole("form", { name: "Create task" });
  if (relation !== undefined) await expect(dialog).toContainText(relation);
  await dialog.getByLabel("Title").fill(title);
  await dialog.getByRole("button", { name: "Add task" }).click();
  await expect(dialog).toBeHidden();
  await expect(
    page.locator(".task-node").filter({ hasText: title }),
  ).toBeVisible();
}

async function dropHandleOnEmptySpace(
  page: Page,
  taskTitle: string,
  endpoint: "source" | "target",
) {
  await settleGraph(page);
  await page.locator(".react-flow__controls-fitview").click();
  await settleGraph(page);
  const handle = page
    .locator(".task-node")
    .filter({ hasText: taskTitle })
    .locator(`.task-handle-${endpoint}`);
  const stage = await page.locator(".graph-stage").boundingBox();
  if (!stage) throw new Error("graph bounds missing");
  await handle.hover();
  await page.mouse.down();
  // ズームのコントロールは左下に出るので、空白は右下の隅を使う。
  await page.mouse.move(
    stage.x + stage.width - 32,
    stage.y + stage.height - 32,
    { steps: 8 },
  );
  await expect(page.locator(".graph-connection-help")).toContainText(
    "Drop in empty space to create a new task",
  );
  await page.mouse.up();
}

async function expectDependencyEdge(
  page: Page,
  blockerTitle: string,
  blockedTitle: string,
  total: number,
) {
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    total,
  );
  const blockerId = await taskNodeId(page, blockerTitle);
  const blockedId = await taskNodeId(page, blockedTitle);
  await expect(
    page.locator(`.react-flow__edge[data-id="${blockerId}-${blockedId}"]`),
  ).toBeAttached();
}

async function taskNodeId(page: Page, title: string) {
  const id = await page
    .locator(".task-node")
    .filter({ hasText: title })
    .evaluate((element) =>
      element.closest(".react-flow__node")?.getAttribute("data-id"),
    );
  if (!id) throw new Error("task node id missing");
  return id;
}

async function settleGraph(page: Page) {
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
