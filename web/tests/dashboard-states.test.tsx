import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Snapshot } from "../src/gen/prx/v1/prx_pb";
import { AutoSyncStatusContext, type AutoSyncStatus } from "../src/sync-status";
import { Dashboard } from "../src/views/Dashboard";
import {
  makeFeature,
  makeGitHubSyncStatus,
  makeSnapshot,
  makeTask,
} from "./factories";

const dashboardMocks = vi.hoisted(() => ({
  state: {
    data: undefined as Snapshot | undefined,
    isPending: true,
    error: null as Error | null,
    refetch: vi.fn(),
  },
  sync: {
    mutate: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
  api: { sync: vi.fn() },
}));

const autoSyncStatus = {
  status: { data: undefined, isError: false },
  checking: false,
  error: null,
} satisfies AutoSyncStatus;

function renderDashboard(status: AutoSyncStatus = autoSyncStatus) {
  return render(
    <AutoSyncStatusContext.Provider value={status}>
      <Dashboard />
    </AutoSyncStatusContext.Provider>,
  );
}

vi.mock("../src/hooks", () => ({
  useSnapshot: () => dashboardMocks.state,
  useDomainMutation: (mutationFn: (...input: unknown[]) => unknown) => {
    dashboardMocks.sync.mutate.mockImplementation((input: unknown) =>
      mutationFn(input, { source: "react-query" }),
    );
    return dashboardMocks.sync;
  },
}));
vi.mock("../src/api", () => ({
  mutations: dashboardMocks.api,
}));
vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => (
    <a href="#test">{children}</a>
  ),
}));

