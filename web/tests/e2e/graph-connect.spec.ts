import { expect, test, type Page } from "@playwright/test";
import {
  addTask,
  createFeature,
  e2eBaseURL,
  guardBrowserErrors,
  settleGraph,
  submitCreateTask,
  taskNodeId,
} from "./helpers";

test.use({ baseURL: e2eBaseURL });

guardBrowserErrors();

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
