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
  await page.getByLabel("Display language").selectOption("ja");
  // 表示の設定はフッタの保存 1 つで適用する。
  await page
    .getByRole("dialog", { name: "Settings" })
    .getByRole("button", { name: "Save" })
    .click();
  await expect(banner).toBeVisible();
  await page.getByRole("button", { name: "閉じる" }).first().click();

  await page.setViewportSize({ width: 320, height: 720 });
  await page.evaluate(() => {
    document.body.style.zoom = "2";
  });
  // toContainText は textContent を見るため、スクリーンリーダーに読む内容が
  // 残っていなくても、隠れた広幅用の文言だけで条件を満たしてしまう。
  const compact = banner.locator(".demo-banner-compact");
  await expect(compact).toBeVisible();
  await expect(banner.locator(".demo-banner-full")).toBeHidden();
  await expect(compact).toHaveText("DEMO · Reset on restart再起動でリセット");
  // 閉じるボタンのアイコンは装飾なので svg は数えない。
  expect(
    await banner.evaluate(
      (element) =>
        Array.from(element.querySelectorAll("[aria-hidden='true']:not(svg)"))
          .length,
    ),
  ).toBe(0);
});

test("keeps the dismissed demo warning hidden until the server restarts", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("button", {
      name: "Hide the demo notice until the demo server restarts",
    })
    .click();
  await expect(page.getByRole("status")).toBeHidden();
  await expect(page.locator(".app-shell")).not.toHaveAttribute("data-demo");

  await page.reload();
  await expect(page.locator(".app-shell")).toBeVisible();
  await expect(page.getByRole("status")).toBeHidden();

  // サーバを起動し直すと HTML の ID が変わる。保存済みの ID を別の値に書き換えて
  // 同じ状況を作る。
  await page.evaluate(() => {
    localStorage.setItem(
      "prx.webui.demoNoticeDismissedSession",
      "restarted-server",
    );
  });
  await page.reload();
  await expect(page.getByRole("status")).toBeVisible();
});