describe("Dashboard states", () => {
  afterEach(cleanup);
  beforeEach(() => {
    dashboardMocks.state.data = undefined;
    dashboardMocks.state.isPending = true;
    dashboardMocks.state.error = null;
    dashboardMocks.state.refetch.mockReset();
    dashboardMocks.sync.mutate.mockReset();
    dashboardMocks.api.sync.mockReset();
    dashboardMocks.sync.isPending = false;
    dashboardMocks.sync.error = null;
  });

  it("shows loading and error states with a retry action", () => {
    const { rerender } = renderDashboard();
    expect(
      screen.getByRole("heading", { name: "Mapping dependencies…" }),
    ).toBeInTheDocument();

    dashboardMocks.state.isPending = false;
    dashboardMocks.state.error = new Error("database unavailable");
    rerender(
      <AutoSyncStatusContext.Provider value={autoSyncStatus}>
        <Dashboard />
      </AutoSyncStatusContext.Provider>,
    );
    expect(
      screen.getByRole("heading", { name: "The roadmap could not be loaded" }),
    ).toBeInTheDocument();
    expect(screen.getByText("database unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(dashboardMocks.state.refetch).toHaveBeenCalledOnce();
  });

  it("renders empty task and feature boards", () => {
    dashboardMocks.state.isPending = false;
    dashboardMocks.state.data = makeSnapshot({
      features: [],
      tasks: [],
      readyTasks: [],
      reviewWaitingTasks: [],
      conflictTasks: [],
      staleTasks: [],
    });
    renderDashboard();

    expect(
      screen.getByRole("heading", { name: "No task is ready yet" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "No features yet" }),
    ).toBeInTheDocument();
  });

  it("projects every overview count and queue from active features", () => {
    const activeTask = makeTask({ id: "active-task", featureId: "active" });
    const archivedTask = makeTask({
      id: "archived-task",
      featureId: "archived",
      title: "Archived task",
    });
    dashboardMocks.state.isPending = false;
    dashboardMocks.state.data = makeSnapshot({
      features: [
        makeFeature({ id: "active", title: "Active graph" }),
        makeFeature({
          id: "archived",
          title: "Archived graph",
          archived: true,
        }),
      ],
      tasks: [activeTask, archivedTask],
      readyTasks: [activeTask, archivedTask],
      reviewWaitingTasks: [archivedTask],
      conflictTasks: [archivedTask],
      staleTasks: [archivedTask],
    });
    const { container } = renderDashboard();

    expect(
      container.querySelector(".page-head .clock"),
    ).not.toBeInTheDocument();
    expect(container.querySelector(".queue-ready > span")).toHaveTextContent(
      "1",
    );
    expect(
      container.querySelector(".queue-review-waiting > span"),
    ).toHaveTextContent("0");
    expect(container.querySelector(".queue-conflict > span")).toHaveTextContent(
      "0",
    );
    expect(container.querySelector(".queue-stale > span")).toHaveTextContent(
      "0",
    );
    expect(screen.getByText("Active graph")).toBeInTheDocument();
    expect(screen.queryByText("Archived graph")).not.toBeInTheDocument();
    expect(screen.queryByText("Archived task")).not.toBeInTheDocument();
  });

  it("shows the complete latest sync timestamp in the dashboard header", () => {
    const timestamp = "2026-08-30T10:24:31Z";
    dashboardMocks.state.isPending = false;
    dashboardMocks.state.data = makeSnapshot();
    renderDashboard({
      ...autoSyncStatus,
      status: {
        data: makeGitHubSyncStatus({ lastUpdatedAt: timestamp }),
        isError: false,
      },
    });

    const label = `Updated · ${new Date(timestamp).toLocaleString("en")}`;
    const time = screen.getByText(label);
    expect(time).toHaveAttribute("dateTime", timestamp);
    expect(time.closest(".page-head")).not.toBeNull();
    expect(time).toHaveTextContent(label);
  });

  it("starts a full GitHub sync from the dashboard", () => {
    dashboardMocks.state.isPending = false;
    dashboardMocks.state.data = makeSnapshot();
    renderDashboard();

    fireEvent.click(screen.getByRole("button", { name: "Sync GitHub now" }));
    expect(dashboardMocks.sync.mutate).toHaveBeenCalledOnce();
    expect(dashboardMocks.sync.mutate).toHaveBeenCalledWith(undefined);
    expect(dashboardMocks.api.sync).toHaveBeenCalledOnce();
    expect(dashboardMocks.api.sync).toHaveBeenCalledWith();
  });

  it("locks repeated syncs and reports a manual sync failure", () => {
    dashboardMocks.state.isPending = false;
    dashboardMocks.state.data = makeSnapshot();
    dashboardMocks.sync.isPending = true;
    const { rerender } = renderDashboard();

    expect(
      screen.getByRole("button", { name: "Syncing GitHub…" }),
    ).toBeDisabled();
    expect(screen.getByLabelText("Updating…")).toBeInTheDocument();

    dashboardMocks.sync.isPending = false;
    dashboardMocks.sync.error = new Error("GitHub is unavailable");
    rerender(
      <AutoSyncStatusContext.Provider value={autoSyncStatus}>
        <Dashboard />
      </AutoSyncStatusContext.Provider>,
    );
    expect(screen.getByRole("alert")).toHaveTextContent(
      "GitHub is unavailable",
    );
    expect(screen.getByLabelText("Status unavailable")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Sync GitHub now" }),
    ).toBeEnabled();
  });

  it.each([
    [
      "an update in progress",
      { ...autoSyncStatus, checking: true },
      "Updating…",
    ],
    [
      "a failed latest run",
      {
        ...autoSyncStatus,
        status: {
          data: makeGitHubSyncStatus({
            lastUpdatedAt: "2026-08-30T10:24:31Z",
            failed: 1,
          }),
          isError: false,
        },
      },
      `Failed · ${new Date("2026-08-30T10:24:31Z").toLocaleString("en")}`,
    ],
    [
      "a status query failure",
      {
        ...autoSyncStatus,
        status: { data: undefined, isError: true },
      },
      "Status unavailable",
    ],
    [
      "an automatic sync check failure",
      { ...autoSyncStatus, error: new Error("sync unavailable") },
      "Status unavailable",
    ],
  ] satisfies [string, AutoSyncStatus, string][])(
    "shows %s",
    (_name, status, label) => {
      dashboardMocks.state.isPending = false;
      dashboardMocks.state.data = makeSnapshot();
      renderDashboard(status);

      expect(screen.getByLabelText(label)).toBeInTheDocument();
    },
  );
});
