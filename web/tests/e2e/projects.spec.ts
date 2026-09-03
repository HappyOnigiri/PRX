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
async function createProject(page: Page, slug: string, title: string) {
  await page.goto("/projects");
  await page.getByRole("button", { name: "New project" }).click();
  const dialog = page.getByRole("form", { name: "Create project" });
  await dialog.getByLabel("Slug").fill(slug);
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

  await page.getByRole("button", { name: "Show archived" }).click();
  await expect(page).toHaveURL(/archived=true/);
  await expect(
    page.getByRole("region", { name: "Project list" }),
  ).toContainText("Sunset initiative");

  // Reloading has to reproduce the archived view, which is why the toggle
  // lives in the URL rather than in browser-local state.
  await page.reload();
  await expect(
    page.getByRole("region", { name: "Project list" }),
  ).toContainText("Sunset initiative");
  await page.getByRole("button", { name: "Show active" }).click();

  await page
    .getByRole("region", { name: "Project list" })
    .getByText("Delivery platform")
    .click();
  const features = page.getByRole("region", {
    name: "Features in this project",
  });
  await expect(features).toContainText("Delivery control showcase");
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
  const slug = `e2e-project-${crypto.randomUUID()}`;
  const title = `E2E project ${slug}`;
  await createProject(page, slug, title);

  const featureSlug = `e2e-member-${crypto.randomUUID()}`;
  const featureTitle = `E2E member ${featureSlug}`;
  await page.getByRole("button", { name: "New feature" }).click();
  const featureDialog = page.getByRole("form", { name: "Create feature" });
  await featureDialog.getByLabel("Slug").fill(featureSlug);
  await featureDialog.getByLabel("Title").fill(featureTitle);
  await featureDialog.getByLabel("Project").selectOption({ label: title });
  await featureDialog.getByRole("button", { name: "Create feature" }).click();
  await expect(page.getByRole("heading", { name: featureTitle })).toBeVisible();
  // The feature header names the project it now belongs to.
  await expect(page.locator(".workspace-project-link")).toHaveText(title);

  // The sidebar links to the project too, so follow the one in the header.
  await page.locator(".workspace-project-link").click();
  await expect(
    page.getByRole("region", { name: "Features in this project" }),
  ).toContainText(featureTitle);
  await page.getByRole("button", { name: "Edit project" }).click();
  await page.getByRole("button", { name: "Archive project" }).click();
  await page
    .getByRole("dialog", { name: `Archive ${title}?` })
    .getByRole("button", { name: "Archive project" })
    .click();
  await expect(page.getByText("Archived · read-only")).toBeVisible();

  // The feature itself is not archived, so the notice has to say the archive
  // came from the project and link back to it instead of offering a restore.
  await page.goto("/archived");
  await page.locator(".feature-list").getByText(featureTitle).click();
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
  await page
    .getByRole("region", { name: "Features in this project" })
    .getByText(featureTitle)
    .click();
  await expect(page.getByRole("button", { name: "Sync GitHub" })).toBeVisible();

  // Deleting the project releases the feature rather than removing it.
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
  await page.goto("/active");
  await expect(page.locator(".feature-list")).toContainText(featureTitle);
});
