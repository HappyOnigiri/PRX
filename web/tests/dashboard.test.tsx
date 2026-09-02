import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { router } from "../src/router";
import { makeFeature, makeSnapshot, makeTask } from "./factories";

const snapshot = makeSnapshot({
  features: [
    makeFeature({
      taskCount: 3,
      readyCount: 1,
      reviewWaitingCount: 1,
      conflictCount: 1,
    }),
  ],
  tasks: [],
  readyTasks: [makeTask()],
  reviewWaitingTasks: [makeTask({ id: "task-2" })],
  conflictTasks: [makeTask({ id: "task-3" })],
});

vi.mock("../src/hooks", () => ({
  useSnapshot: () => ({
    data: snapshot,
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useAutoSync: () => ({
    status: { data: undefined, isError: false },
    checking: false,
    error: null,
  }),
  useDomainMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  }),
}));

describe("Dashboard", () => {
  afterEach(cleanup);
  beforeEach(async () => {
    localStorage.clear();
    router.update({ history: createMemoryHistory({ initialEntries: ["/"] }) });
    await router.load();
  });

  it("shows derived queues and the next task", async () => {
    render(<RouterProvider router={router} />);
    expect(
      await screen.findByRole("heading", { name: /What can move/ }),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(
        "Every queue is derived from the dependency graph—never manually marked ready.",
      ),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("nodes under control")).not.toBeInTheDocument();
    expect(screen.getByText("Build API")).toBeInTheDocument();
    expect(screen.getByText("1 ready")).toBeInTheDocument();
    expect(screen.getByText("Conflicts")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Sync GitHub" }),
    ).toBeInTheDocument();
  });
});
