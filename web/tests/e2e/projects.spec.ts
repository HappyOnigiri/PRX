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

// デモデータは全ワーカーで共有するため、各テストは自前の project と feature を
// 作り、他のテストが読むものはアーカイブしない。
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

  // リロードしてもアーカイブ表示を再現する必要があるため、タブの状態は
  // ブラウザローカルではなく URL に持たせている。
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

// サイドバーの project 行は feature 行のクラスを流用するので、1 カラムの
// track 指定が feature 側に勝つ必要がある。この上書きが失われると、行が隠れる
// のではなくタイトルが 8px のステータスドット列に押し込まれる。
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

  // feature は project ページから作るので、所属はダイアログの入力欄ではなく
  // ページから決まる。
  const featureTitle = `E2E member ${crypto.randomUUID()}`;
  await page.getByRole("button", { name: "Create feature" }).click();
  const featureDialog = page.getByRole("form", { name: "Create feature" });
  await expect(featureDialog.getByLabel("Project")).toHaveCount(0);
  await featureDialog.getByLabel("Title").fill(featureTitle);
  await featureDialog.getByRole("button", { name: "Create feature" }).click();
  await expect(page.getByRole("heading", { name: featureTitle })).toBeVisible();
  // feature のヘッダーには、所属することになった project 名が出る。
  await expect(page.locator(".workspace-project-link")).toHaveText(title);

  // サイドバーにも project へのリンクがあるので、ヘッダー側をたどる。
  await page.locator(".workspace-project-link").click();
  await expect(page.getByRole("tabpanel")).toContainText(featureTitle);
  await page.getByRole("button", { name: "Edit project" }).click();
  await page.getByRole("button", { name: "Archive project" }).click();
  await page
    .getByRole("dialog", { name: `Archive ${title}?` })
    .getByRole("button", { name: "Archive project" })
    .click();
  await expect(page.getByText("Archived · read-only")).toBeVisible();

  // feature 自体はアーカイブされていないので、通知は復元を促すのではなく
  // project 由来のアーカイブであることを示し、project へ戻すリンクを出す。
  // read-only になったメンバーは project のアーカイブタブに並ぶ。
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
  // メンバーが進行中に戻るので、いたアーカイブタブから外れる。
  await expect(page.getByRole("tabpanel")).not.toContainText(featureTitle);
  await page.getByRole("tab", { name: "Active" }).click();
  await page.getByRole("tabpanel").getByText(featureTitle).click();
  await expect(page.getByRole("button", { name: "Sync GitHub" })).toBeVisible();

  // project を削除すると、抱えている feature も一緒に消える。
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
  // feature は project と一緒に消えるので、projects 画面にもタスク検索にも
  // その名前は残らない。
  await expect(
    page.getByRole("region", { name: "Project list" }),
  ).not.toContainText(title);
  await page.goto("/tasks");
  await expect(page.locator("body")).not.toContainText(featureTitle);
});

// サイドバーのツリーは project を畳めて、その状態をリロード後も覚えている。
// feature 行からはワークスペースを直接開ける。
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

// 601px〜900px では rail がリンク 1 行の横並びになり、入れ子のリストは置けない。
// そこではツリーが消えて Projects リンクが残り、ツリーの役割はページ側が担う。
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
