import { expect, test, type Locator, type Page } from "@playwright/test";
import { mkdir } from "node:fs/promises";

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

async function openDisplaySettings(page: Page, language: "en" | "ja" = "en") {
  const labels =
    language === "en"
      ? { button: "Settings", dialog: "Settings", tab: "Display" }
      : { button: "設定", dialog: "設定", tab: "表示" };
  await page.getByRole("button", { name: labels.button }).click();
  await expect(page.getByRole("dialog", { name: labels.dialog })).toBeVisible();
  await page.getByRole("tab", { name: labels.tab }).click();
}

test("keeps the bilingual demo reset warning visible", async ({ page }) => {
  await page.goto("/");
  const banner = page.getByRole("status");
  await expect(banner).toContainText("DEMO");
  await expect(banner).toContainText("Changes reset on restart");
  await expect(banner).toContainText("変更は再起動時にリセットされます");

  await openDisplaySettings(page);
  await page.getByLabel("Display theme").selectOption("dark");
  await expect(banner).toBeVisible();
  await page.getByLabel("Display language").selectOption("ja");
  await expect(banner).toBeVisible();
  await page.getByRole("button", { name: "完了" }).click();

  await page.setViewportSize({ width: 320, height: 720 });
  await page.evaluate(() => {
    document.body.style.zoom = "2";
  });
  // toContainText reads textContent, so the hidden wide-viewport wording would
  // satisfy it even when nothing is left for a screen reader to announce.
  const compact = banner.locator(".demo-banner-compact");
  await expect(compact).toBeVisible();
  await expect(banner.locator(".demo-banner-full")).toBeHidden();
  await expect(compact).toHaveText("DEMO · Reset on restart再起動でリセット");
  expect(
    await banner.evaluate(
      (element) =>
        Array.from(element.querySelectorAll("[aria-hidden='true']")).length,
    ),
  ).toBe(0);
});

// The demo banner takes its height from the viewport, so a workspace still
// sized to the full viewport pushes its own bottom edge — the graph canvas and
// its zoom controls — off screen.
test("keeps the demo workspace inside the viewport", async ({ page }) => {
  const overflow = () =>
    page.evaluate(() => {
      const root = document.scrollingElement ?? document.documentElement;
      const workspace = document.querySelector(".workspace");
      return {
        page: root.scrollHeight - window.innerHeight,
        workspace: workspace
          ? workspace.getBoundingClientRect().bottom - window.innerHeight
          : Number.NaN,
      };
    });
  await page.goto("/");
  await page
    .getByRole("link", { name: /Delivery control showcase/ })
    .first()
    .click();
  await expect(page.getByRole("status")).toBeVisible();
  await expect(page.getByTestId("feature-graph")).toBeVisible();
  expect(await overflow()).toEqual({ page: 0, workspace: 0 });

  await page.setViewportSize({ width: 320, height: 720 });
  await expect(page.getByTestId("feature-graph")).toBeVisible();
  expect(await overflow()).toEqual({ page: 0, workspace: 0 });
});

test("shows GitHub sync diagnostics and runs a full refresh", async ({
  page,
}) => {
  await page.goto("/");
  await expect(page.locator(".page-head .dashboard-sync-status")).toBeVisible();
  await expect(page.locator(".rail .dashboard-sync-status")).toHaveCount(0);
  await expect(page.locator(".rail")).not.toContainText("GitHub sync");
  const syncButton = page.getByRole("button", { name: "Sync GitHub now" });
  await expect(syncButton).toBeEnabled();
  const syncResponse = page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      response.url().endsWith("/Sync"),
  );
  await syncButton.click();
  expect((await syncResponse).ok()).toBe(true);
  await expect(syncButton).toBeEnabled();
});

