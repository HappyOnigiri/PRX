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

// この spec は幅と最小化しか触らないので、デモグラフを読むだけの他の spec と
// 同じサーバーに対して並列に実行する。
test("resizes the sidebar by dragging and keeps the width after a reload", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1200, height: 900 });
  await page.goto("/");
  const rail = page.locator(".rail");
  const handle = page.getByRole("separator", { name: "Sidebar width" });
  expect((await rail.boundingBox())?.width).toBe(248);

  const box = await handle.boundingBox();
  if (!box) throw new Error("the resize handle is not visible");
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2 + 80, box.y + box.height / 2);
  await page.mouse.up();
  await expect(handle).toHaveAttribute("aria-valuenow", "328");
  expect((await rail.boundingBox())?.width).toBe(328);

  await page.reload();
  expect((await page.locator(".rail").boundingBox())?.width).toBe(328);
});

// 隠している間は main が全幅になり、復元ボタンだけが残る。幅の記憶は最小化を
// またいでも消えない。
test("hides the sidebar and restores it at the remembered width", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1200, height: 900 });
  await page.goto("/");
  const rail = page.locator(".rail");
  const main = page.locator(".main-stage");
  await page.getByRole("separator", { name: "Sidebar width" }).press("End");
  expect((await rail.boundingBox())?.width).toBe(480);

  await page.getByRole("button", { name: "Hide the sidebar" }).click();
  await expect(rail).toBeHidden();
  expect((await main.boundingBox())?.width).toBe(1200);

  const restore = page.getByRole("button", { name: "Show the sidebar" });
  await expect(restore).toBeVisible();
  await restore.click();
  await expect(rail).toBeVisible();
  expect((await rail.boundingBox())?.width).toBe(480);
});

// 最小化を CSS で表現しているので、rail が横 1 行に変形する幅まで縮めると
// ナビゲーションが自動的に戻る。JS でアンマウントしていたら消えたままになる。
test("brings the hidden sidebar back once the rail turns horizontal", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1200, height: 900 });
  await page.goto("/");
  await page.getByRole("button", { name: "Hide the sidebar" }).click();
  await expect(page.locator(".rail")).toBeHidden();

  await page.setViewportSize({ width: 800, height: 900 });
  await expect(page.locator(".rail")).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "PRX navigation" }),
  ).toBeVisible();
  await expect(
    page.getByRole("separator", { name: "Sidebar width" }),
  ).toBeHidden();
  await expect(
    page.getByRole("button", { name: "Hide the sidebar" }),
  ).toBeHidden();
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 800);
});
