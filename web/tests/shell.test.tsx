import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureStatus } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { AppShell } from "../src/shell";
import { appVersion } from "../src/version";
import { makeFeature, makeSnapshot } from "./factories";

const shellMocks = vi.hoisted(() => ({
  navigate: vi.fn().mockResolvedValue(undefined),
  pathname: "/",
  mutation: {
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
  autoSync: vi.fn((enabled: boolean) => ({
    enabled,
    status: { data: undefined, isError: false },
    checking: false,
    error: null,
  })),
}));

const snapshot = makeSnapshot({
  features: [
    makeFeature({ id: "active", title: "Active feature", readyCount: 1 }),
    makeFeature({
      id: "conflict",
      title: "Conflict feature",
      conflictCount: 1,
      readyCount: 0,
    }),
    makeFeature({ id: "archived", title: "Archived feature", archived: true }),
    makeFeature({
      id: "completed",
      title: "Completed feature",
      displayStatus: FeatureStatus.COMPLETED,
      taskCount: 2,
      finishedCount: 2,
      readyCount: 0,
    }),
  ],
});

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    className,
  }: {
    children: ReactNode;
    className?: string;
  }) => <span className={className}>{children}</span>,
  useNavigate: () => shellMocks.navigate,
  useLocation: ({
    select,
  }: {
    select: (location: { pathname: string }) => unknown;
  }) => select({ pathname: shellMocks.pathname }),
}));
vi.mock("../src/api", () => ({
  mutations: { createFeature: vi.fn() },
  configMutations: {
    addHost: vi.fn(),
    updateHost: vi.fn(),
    deleteHost: vi.fn(),
    addAuth: vi.fn(),
    updateAuth: vi.fn(),
    deleteAuth: vi.fn(),
    reorderAuth: vi.fn(),
  },
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => ({ data: snapshot, isError: false }),
  useAutoSync: (enabled: boolean) => shellMocks.autoSync(enabled),
  useDomainMutation: () => shellMocks.mutation,
  useConfig: () => ({ data: { hosts: [], authMethods: [] }, isPending: false }),
  useConfigMutation: () => shellMocks.mutation,
}));

describe("AppShell", () => {
  afterEach(cleanup);
  // isDemoMode reads the document directly, so a meta left behind by a failed
  // assertion would render every later case in demo mode.
  afterEach(() => {
    document.querySelector('meta[name="prx-demo"]')?.remove();
  });
  beforeEach(async () => {
    localStorage.clear();
    await setDisplayLanguage("en");
    shellMocks.navigate.mockClear();
    shellMocks.autoSync.mockClear();
    shellMocks.mutation.mutateAsync.mockReset();
    shellMocks.mutation.mutateAsync.mockResolvedValue({
      feature: makeFeature({ id: "created" }),
    });
    shellMocks.mutation.isPending = false;
    shellMocks.mutation.error = null;
  });

  it("shows active features and changes display settings from Settings", async () => {
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );

    expect(screen.getByText("Active feature")).toBeInTheDocument();
    expect(screen.getByText("Conflict feature")).toBeInTheDocument();
    expect(screen.queryByText("Archived feature")).not.toBeInTheDocument();
    expect(screen.queryByText("Completed feature")).not.toBeInTheDocument();
    expect(screen.getByText("Overview")).toBeInTheDocument();
    expect(screen.getByText(/Active features/)).toHaveTextContent(
      "Active features 2",
    );
    expect(screen.getByText(/Completed features/)).toHaveTextContent(
      "Completed features 1",
    );
    expect(screen.getByText(/Archived features/)).toHaveTextContent(
      "Archived features 1",
    );
    expect(screen.getByText("Workspace")).toBeInTheDocument();
    expect(screen.getByText(`v${appVersion()}`)).toBeInTheDocument();
    expect(screen.queryByText("Local database online")).not.toBeInTheDocument();
    expect(document.querySelector(".rail-foot")).not.toBeInTheDocument();
    expect(screen.queryByText("Dependency control")).not.toBeInTheDocument();
    expect(screen.queryByText("GitHub sync")).not.toBeInTheDocument();
    expect(shellMocks.autoSync).toHaveBeenCalledWith(true);
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    expect(screen.getByRole("tab", { name: "Server" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(
      screen.queryByRole("combobox", { name: "Display language" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Display" }));
    fireEvent.change(
      screen.getByRole("combobox", { name: "Display language" }),
      {
        target: { value: "ja" },
      },
    );
    await waitFor(() => {
      expect(document.documentElement.lang).toBe("ja");
    });
    expect(screen.getByRole("dialog", { name: "設定" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "表示" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    fireEvent.change(screen.getByRole("combobox", { name: "表示テーマ" }), {
      target: { value: "dark" },
    });
    expect(document.documentElement.dataset["theme"]).toBe("dark");
    expect(
      JSON.parse(localStorage.getItem("prx.webui.settings") ?? "{}"),
    ).toEqual({ language: "ja", theme: "dark" });
  });

  it("keeps the demo reset warning visible and identifies temporary storage", () => {
    const meta = document.createElement("meta");
    meta.name = "prx-demo";
    meta.content = "true";
    document.head.append(meta);

    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );

    expect(screen.getByRole("status")).toHaveTextContent(
      "DEMO — Changes reset on restart / 変更は再起動時にリセットされます",
    );
    expect(
      screen.queryByText("Temporary demo database"),
    ).not.toBeInTheDocument();
    expect(document.querySelector(".rail-foot")).not.toBeInTheDocument();
  });

  it("creates a feature, navigates to it, and supports cancellation", async () => {
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    fireEvent.click(screen.getByRole("button", { name: "New feature" }));
    fireEvent.change(screen.getByLabelText("Slug"), {
      target: { value: "release" },
    });
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Release" },
    });
    fireEvent.change(screen.getByLabelText("Description"), {
      target: { value: "Ship it" },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Create feature" }));

    await waitFor(() => {
      expect(shellMocks.navigate).toHaveBeenCalledOnce();
    });
    expect(shellMocks.mutation.mutateAsync).toHaveBeenCalledWith({
      slug: "release",
      title: "Release",
      description: "Ship it",
    });
    expect(shellMocks.navigate).toHaveBeenCalledWith({
      to: "/features/$featureId",
      params: { featureId: "created" },
    });

    fireEvent.click(screen.getByRole("button", { name: "New feature" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("form", { name: "Create feature" }),
    ).not.toBeInTheDocument();
  });

  it("opens and closes Settings from the rail", () => {
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    expect(
      screen.getByRole("dialog", { name: "Settings" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Done" }));
    expect(
      screen.queryByRole("dialog", { name: "Settings" }),
    ).not.toBeInTheDocument();
  });
});
