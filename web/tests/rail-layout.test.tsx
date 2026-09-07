import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setDisplayLanguage } from "../src/i18n";
import {
  maxRailWidth,
  minRailWidth,
  webUISettingsKey,
} from "../src/i18n/settings";
import { AppShell } from "../src/shell";
import { makeFeature, makeProject, makeSnapshot } from "./factories";

const snapshot = makeSnapshot({
  projects: [makeProject({ id: "P-1", title: "Delivery platform" })],
  features: [makeFeature({ id: "F-1", title: "Checkout", projectId: "P-1" })],
});

const mutation = { mutateAsync: vi.fn(), isPending: false, error: null };

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    className,
  }: {
    children: ReactNode;
    className?: string;
  }) => <span className={className}>{children}</span>,
  useNavigate: () => vi.fn(),
}));
vi.mock("../src/api", () => ({
  mutations: { createFeature: vi.fn() },
  configMutations: {},
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => ({ data: snapshot, isError: false }),
  useAutoSync: () => ({
    enabled: true,
    status: { data: undefined, isError: false },
    checking: false,
    error: null,
  }),
  useDomainMutation: () => mutation,
  useConfig: () => ({ data: { hosts: [], authMethods: [] }, isPending: false }),
  useConfigMutation: () => mutation,
}));

function shell() {
  const element = document.querySelector<HTMLElement>(".app-shell");
  if (!element) throw new Error("the app shell is not rendered");
  return element;
}

function railWidth() {
  return shell().style.getPropertyValue("--rail-width");
}

function stored(): Record<string, unknown> {
  const value = localStorage.getItem(webUISettingsKey);
  return value ? (JSON.parse(value) as Record<string, unknown>) : {};
}

function resizer() {
  return screen.getByRole("separator", { name: "Sidebar width" });
}

function renderShell() {
  render(
    <AppShell>
      <p>Workspace</p>
    </AppShell>,
  );
}

// jsdom は CSS を読まないので、幅と最小化の契約はインラインの --rail-width と
// .app-shell の data 属性に置く。実際の見た目は E2E が受け持つ。
describe("AppShell rail layout", () => {
  afterEach(cleanup);
  beforeEach(async () => {
    localStorage.clear();
    await setDisplayLanguage("en");
  });

  it("resizes the rail by dragging and saves the width only when the drag ends", () => {
    renderShell();
    expect(railWidth()).toBe("248px");
    expect(resizer()).toHaveAttribute("aria-valuenow", "248");
    expect(resizer()).toHaveAttribute("aria-valuemin", String(minRailWidth));
    expect(resizer()).toHaveAttribute("aria-valuemax", String(maxRailWidth));
    expect(resizer()).toHaveAttribute("aria-controls", "prx-rail");

    fireEvent.pointerDown(resizer(), { button: 0, pointerId: 1, clientX: 248 });
    fireEvent.pointerMove(resizer(), { pointerId: 1, clientX: 328 });
    expect(railWidth()).toBe("328px");
    expect(shell()).toHaveAttribute("data-rail-resizing", "true");
    expect(stored()["railWidth"]).toBeUndefined();

    fireEvent.pointerUp(resizer(), { pointerId: 1, clientX: 328 });
    expect(railWidth()).toBe("328px");
    expect(shell()).not.toHaveAttribute("data-rail-resizing");
    expect(stored()["railWidth"]).toBe(328);
  });

  it("keeps a drag inside the allowed width range", () => {
    renderShell();
    fireEvent.pointerDown(resizer(), { button: 0, pointerId: 1, clientX: 248 });
    fireEvent.pointerMove(resizer(), { pointerId: 1, clientX: -400 });
    expect(railWidth()).toBe(`${minRailWidth}px`);
    fireEvent.pointerMove(resizer(), { pointerId: 1, clientX: 2000 });
    expect(railWidth()).toBe(`${maxRailWidth}px`);
    fireEvent.pointerUp(resizer(), { pointerId: 1, clientX: 2000 });
    expect(stored()["railWidth"]).toBe(maxRailWidth);
  });

  // ポインタを離せないまま操作が奪われても、幅とドラッグ中の印を残さない。
  it("finishes the drag when the pointer is cancelled", () => {
    renderShell();
    fireEvent.pointerDown(resizer(), { button: 0, pointerId: 1, clientX: 248 });
    fireEvent.pointerCancel(resizer(), { pointerId: 1, clientX: 300 });
    expect(railWidth()).toBe("300px");
    expect(shell()).not.toHaveAttribute("data-rail-resizing");
    expect(stored()["railWidth"]).toBe(300);
  });

  it("ignores a secondary button and a foreign pointer", () => {
    renderShell();
    fireEvent.pointerDown(resizer(), { button: 2, pointerId: 1, clientX: 248 });
    fireEvent.pointerMove(resizer(), { pointerId: 1, clientX: 400 });
    expect(railWidth()).toBe("248px");

    fireEvent.pointerDown(resizer(), { button: 0, pointerId: 1, clientX: 248 });
    fireEvent.pointerMove(resizer(), { pointerId: 9, clientX: 400 });
    fireEvent.pointerUp(resizer(), { pointerId: 9, clientX: 400 });
    expect(railWidth()).toBe("248px");
    expect(stored()["railWidth"]).toBeUndefined();
  });

  it("adjusts and saves the width from the keyboard", () => {
    renderShell();
    fireEvent.keyDown(resizer(), { key: "ArrowRight" });
    expect(railWidth()).toBe("264px");
    expect(stored()["railWidth"]).toBe(264);
    fireEvent.keyDown(resizer(), { key: "ArrowLeft" });
    expect(railWidth()).toBe("248px");
    fireEvent.keyDown(resizer(), { key: "Home" });
    expect(railWidth()).toBe(`${minRailWidth}px`);
    fireEvent.keyDown(resizer(), { key: "End" });
    expect(railWidth()).toBe(`${maxRailWidth}px`);
    fireEvent.keyDown(resizer(), { key: "Enter" });
    expect(railWidth()).toBe(`${maxRailWidth}px`);
    expect(stored()["railWidth"]).toBe(maxRailWidth);
  });

  it("hides and restores the rail and moves focus to the other control", () => {
    renderShell();
    const hide = screen.getByRole("button", { name: "Hide the sidebar" });
    const show = screen.getByRole("button", { name: "Show the sidebar" });
    expect(hide).toHaveAttribute("aria-expanded", "true");
    expect(show).toHaveAttribute("aria-expanded", "false");
    expect(shell()).not.toHaveAttribute("data-rail-collapsed");

    fireEvent.click(hide);
    expect(shell()).toHaveAttribute("data-rail-collapsed", "true");
    expect(stored()["railCollapsed"]).toBe(true);
    expect(show).toHaveFocus();

    fireEvent.click(show);
    expect(shell()).not.toHaveAttribute("data-rail-collapsed");
    expect(stored()["railCollapsed"]).toBe(false);
    expect(hide).toHaveFocus();
  });

  // 最小化した状態で開いたときに勝手にフォーカスを奪うと、読み手はページの
  // 途中から読み始めることになる。
  it("restores the saved width and collapse without taking focus", () => {
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ railWidth: 300, railCollapsed: true }),
    );
    renderShell();
    expect(railWidth()).toBe("300px");
    expect(shell()).toHaveAttribute("data-rail-collapsed", "true");
    expect(document.body).toHaveFocus();
  });

  // 初期描画で書き込むと、まだ何も触っていない利用者の設定を上書きしてしまう。
  it("does not save the rail layout on the first render", () => {
    renderShell();
    expect(stored()).toEqual({ language: "en" });
  });
});
