import { expect, test, type Page } from "@playwright/test";
import { mkdir } from "node:fs/promises";

const browserErrors: string[] = [];

test.beforeEach(async ({ page }) => {
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

test.afterEach(() =>
  expect(browserErrors, browserErrors.join("\n")).toEqual([]),
);

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
  await page.locator(".task-node").filter({ hasText: title }).click();
  await expect(
    page.getByRole("complementary", { name: "Task inspector" }),
  ).toContainText(title);
}

async function addBlocker(page: Page, blockerTitle: string) {
  const inspector = page.getByRole("complementary", { name: "Task inspector" });
  const section = inspector
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Blocked by" }) });
  await section
    .getByLabel("Blocker task")
    .selectOption({ label: blockerTitle });
  await section.getByRole("button", { name: "Add" }).click();
}

test("creates and edits a feature DAG while preserving state", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "New feature" }).click();
  const featureDialog = page.getByRole("form", { name: "Create feature" });
  await featureDialog.getByLabel("Slug").fill("e2e-rollout");
  await featureDialog.getByLabel("Title").fill("E2E rollout");
  await featureDialog
    .getByLabel("Description")
    .fill("Browser-tested delivery circuit");
  await featureDialog.getByRole("button", { name: "Create feature" }).click();
  await expect(
    page.getByRole("heading", { name: "E2E rollout" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Edit feature" }).click();
  const editFeature = page.getByRole("form", { name: "Edit feature" });
  await editFeature
    .getByLabel("Description")
    .fill("Browser-tested delivery circuit, updated");
  await editFeature.getByRole("button", { name: "Save feature" }).click();
  await expect(
    page.getByText("Browser-tested delivery circuit, updated"),
  ).toBeVisible();

  await addTask(page, "E2E API");
  await addTask(page, "E2E worker");
  await addTask(page, "E2E UI");
  await openTask(page, "E2E worker");
  await addBlocker(page, "E2E API");
  await page.getByRole("button", { name: "Close inspector" }).click();
  await openTask(page, "E2E UI");
  await addBlocker(page, "E2E worker");
  await page.getByRole("button", { name: "Close inspector" }).click();
  await openTask(page, "E2E API");
  await addBlocker(page, "E2E UI");
  await expect(page.getByRole("alert")).toContainText("cycle");
  expect(
    browserErrors.filter((item) => item.includes("400 (Bad Request)")),
  ).toHaveLength(1);
  browserErrors.splice(0, browserErrors.length);

  const inspector = page.getByRole("complementary", { name: "Task inspector" });
  const prSection = inspector
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Pull request" }) });
  await prSection
    .getByPlaceholder("https://github.com/org/repo/pull/42")
    .fill("https://github.com/HappyOnigiri/PRX/pull/42");
  await prSection.getByRole("button", { name: "Attach" }).click();
  await expect(
    prSection.getByRole("link", { name: /HappyOnigiri\/PRX #42/ }),
  ).toBeVisible();
  const reference = inspector
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Reference" }) });
  await reference.locator("select[name=kind]").selectOption("markdown_path");
  await reference.getByPlaceholder("Design notes").fill("Delivery plan");
  await reference
    .getByPlaceholder("https://… or docs/plan.md")
    .fill("docs/delivery.md");
  await reference.getByRole("button", { name: "Add reference" }).click();
  await expect(reference.locator(".document-chip")).toHaveCount(1);
  await reference.locator("select[name=kind]").selectOption("url");
  await reference.getByPlaceholder("Design notes").fill("Release runbook");
  await reference
    .getByPlaceholder("https://… or docs/plan.md")
    .fill("https://example.com/runbook");
  await reference.getByRole("button", { name: "Add reference" }).click();
  await expect(reference.locator(".document-chip")).toHaveCount(2);
  await reference
    .getByRole("button", { name: "Delete Release runbook" })
    .click();
  await expect(reference.locator(".document-chip")).toHaveCount(1);
  await inspector.locator("select[name=status]").selectOption("in_progress");
  await inspector.locator("input[name=assignee]").fill("");
  await inspector.getByRole("button", { name: "Save task" }).click();
  await page.getByRole("button", { name: "Close inspector" }).click();
  await page.reload();
  await openTask(page, "E2E API");
  await expect(inspector.locator("input[name=assignee]")).toHaveValue("");
  await page.getByRole("button", { name: "Close inspector" }).click();
  await page.getByRole("button", { name: "Sync GitHub" }).click();
  await expect(
    page.locator(".task-node").filter({ hasText: "E2E API" }),
  ).toContainText("conflict");

  await page.reload();
  await expect(
    page.locator(".task-node").filter({ hasText: "E2E UI" }),
  ).toBeVisible();
  await openTask(page, "E2E worker");
  await expect(inspector.locator(".dependency-chip")).toContainText("E2E API");
  await inspector.getByRole("button", { name: "Remove dependency" }).click();
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
  const dialog = page.getByRole("form", { name: "Create feature" });
  await dialog.getByLabel("Slug").fill("temporary-feature");
  await dialog.getByLabel("Title").fill("Temporary feature");
  await dialog.getByRole("button", { name: "Create feature" }).click();
  await page.getByRole("button", { name: "Archive feature" }).click();
  await expect(
    page
      .getByRole("navigation", { name: "Features" })
      .getByText("Temporary feature"),
  ).toHaveCount(0);
  page.once("dialog", (confirmation) => confirmation.accept());
  await page.getByRole("button", { name: "Delete feature" }).click();
  await expect(
    page.getByRole("heading", { name: /What can move/ }),
  ).toBeVisible();
  await expect(page.getByText("Temporary feature")).toHaveCount(0);
});

for (const size of [8, 50, 100]) {
  test(`renders and inspects the ${size}-node graph`, async ({ page }) => {
    await page.goto("/");
    await page
      .getByRole("link", {
        name: new RegExp(`Cross-repository launch · ${size} nodes`),
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
    for (let i = 0; i < visibleBoxes.length; i++)
      for (let j = i + 1; j < visibleBoxes.length; j++) {
        const a = visibleBoxes[i],
          b = visibleBoxes[j];
        const overlap =
          Math.min(a.right, b.right) - Math.max(a.left, b.left) > 2 &&
          Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top) > 2;
        expect(overlap, `nodes ${i} and ${j} overlap`).toBe(false);
      }
    await mkdir("../test-results/screenshots", { recursive: true });
    if (size > 8)
      await page.screenshot({
        path: `../test-results/screenshots/graph-${size}-overview.png`,
        fullPage: true,
      });
    const zoomSteps = size === 100 ? 3 : size === 50 ? 2 : 0;
    for (let index = 0; index < zoomSteps; index++)
      await page.locator(".react-flow__controls-zoomin").click();
    await page.screenshot({
      path: `../test-results/screenshots/graph-${size}.png`,
      fullPage: true,
    });
    const clickPoint = await page.evaluate(() => {
      const stage = document
        .querySelector("[data-testid=feature-graph]")!
        .getBoundingClientRect();
      return [...document.querySelectorAll(".task-node")]
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
    });
    expect(clickPoint).toBeTruthy();
    await page.mouse.click(clickPoint!.x, clickPoint!.y);
    await expect(
      page.getByRole("complementary", { name: "Task inspector" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Close inspector" }).click();
  });
}

test("keeps controls usable at a narrow viewport", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 720 });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: /What can move/ }),
  ).toBeVisible();
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 320);
});