test("switches the display language and restores it from Local Storage", async ({
  page,
}) => {
  await page.goto("/");
  await openDisplaySettings(page);
  await page.getByLabel("Display language").selectOption("ja");
  await expect(
    page.getByRole("heading", { name: /いま動かせるタスク/ }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "ja");
  await page.reload();
  await openDisplaySettings(page, "ja");
  await expect(page.getByLabel("表示言語")).toHaveValue("ja");
  await expect(
    page.getByRole("heading", { name: /いま動かせるタスク/ }),
  ).toBeVisible();
});

test("keeps the Settings dialog size while switching tabs", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Settings" }).click();
  const dialog = page.getByRole("dialog", { name: "Settings" });
  await expect(dialog).toBeVisible();
  const tablistBounds = await page.getByRole("tablist").boundingBox();
  expect(tablistBounds?.height).toBeGreaterThanOrEqual(40);
  const panelBounds = await page
    .getByRole("tabpanel", { name: "Server" })
    .boundingBox();
  const syncFormBounds = await page
    .getByRole("heading", { name: "Automatic GitHub updates" })
    .locator("..")
    .locator("..")
    .locator("form")
    .boundingBox();
  expect(panelBounds).not.toBeNull();
  expect(syncFormBounds).not.toBeNull();
  if (panelBounds && syncFormBounds) {
    const rightInset =
      panelBounds.x +
      panelBounds.width -
      (syncFormBounds.x + syncFormBounds.width);
    expect(rightInset).toBeGreaterThanOrEqual(12);
  }
  const serverBounds = await dialog.boundingBox();
  expect(serverBounds).not.toBeNull();

  const displayTab = page.getByRole("tab", { name: "Display" });
  const displayTabBounds = await displayTab.boundingBox();
  expect(displayTabBounds).not.toBeNull();
  if (!displayTabBounds) return;
  await page.mouse.move(
    displayTabBounds.x + displayTabBounds.width / 2,
    displayTabBounds.y + displayTabBounds.height / 2,
  );
  await page.mouse.down();
  expect(await displayTab.boundingBox()).toEqual(displayTabBounds);
  await page.mouse.up();
  await expect(page.getByLabel("Display language")).toBeVisible();
  expect(await dialog.boundingBox()).toEqual(serverBounds);
});

test("follows the system theme unless the user selects an override", async ({
  page,
}) => {
  const root = page.locator("html");
  const background = () =>
    page
      .locator("body")
      .evaluate((element) => getComputedStyle(element).backgroundColor);

  await page.emulateMedia({ colorScheme: "no-preference" });
  await page.goto("/");
  await openDisplaySettings(page);
  await expect(page.getByLabel("Display theme")).toHaveValue("system");
  await expect(root).not.toHaveAttribute("data-theme");
  await expect.poll(background).toBe("rgb(245, 246, 248)");

  await page.emulateMedia({ colorScheme: "dark" });
  await expect.poll(background).toBe("rgb(25, 27, 31)");

  await page.getByLabel("Display theme").selectOption("light");
  await expect(root).toHaveAttribute("data-theme", "light");
  await expect.poll(background).toBe("rgb(245, 246, 248)");
  await page.reload();
  await openDisplaySettings(page);
  await expect(page.getByLabel("Display theme")).toHaveValue("light");
  await expect.poll(background).toBe("rgb(245, 246, 248)");

  await page.getByLabel("Display theme").selectOption("system");
  await expect(root).not.toHaveAttribute("data-theme");
  await expect.poll(background).toBe("rgb(25, 27, 31)");
  await page.emulateMedia({ colorScheme: "light" });
  await expect.poll(background).toBe("rgb(245, 246, 248)");
});

async function addTask(page: Page, title: string) {
  await page.getByRole("button", { name: "Add task" }).first().click();
  const dialog = page.getByRole("form", { name: "Create task" });
  await dialog.getByLabel("Title").fill(title);
  await dialog.getByLabel("Scope").fill(`Acceptance boundary for ${title}`);
  await dialog.getByLabel("Assignee").fill("Mika");
  await dialog.getByRole("button", { name: "Add task" }).click();
  await expect(
    page.locator(".task-node").filter({ hasText: title }),
  ).toBeVisible();
}

async function openTask(page: Page, title: string) {
  await page
    .locator(".task-node")
    .filter({ hasText: title })
    .getByRole("button", { name: `Edit ${title}` })
    .click();
  await expect(
    page.getByRole("complementary", { name: "Task inspector" }),
  ).toContainText(title);
}

