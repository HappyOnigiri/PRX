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

test("keeps the bilingual demo reset warning visible", async ({ page }) => {
  await page.goto("/");
  const banner = page.getByRole("status");
  await expect(banner).toContainText("DEMO");
  await expect(banner).toContainText("Changes reset on restart");
  await expect(banner).toContainText("変更は再起動時にリセットされます");

  await page.getByLabel("Display theme").selectOption("dark");
  await expect(banner).toBeVisible();
  await page.getByLabel("Display language").selectOption("ja");
  await expect(banner).toBeVisible();

  await page.setViewportSize({ width: 320, height: 720 });
  await page.evaluate(() => {
    document.body.style.zoom = "2";
  });
  await expect(banner.locator(".demo-banner-compact")).toBeVisible();
  await expect(banner).toContainText("Reset on restart");
  await expect(banner).toContainText("再起動でリセット");
});

test("switches the display language and restores it from Local Storage", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByLabel("Display language").selectOption("ja");
  await expect(
    page.getByRole("heading", { name: /いま動かせるタスク/ }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "ja");
  await page.reload();
  await expect(page.getByLabel("表示言語")).toHaveValue("ja");
  await expect(
    page.getByRole("heading", { name: /いま動かせるタスク/ }),
  ).toBeVisible();
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
  await expect(page.getByLabel("Display theme")).toHaveValue("system");
  await expect(root).not.toHaveAttribute("data-theme");
  await expect.poll(background).toBe("rgb(245, 246, 248)");

  await page.emulateMedia({ colorScheme: "dark" });
  await expect.poll(background).toBe("rgb(25, 27, 31)");

  await page.getByLabel("Display theme").selectOption("light");
  await expect(root).toHaveAttribute("data-theme", "light");
  await expect.poll(background).toBe("rgb(245, 246, 248)");
  await page.reload();
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

async function disconnectTasks(
  page: Page,
  blockerTitle: string,
  blockedTitle: string,
) {
  await settleGraph(page);
  const blocker = page.locator(".task-node").filter({ hasText: blockerTitle });
  const blocked = page.locator(".task-node").filter({ hasText: blockedTitle });
  const blockerId = await blocker.evaluate((element) =>
    element.closest(".react-flow__node")?.getAttribute("data-id"),
  );
  const blockedId = await blocked.evaluate((element) =>
    element.closest(".react-flow__node")?.getAttribute("data-id"),
  );
  if (!blockerId || !blockedId) throw new Error("task node id missing");

  const edge = page.locator(
    `.react-flow__edge[data-id="${blockerId}-${blockedId}"]`,
  );
  const sourceEndpoint = edge.locator(".react-flow__edgeupdater-source");
  const endpointBox = await sourceEndpoint.boundingBox();
  const stageBox = await page.locator(".graph-stage").boundingBox();
  if (!endpointBox || !stageBox) throw new Error("graph bounds missing");

  await page.mouse.move(
    endpointBox.x + endpointBox.width / 2,
    endpointBox.y + endpointBox.height / 2,
  );
  await expect(sourceEndpoint).toHaveCSS("opacity", "1");
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
    .selectOption({ label: "Markdown path" });
  await reference.getByPlaceholder("Design notes").fill("Delivery plan");
  await reference
    .getByPlaceholder("https://… or docs/plan.md")
    .fill("README.md");
  await reference.getByRole("button", { name: "Add reference" }).click();
  await expect(reference.locator(".document-chip")).toHaveCount(1);
  await reference.locator("select[name=kind]").selectOption({ label: "URL" });
  await reference.getByPlaceholder("Design notes").fill("Release runbook");
  await reference
    .getByPlaceholder("https://… or docs/plan.md")
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
  await disconnectTasks(page, "E2E API", "E2E worker");
  await expect(page.locator(".react-flow__edge.dependency-edge")).toHaveCount(
    1,
  );
  await openTask(page, "E2E worker");
  await expect(inspector.locator(".dependency-chip")).toHaveCount(0);
  await page.getByRole("button", { name: "Close inspector" }).click();
  await openTask(page, "E2E UI");
  page.once("dialog", (dialog) => dialog.accept());
  await inspector
    .getByRole("button", { name: "Delete task and references" })
    .click();
  await expect(
    page.locator(".task-node").filter({ hasText: "E2E UI" }),
  ).toHaveCount(0);
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
  await expect(
    page.getByRole("link", { name: /Archived features/ }),
  ).toBeVisible();
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 320);
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
  const secondaryActions = page.locator(
    ".workspace-actions .icon-button-secondary",
  );
  await expect(secondaryActions).toHaveCount(2);
  for (const action of await secondaryActions.all()) {
    await expect(action).toBeHidden();
  }
  await expect(
    page.locator(".workspace-actions .icon-button-danger"),
  ).toHaveCount(0);
  const addTaskBounds = await addTaskButton.boundingBox();
  expect(addTaskBounds).not.toBeNull();
  if (addTaskBounds) {
    expect(addTaskBounds.x + addTaskBounds.width).toBeLessThanOrEqual(320);
  }
});
