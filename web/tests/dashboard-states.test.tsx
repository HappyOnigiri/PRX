import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Snapshot } from "../src/gen/prx/v1/prx_pb";
import { Dashboard } from "../src/views/Dashboard";
import { makeFeature, makeSnapshot, makeTask } from "./factories";

const dashboardMocks = vi.hoisted(() => ({
  state: {
    data: undefined as Snapshot | undefined,
    isPending: true,
    error: null as Error | null,
    refetch: vi.fn(),
  },
}));

vi.mock("../src/hooks", () => ({
  useSnapshot: () => dashboardMocks.state,
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
  });

  it("shows loading and error states with a retry action", () => {
    const { rerender } = render(<Dashboard />);
    expect(
      screen.getByRole("heading", { name: "Mapping dependencies…" }),
    ).toBeInTheDocument();

    dashboardMocks.state.isPending = false;
    dashboardMocks.state.error = new Error("database unavailable");
    rerender(<Dashboard />);
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
    render(<Dashboard />);

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
    const { container } = render(<Dashboard />);

    expect(container.querySelector(".clock span")).toHaveTextContent("1");
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
});