async function handleCenterWithinNode(handle: Locator) {
  return handle.evaluate((element) => {
    const node = element.closest(".task-node");
    if (!node) throw new Error("task node missing");
    const handleBox = element.getBoundingClientRect();
    const nodeBox = node.getBoundingClientRect();
    return {
      x: (handleBox.left + handleBox.width / 2 - nodeBox.left) / nodeBox.width,
      y: (handleBox.top + handleBox.height / 2 - nodeBox.top) / nodeBox.height,
    };
  });
}

async function connectTasks(
  page: Page,
  blockerTitle: string,
  blockedTitle: string,
) {
  await settleGraph(page);
  await page.locator(".react-flow__controls-fitview").click();
  await settleGraph(page);
  const blocker = page.locator(".task-node").filter({ hasText: blockerTitle });
  const blocked = page.locator(".task-node").filter({ hasText: blockedTitle });
  const source = blocker.locator(".react-flow__handle.source");
  const target = blocked.locator(".react-flow__handle.target");
  await expect(source).toBeVisible();
  await expect(target).toBeVisible();
  const initialCenter = await handleCenterWithinNode(source);
  await source.hover();
  const hoveredCenter = await handleCenterWithinNode(source);
  expect(hoveredCenter.x).toBeCloseTo(initialCenter.x, 2);
  expect(hoveredCenter.y).toBeCloseTo(initialCenter.y, 2);
  await page.mouse.down();
  await target.hover();
  await page.mouse.up();
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

async function selectDependencyEdge(
  page: Page,
  blockerTitle: string,
  blockedTitle: string,
) {
  await settleGraph(page);
  const blockerId = await taskNodeId(page, blockerTitle);
  const blockedId = await taskNodeId(page, blockedTitle);
  const edge = page.locator(
    `.react-flow__edge[data-id="${blockerId}-${blockedId}"]`,
  );
  await edge.click({ force: true });
  await expect(edge).toHaveClass(/selected/);
  return edge;
}

async function disconnectTasks(
  page: Page,
  blockerTitle: string,
  blockedTitle: string,
  endpoint: "source" | "target" = "source",
) {
  const edge = await selectDependencyEdge(page, blockerTitle, blockedTitle);
  const endpointHandle = edge.locator(`.react-flow__edgeupdater-${endpoint}`);
  const endpointBox = await endpointHandle.boundingBox();
  const stageBox = await page.locator(".graph-stage").boundingBox();
  if (!endpointBox || !stageBox) throw new Error("graph bounds missing");

  await page.mouse.move(
    endpointBox.x + endpointBox.width / 2,
    endpointBox.y + endpointBox.height / 2,
  );
  await expect(endpointHandle).toHaveCSS("opacity", "1");
  await page.mouse.down();
  await page.mouse.move(stageBox.x + 28, stageBox.y + 28, { steps: 6 });
  await page.mouse.up();
}

async function settleGraph(page: Page) {
  const stage = page.locator(".graph-stage");
  const viewport = page.locator(".react-flow__viewport");
  const nodes = page.locator(".react-flow__node");
  await expect(stage).toHaveAttribute("aria-busy", "false");
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

async function graphZoom(page: Page) {
  return page.locator(".react-flow__viewport").evaluate((viewport) => {
    const transform = new DOMMatrix(getComputedStyle(viewport).transform);
    return transform.a;
  });
}

async function taskGroupBounds(page: Page, titles: string[]) {
  const boxes = await Promise.all(
    titles.map((title) =>
      page
        .locator(".task-node")
        .filter({
          has: page.getByRole("heading", { name: title, exact: true }),
        })
        .boundingBox(),
    ),
  );
  if (boxes.some((box) => !box)) throw new Error("task bounds missing");
  const presentBoxes = boxes.filter((box) => box !== null);
  return {
    left: Math.min(...presentBoxes.map((box) => box.x)),
    right: Math.max(...presentBoxes.map((box) => box.x + box.width)),
    top: Math.min(...presentBoxes.map((box) => box.y)),
    bottom: Math.max(...presentBoxes.map((box) => box.y + box.height)),
  };
}

function boundsGap(
  first: Awaited<ReturnType<typeof taskGroupBounds>>,
  second: Awaited<ReturnType<typeof taskGroupBounds>>,
) {
  const horizontal = Math.max(
    0,
    Math.max(first.left, second.left) - Math.min(first.right, second.right),
  );
  const vertical = Math.max(
    0,
    Math.max(first.top, second.top) - Math.min(first.bottom, second.bottom),
  );
  return Math.hypot(horizontal, vertical);
}

test("creates and edits a feature DAG while preserving state", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  // The server keeps one database for the whole run, so a fixed slug would make
  // every retry fail on the unique constraint instead of absorbing a flake.
  const slug = `e2e-rollout-${crypto.randomUUID()}`;
  const title = `E2E rollout ${slug}`;
  // The demo fixture derives the PR state from the number, and 4k+2 maps to a
  // conflicting pull request. Keeping it unique avoids colliding with the pull
  // request a previous attempt attached to a task that still exists.
  const prNumber = Math.floor(Math.random() * 1_000_000) * 4 + 2;
  await page.goto("/");
  await page.getByRole("button", { name: "New feature" }).click();
  const featureDialog = page.getByRole("form", { name: "Create feature" });
  await featureDialog.getByLabel("Slug").fill(slug);
  await featureDialog.getByLabel("Title").fill(title);
  await featureDialog
    .getByLabel("Description")
    .fill("Browser-tested delivery circuit");
  await featureDialog.getByRole("button", { name: "Create feature" }).click();
  await expect(page.getByRole("heading", { name: title })).toBeVisible();
  await page.getByRole("button", { name: "Edit feature" }).click();
  const editFeature = page.getByRole("form", { name: "Edit feature" });
  await editFeature
    .getByLabel("Description")
    .fill("Browser-tested delivery circuit, updated");
  await editFeature.getByRole("button", { name: "Save feature" }).click();
  await expect(page.locator(".workspace-title")).toHaveAttribute(
    "title",
    "Browser-tested delivery circuit, updated",
  );

  await page.getByRole("button", { name: "References" }).click();
  await page.getByRole("button", { name: "Add reference" }).click();
  const featureReferenceDialog = page.getByRole("dialog", {
    name: "Add feature reference",
  });
  await featureReferenceDialog
    .getByLabel("Type")
    .selectOption({ label: "Local file" });
  await featureReferenceDialog
    .getByLabel("Title (optional)")
    .fill("Feature brief");
  await featureReferenceDialog.getByLabel("URL or file path").fill("README.md");
  await featureReferenceDialog
    .getByRole("button", { name: "Add reference" })
    .click();
  await expect(featureReferenceDialog).toBeHidden();
  await page.getByRole("button", { name: "References" }).click();
  const featureReferences = page.getByRole("region", { name: "References" });
  await expect(featureReferences.locator(".document-chip")).toHaveCount(1);
  await featureReferences
    .getByRole("button", { name: /^Feature brief README\.md$/ })
    .click();
  await expect(featureReferences).toBeHidden();
  const featurePreview = page.getByRole("dialog", { name: "Feature brief" });
  await expect(featurePreview.locator("article")).toContainText("PRX");
  await featurePreview
    .getByRole("button", { name: "Close Markdown preview" })
    .click();

  await addTask(page, "E2E API");
  await addTask(page, "E2E worker");
  await addTask(page, "E2E UI");
  await connectTasks(page, "E2E API", "E2E worker");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    1,
  );
  await connectTasks(page, "E2E worker", "E2E UI");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    2,
  );
  await settleGraph(page);
  await page.locator(".react-flow__controls-fitview").click();
  await settleGraph(page);
  await connectTasks(page, "E2E UI", "E2E API");
  await expect(page.getByRole("alert")).toContainText("cycle");
  expect(
    browserErrors.filter((item) => item.includes("400 (Bad Request)")),
  ).toHaveLength(1);
  browserErrors.splice(0, browserErrors.length);

  await openTask(page, "E2E API");
  const inspector = page.getByRole("complementary", { name: "Task inspector" });
  const prSection = inspector
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Pull request" }) });
  await prSection
    .getByPlaceholder("https://github.com/org/repo/pull/42")
    .fill("https://example.com/not-a-pull-request");
  await prSection.getByRole("button", { name: "Attach" }).click();
  await expect(prSection.getByRole("alert")).toContainText("github.com");
  expect(
    browserErrors.filter((item) => item.includes("400 (Bad Request)")),
  ).toHaveLength(1);
  browserErrors.splice(0, browserErrors.length);
  await prSection
    .getByPlaceholder("https://github.com/org/repo/pull/42")
    .fill(`https://github.com/HappyOnigiri/PRX/pull/${prNumber}`);
  await prSection.getByRole("button", { name: "Attach" }).click();
  await expect(
    prSection.getByRole("link", {
      name: new RegExp(`HappyOnigiri/PRX #${prNumber}`),
    }),
  ).toBeVisible();
  const reference = inspector
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Reference" }) });
  await reference
    .locator("select[name=kind]")
    .selectOption({ label: "Local file" });
  await reference.getByPlaceholder("Design notes").fill("Delivery plan");
  await reference.getByPlaceholder("docs/plan.md").fill("README.md");
  await reference.getByRole("button", { name: "Add reference" }).click();
  await expect(reference.locator(".document-chip")).toHaveCount(1);
  await reference.locator("select[name=kind]").selectOption({ label: "URL" });
  await reference.getByPlaceholder("Design notes").fill("Release runbook");
  await reference
    .getByPlaceholder("https://example.com/document")
    .fill("https://example.com/runbook");
  await reference.getByRole("button", { name: "Add reference" }).click();
  await expect(reference.locator(".document-chip")).toHaveCount(2);
  await inspector
    .locator("select[name=status]")
    .selectOption({ label: "In progress" });
  await inspector.locator("input[name=assignee]").fill("");
  await inspector.getByRole("button", { name: "Save task" }).click();
  await page.getByRole("button", { name: "Close inspector" }).click();
  const apiCard = page.locator(".task-node").filter({ hasText: "E2E API" });
  await expect(
    apiCard.getByRole("link", {
      name: new RegExp(`HappyOnigiri/PRX #${prNumber}`),
    }),
  ).toHaveAttribute("target", "_blank");
  await expect(
    apiCard.getByRole("link", { name: /Release runbook/ }),
  ).toHaveAttribute("target", "_blank");
  await apiCard.getByRole("heading", { name: "E2E API" }).click();
  await expect(
    page.getByRole("complementary", { name: "Task inspector" }),
  ).toHaveCount(0);
  await apiCard.getByRole("button", { name: /Delivery plan/ }).click();
  const preview = page.getByRole("dialog", { name: "Delivery plan" });
  await expect(
    preview.getByRole("heading", { name: "PRX", level: 1 }),
  ).toBeVisible();
  await preview.getByRole("button", { name: "Copy full text" }).click();
  await expect(preview).toContainText("Full text copied.");
  await preview.getByRole("button", { name: "Copy file path" }).click();
  await expect(preview).toContainText("File path copied.");
  await preview.getByRole("button", { name: "Close Markdown preview" }).click();
  await page.reload();
  await openTask(page, "E2E API");
  await expect(inspector.locator("input[name=assignee]")).toHaveValue("");
  await page.getByRole("button", { name: "Close inspector" }).click();
  await page.getByRole("button", { name: "Sync GitHub" }).click();
  await expect(
    page.locator(".task-node").filter({ hasText: "E2E API" }),
  ).toHaveClass(/state-in-progress/);
  await openTask(page, "E2E API");
  await expect(inspector.locator(".linked-pr")).toContainText("conflict");
  await page.getByRole("button", { name: "Close inspector" }).click();

  await page.reload();
  await expect(
    page.locator(".task-node").filter({ hasText: "E2E UI" }),
  ).toBeVisible();
  await selectDependencyEdge(page, "E2E worker", "E2E UI");
  await page.locator(".dependency-edge-remove").click();
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    1,
  );
  await connectTasks(page, "E2E worker", "E2E UI");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    2,
  );
  // Keyboard users reach the toolbar and the Delete key only when React Flow's
  // own selection change is applied back to the controlled edges.
  await settleGraph(page);
  const keyboardEdge = page.locator(
    `.react-flow__edge[data-id="${await taskNodeId(page, "E2E worker")}-${await taskNodeId(page, "E2E UI")}"]`,
  );
  await keyboardEdge.focus();
  await page.keyboard.press("Enter");
  await expect(keyboardEdge).toHaveClass(/selected/);
  await page.keyboard.press("Delete");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    1,
  );
  await connectTasks(page, "E2E worker", "E2E UI");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    2,
  );
  await selectDependencyEdge(page, "E2E worker", "E2E UI");
  await page.keyboard.press("Delete");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    1,
  );
  await connectTasks(page, "E2E worker", "E2E UI");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    2,
  );
  await disconnectTasks(page, "E2E API", "E2E worker", "target");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    1,
  );
  await openTask(page, "E2E UI");
  page.once("dialog", (dialog) => dialog.accept());
  await inspector
    .getByRole("button", { name: "Delete task and references" })
    .click();
  await expect(
    page.locator(".task-node").filter({ hasText: "E2E UI" }),
  ).toHaveCount(0);
});

