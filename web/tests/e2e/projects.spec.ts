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

// The demo data is shared by every worker, so each test builds its own project
// and feature and never archives one the other tests read.
async function createProject(page: Page, title: string) {
  await page.goto("/projects");
  await page.getByRole("button", { name: "New project" }).click();
  const dialog = page.getByRole("form", { name: "Create project" });
  await dialog.getByLabel("Title").fill(title);
  await dialog.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: title })).toBeVisible();
}

test("presents the demo projects and their shared references", async ({
  page,
}) => {
  await page.goto("/projects");
  const list = page.getByRole("region", { name: "Project list" });
  await expect(list).toContainText("Delivery platform");
  await expect(list).not.toContainText("Sunset initiative");

  await page.getByRole("tab", { name: "Archived" }).click();
  await expect(page).toHaveURL(/archived=true/);
  await expect(
    page.getByRole("region", { name: "Project list" }),
  ).toContainText("Sunset initiative");

  // Reloading has to reproduce the archived view, which is why the tab lives
  // in the URL rather than in browser-local state.
  await page.reload();
  await expect(
    page.getByRole("region", { name: "Project list" }),
  ).toContainText("Sunset initiative");
  await page.getByRole("tab", { name: "Active" }).click();

  await page
    .getByRole("region", { name: "Project list" })
    .getByText("Delivery platform")
    .click();
  await expect(page.getByRole("tabpanel")).toContainText(
    "Delivery control showcase",
  );
  await page.getByRole("button", { name: "References" }).click();
  await expect(page.getByRole("region", { name: "References" })).toContainText(
    "Platform charter",
  );
});

// A sidebar project row reuses the feature row's class, so its own single-column
// track list has to outrank the feature one. Losing that override drops the
// title into the 8px status-dot column, where the row still reads as present
// but shows one clipped character.
test("gives the sidebar project title the whole row", async ({ page }) => {
  await page.goto("/projects");
  const title = page
    .locator(".project-link", { hasText: "Delivery platform" })
    .locator("span");
  await expect(title).toBeVisible();
  const clipped = await title.evaluate(
    (element) => element.scrollWidth - element.clientWidth,
  );
  expect(clipped).toBeLessThanOrEqual(0);
});

test("archives a project and makes its feature read-only", async ({ page }) => {
  const title = `E2E project ${crypto.randomUUID()}`;
  await createProject(page, title);

  // The project page is where a feature is created, so the membership follows
  // from the page instead of a field in the dialog.
  const featureTitle = `E2E member ${crypto.randomUUID()}`;
  await page.getByRole("button", { name: "Create feature" }).click();
  const featureDialog = page.getByRole("form", { name: "Create feature" });
  await expect(featureDialog.getByLabel("Project")).toHaveCount(0);
  await featureDialog.getByLabel("Title").fill(featureTitle);
  await featureDialog.getByRole("button", { name: "Create feature" }).click();
  await expect(page.getByRole("heading", { name: featureTitle })).toBeVisible();
  // The feature header names the project it now belongs to.
  await expect(page.locator(".workspace-project-link")).toHaveText(title);

  // The sidebar links to the project too, so follow the one in the header.
  await page.locator(".workspace-project-link").click();
  await expect(page.getByRole("tabpanel")).toContainText(featureTitle);
  await page.getByRole("button", { name: "Edit project" }).click();
  await page.getByRole("button", { name: "Archive project" }).click();
  await page
    .getByRole("dialog", { name: `Archive ${title}?` })
    .getByRole("button", { name: "Archive project" })
    .click();
  await expect(page.getByText("Archived · read-only")).toBeVisible();

  // The feature itself is not archived, so the notice has to say the archive
  // came from the project and link back to it instead of offering a restore.
  // A project's archived tab is where a read-only member is now listed.
  await page.getByRole("tab", { name: "Archived" }).click();
  await page.getByRole("tabpanel").getByText(featureTitle).click();
  await expect(page.getByText("Project archived · read-only")).toBeVisible();
  await expect(page.getByRole("button", { name: "Sync GitHub" })).toHaveCount(
    0,
  );
  await page.getByRole("button", { name: "Manage feature" }).click();
  const manage = page.getByRole("dialog", {
    name: "Manage archived feature",
  });
  await expect(
    manage.getByRole("button", { name: "Restore feature" }),
  ).toHaveCount(0);
  await manage.getByRole("button", { name: "Close" }).click();

  await page.getByRole("link", { name: "Open project" }).click();
  await page.getByRole("button", { name: "Manage project" }).click();
  await page.getByRole("button", { name: "Activate project" }).click();
  await expect(page.getByText("Archived · read-only")).toHaveCount(0);
  // The member is back in flight, so it leaves the archived tab it was on.
  await expect(page.getByRole("tabpanel")).not.toContainText(featureTitle);
  await page.getByRole("tab", { name: "Active" }).click();
  await page.getByRole("tabpanel").getByText(featureTitle).click();
  await expect(page.getByRole("button", { name: "Sync GitHub" })).toBeVisible();

  // Deleting the project takes the feature it holds with it.
  await page.locator(".workspace-project-link").click();
  await page.getByRole("button", { name: "Edit project" }).click();
  await page.getByRole("button", { name: "Delete project" }).click();
  await page
    .getByRole("dialog", { name: `Delete ${title}?` })
    .getByRole("button", { name: "Delete permanently" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Projects", exact: true }),
  ).toBeVisible();
  // The feature went with the project, so nothing on the projects screen or in
  // task search still names it.
  await expect(
    page.getByRole("region", { name: "Project list" }),
  ).not.toContainText(title);
  await page.goto("/tasks");
  await expect(page.locator("body")).not.toContainText(featureTitle);
});

// The sidebar tree folds a project away and remembers that across a reload,
// and its feature rows open the workspaces directly.
test("folds a sidebar project and restores the fold after a reload", async ({
  page,
}) => {
  await page.goto("/");
  const rail = page.getByRole("navigation", { name: "PRX navigation" });
  const toggle = rail.getByRole("button", {
    name: "Expand or collapse Delivery platform",
  });
  const child = rail.getByRole("link", { name: /Delivery control showcase/ });

  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  await expect(child).toBeVisible();

  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await expect(child).toBeHidden();

  await page.reload();
  await expect(
    rail.getByRole("button", { name: "Expand or collapse Delivery platform" }),
  ).toHaveAttribute("aria-expanded", "false");

  await rail
    .getByRole("button", { name: "Expand or collapse Delivery platform" })
    .click();
  await child.click();
  await expect(
    page.getByRole("heading", { name: "Delivery control showcase" }),
  ).toBeVisible();
});

// Between 601px and 900px the rail is one horizontal row of links, which a
// nested list cannot sit in. The tree goes away there and the Projects link
// stays, so the page keeps carrying the tree's job.
test("drops the sidebar tree once the rail turns horizontal", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1200, height: 900 });
  await page.goto("/");
  const rail = page.getByRole("navigation", { name: "PRX navigation" });
  await expect(rail.locator(".nav-tree")).toBeVisible();

  await page.setViewportSize({ width: 800, height: 900 });
  await expect(rail.locator(".nav-tree")).toBeHidden();
  await expect(rail.getByRole("link", { name: /Projects/ })).toBeVisible();
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 800);
});
