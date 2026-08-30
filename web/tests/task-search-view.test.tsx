import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TaskDisplayState, type Snapshot } from "../src/gen/prx/v1/prx_pb";
import { TaskSearch } from "../src/views/TaskSearch";
import { makeSnapshot, makeTask } from "./factories";

const mocks = vi.hoisted(() => ({
  query: "task-status:ready",
  navigate: vi.fn(),
  state: {
    isPending: false,
    error: null as Error | null,
    refetch: vi.fn(),
    data: undefined as Snapshot | undefined,
  },
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => (
    <a href="#feature">{children}</a>
  ),
  useNavigate: () => mocks.navigate,
  useSearch: () => ({ q: mocks.query }),
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => mocks.state,
}));

describe("TaskSearch view", () => {
  afterEach(cleanup);
  beforeEach(() => {
    mocks.query = "task-status:ready";
    mocks.navigate.mockReset();
    mocks.state.isPending = false;
    mocks.state.error = null;
    mocks.state.refetch.mockReset();
    mocks.state.data = makeSnapshot({
      tasks: [
        makeTask({
          id: "ready-task",
          title: "Ready task",
          displayState: TaskDisplayState.NOT_STARTED,
          ready: true,
        }),
        makeTask({
          id: "blocked-task",
          title: "Blocked task",
          displayState: TaskDisplayState.IN_PROGRESS,
          ready: false,
        }),
      ],
    });
  });

  it("renders active search results and keeps the query in navigation", () => {
    render(<TaskSearch />);

    expect(
      screen.getByRole("heading", { name: "Task search" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: "Search active tasks" }),
    ).toHaveValue("task-status:ready");
    expect(screen.getByText("Ready task")).toBeInTheDocument();
    expect(screen.queryByText("Blocked task")).not.toBeInTheDocument();

    const input = screen.getByRole("textbox", { name: "Search active tasks" });
    fireEvent.change(input, { target: { value: "payments" } });
    fireEvent.submit(screen.getByRole("search"));
    expect(mocks.navigate).toHaveBeenCalledWith({
      to: "/tasks",
      search: { q: "payments" },
    });
  });

  it("distinguishes invalid syntax from an empty result", () => {
    mocks.query = "github-status:waiting";
    const { rerender } = render(<TaskSearch />);
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Search needs a correction",
    );

    mocks.query = "does-not-exist";
    rerender(<TaskSearch />);
    expect(
      screen.getByRole("heading", { name: "No matching active tasks" }),
    ).toBeInTheDocument();
  });

  it("offers a retry when the snapshot cannot be loaded", () => {
    mocks.state.error = new Error("snapshot unavailable");
    render(<TaskSearch />);

    expect(
      screen.getByRole("heading", {
        name: "Task search could not be loaded",
      }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(mocks.state.refetch).toHaveBeenCalledOnce();
  });
});