test("visually separates disconnected dependency chains", async ({ page }) => {
  const slug = `disconnected-chains-${crypto.randomUUID()}`;
  await page.goto("/");
  await page.getByRole("button", { name: "New feature" }).click();
  const featureDialog = page.getByRole("form", { name: "Create feature" });
  await featureDialog.getByLabel("Slug").fill(slug);
  await featureDialog.getByLabel("Title").fill(`Disconnected chains ${slug}`);
  await featureDialog.getByRole("button", { name: "Create feature" }).click();

  for (const title of ["Chain A1", "Chain A2", "Chain B1", "Chain B2"])
    await addTask(page, title);
  await page.locator(".react-flow__controls-fitview").click();
  await settleGraph(page);
  await connectTasks(page, "Chain A1", "Chain A2");
  await connectTasks(page, "Chain B1", "Chain B2");
  await settleGraph(page);

  const firstChain = await taskGroupBounds(page, ["Chain A1", "Chain A2"]);
  const secondChain = await taskGroupBounds(page, ["Chain B1", "Chain B2"]);
  const componentGap =
    boundsGap(firstChain, secondChain) / (await graphZoom(page));
  expect(componentGap).toBeGreaterThanOrEqual(118);
});

test("archives and safely deletes a feature", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "New feature" }).click();
  const slug = `temporary-feature-${crypto.randomUUID()}`;
  const title = `Temporary feature ${slug}`;
  const dialog = page.getByRole("form", { name: "Create feature" });
  await dialog.getByLabel("Slug").fill(slug);
  await dialog.getByLabel("Title").fill(title);
  await dialog.getByRole("button", { name: "Create feature" }).click();
  await addTask(page, "Archived E2E task");
  await page.getByRole("button", { name: "Edit feature" }).click();
  await page.getByRole("button", { name: "Archive feature" }).click();
  const archiveConfirmation = page.getByRole("dialog", {
    name: `Archive ${title}?`,
  });
  await archiveConfirmation
    .getByRole("button", { name: "Archive feature" })
    .click();
  await expect(page.getByText("Archived · read-only")).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Features" }).getByText(title),
  ).toHaveCount(0);
  await page.getByRole("link", { name: /Archived features/ }).click();
  await expect(
    page.getByRole("heading", { name: "Archived features", exact: true }),
  ).toBeVisible();
  await page.getByText(title).click();
  await expect(page.getByRole("button", { name: "Sync GitHub" })).toHaveCount(
    0,
  );
  await page
    .getByRole("button", { name: "View Archived E2E task details" })
    .click();
  const inspector = page.getByRole("complementary", { name: "Task inspector" });
  await expect(inspector).toContainText("Archived task · read-only");
  await expect(inspector.getByRole("textbox")).toHaveCount(0);
  await inspector.getByRole("button", { name: "Close inspector" }).click();

  await page.getByRole("button", { name: "Manage feature" }).click();
  await page.getByRole("button", { name: "Restore feature" }).click();
  await expect(page.getByRole("button", { name: "Sync GitHub" })).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Features" }).getByText(title),
  ).toHaveCount(1);

  await page.getByRole("button", { name: "Edit feature" }).click();
  await page.getByRole("button", { name: "Archive feature" }).click();
  await page
    .getByRole("dialog", { name: `Archive ${title}?` })
    .getByRole("button", { name: "Archive feature" })
    .click();
  await page.getByRole("button", { name: "Manage feature" }).click();
  await page.getByRole("button", { name: "Delete feature" }).click();
  const deleteConfirmation = page.getByRole("dialog", {
    name: `Delete ${title}?`,
  });
  await deleteConfirmation.getByRole("button", { name: "Cancel" }).click();
  await expect(
    page.getByRole("heading", { name: "Feature details" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Delete feature" }).click();
  await page
    .getByRole("dialog", { name: `Delete ${title}?` })
    .getByRole("button", { name: "Delete permanently" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Archived features", exact: true }),
  ).toBeVisible();
  await expect(page.getByText(title)).toHaveCount(0);
});

for (const { title, size } of [
  { title: "Delivery control showcase", size: 13 },
  { title: "Completed 100-task program", size: 100 },
]) {
  test(`renders and inspects the ${size}-node graph`, async ({ page }) => {
    await page.goto("/");
    await page
      .getByRole("link", {
        name: new RegExp(title),
      })
      .first()
      .click();
    const nodes = page.locator(".task-node");
    await expect(nodes).toHaveCount(size, { timeout: 25_000 });
    await expect(page.getByTestId("feature-graph")).toBeVisible();
    await page.locator(".react-flow__controls-fitview").click();
    await page.waitForTimeout(400);
    const visibleBoxes = await nodes.evaluateAll((items) =>
      items.slice(0, 30).map((item) => {
        const rect = item.getBoundingClientRect();
        return {
          left: rect.left,
          right: rect.right,
          top: rect.top,
          bottom: rect.bottom,
        };
      }),
    );
    for (const [i, a] of visibleBoxes.entries())
      for (const [offset, b] of visibleBoxes.slice(i + 1).entries()) {
        const j = i + offset + 1;
        const overlap =
          Math.min(a.right, b.right) - Math.max(a.left, b.left) > 2 &&
          Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top) > 2;
        expect(overlap, `nodes ${i} and ${j} overlap`).toBe(false);
      }
    await mkdir("../test-results/screenshots", { recursive: true });
    if (size > 13)
      await page.screenshot({
        path: `../test-results/screenshots/graph-${size}-overview.png`,
        fullPage: true,
      });
    const zoomSteps = size === 100 ? 3 : 0;
    for (let index = 0; index < zoomSteps; index++)
      await page.locator(".react-flow__controls-zoomin").click();
    await page.screenshot({
      path: `../test-results/screenshots/graph-${size}.png`,
      fullPage: true,
    });
    const clickPoint = await page.evaluate(() => {
      const stageElement = document.querySelector(
        "[data-testid=feature-graph]",
      );
      if (!stageElement) throw new Error("The feature graph stage is missing.");
      const stage = stageElement.getBoundingClientRect();
      const point = [...document.querySelectorAll(".task-node .node-edit")]
        .map((item) => item.getBoundingClientRect())
        .map((rect) => ({
          x: rect.left + rect.width / 2,
          y: rect.top + rect.height / 2,
        }))
        .find(
          (point) =>
            point.x > stage.left + 20 &&
            point.x < stage.right - 20 &&
            point.y > stage.top + 20 &&
            point.y < stage.bottom - 20,
        );
      if (!point) throw new Error("No task edit button is visible.");
      return point;
    });
    await page.mouse.click(clickPoint.x, clickPoint.y);
    await expect(
      page.getByRole("complementary", { name: "Task inspector" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Close inspector" }).click();
  });
}

test("keeps the user's graph zoom across features and reloads", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("link", { name: /Delivery control showcase/ })
    .first()
    .click();
  await expect(page.locator(".task-node")).toHaveCount(13, {
    timeout: 25_000,
  });
  await page.locator(".react-flow__controls-zoomout").click();
  await page.locator(".react-flow__controls-zoomout").click();
  await expect.poll(() => graphZoom(page)).toBeLessThan(1);
  const savedZoom = await graphZoom(page);

  await page
    .getByRole("navigation", { name: "Features" })
    .getByText("Completed 100-task program")
    .click();
  await expect(page.locator(".task-node")).toHaveCount(100, {
    timeout: 25_000,
  });
  await expect.poll(() => graphZoom(page)).toBeCloseTo(savedZoom, 5);

  await page.reload();
  await expect(page.locator(".task-node")).toHaveCount(100, {
    timeout: 25_000,
  });
  await expect.poll(() => graphZoom(page)).toBeCloseTo(savedZoom, 5);
});

test("keeps controls usable at a narrow viewport", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 720 });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: /What can move/ }),
  ).toBeVisible();
  const syncStatus = page.locator(".page-head .dashboard-sync-status");
  await expect(syncStatus).toBeVisible();
  await expect(page.locator(".rail .dashboard-sync-status")).toHaveCount(0);
  const dashboardSyncButton = page.getByRole("button", {
    name: "Sync GitHub now",
  });
  await expect(dashboardSyncButton).toBeVisible();
  const dashboardSyncBounds = await dashboardSyncButton.boundingBox();
  expect(dashboardSyncBounds).not.toBeNull();
  if (dashboardSyncBounds)
    expect(
      dashboardSyncBounds.x + dashboardSyncBounds.width,
    ).toBeLessThanOrEqual(320);
  const syncValueStyle = await syncStatus
    .locator(":scope > :last-child")
    .evaluate((element) => {
      const style = getComputedStyle(element);
      return {
        overflow: style.overflow,
        textOverflow: style.textOverflow,
        whiteSpace: style.whiteSpace,
      };
    });
  expect(syncValueStyle.overflow).not.toBe("hidden");
  expect(syncValueStyle.textOverflow).not.toBe("ellipsis");
  expect(syncValueStyle.whiteSpace).not.toBe("nowrap");
  await expect(
    page.getByRole("link", { name: /Archived features/ }),
  ).toBeVisible();
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 320);
  await openDisplaySettings(page);
  await page.getByLabel("Display language").selectOption("ja");
  const settingsDialog = page.getByRole("dialog", { name: "設定" });
  await expect(settingsDialog).toBeVisible();
  await expect(page.getByRole("tab", { name: "表示" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  for (const control of await settingsDialog.getByRole("combobox").all()) {
    const bounds = await control.boundingBox();
    expect(bounds).not.toBeNull();
    if (bounds) {
      expect(bounds.x).toBeGreaterThanOrEqual(0);
      expect(bounds.x + bounds.width).toBeLessThanOrEqual(320);
    }
  }
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 320);
  await page.getByLabel("表示言語").selectOption("en");
  await page
    .getByRole("dialog", { name: "Settings" })
    .getByRole("button", { name: "Done" })
    .click();
  await page
    .getByRole("link", { name: /Delivery control showcase/ })
    .first()
    .click();
  const addTaskButton = page.getByRole("button", { name: "Add task" });
  await expect(addTaskButton).toBeVisible();
  await expect(addTaskButton.locator("svg")).toHaveAttribute(
    "aria-hidden",
    "true",
  );
  await expect(page.getByRole("button", { name: "Sync GitHub" })).toBeHidden();
  await expect(page.getByRole("button", { name: "Edit feature" })).toBeHidden();
  const referencesButton = page.getByRole("button", { name: "References" });
  await expect(referencesButton).toBeVisible();
  await referencesButton.click();
  await expect(page.getByRole("region", { name: "References" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Add reference" }),
  ).toBeVisible();
  await expect(
    page.locator(".workspace-actions > .icon-button-danger"),
  ).toHaveCount(0);
  const addTaskBounds = await addTaskButton.boundingBox();
  expect(addTaskBounds).not.toBeNull();
  if (addTaskBounds) {
    expect(addTaskBounds.x + addTaskBounds.width).toBeLessThanOrEqual(320);
  }
});
