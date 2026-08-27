import { render, screen } from "@testing-library/react";
import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { router } from "../src/router";

const snapshot = {
  features: [
    {
      id: "feature-1",
      slug: "payments",
      title: "Payments rollout",
      description: "",
      status: "active",
      archived: false,
      createdAt: "",
      updatedAt: "",
      taskCount: 3,
      readyCount: 1,
      reviewWaitingCount: 1,
      conflictCount: 1,
      mergedCount: 0,
      $typeName: "prmap.v1.Feature",
    },
  ],
  tasks: [],
  dependencies: [],
  pullRequests: [],
  documents: [],
  readyTasks: [
    {
      id: "task-1",
      featureId: "feature-1",
      title: "Build API",
      scope: "",
      kind: "pr",
      status: "planned",
      assignee: "Mika",
      createdAt: "",
      updatedAt: "",
      ready: true,
      displayState: "unlinked",
      blockedReason: "",
      $typeName: "prmap.v1.Task",
    },
  ],
  reviewWaitingTasks: [{ id: "task-2" }],
  conflictTasks: [{ id: "task-3" }],
  staleTasks: [],
  $typeName: "prmap.v1.Snapshot",
};

vi.mock("../src/hooks", () => ({
  useSnapshot: () => ({
    data: snapshot,
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useDomainMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  }),
}));

describe("Dashboard", () => {
  beforeEach(async () => {
    router.update({ history: createMemoryHistory({ initialEntries: ["/"] }) });
    await router.load();
  });
  it("shows derived queues and the next task", async () => {
    render(<RouterProvider router={router} />);
    expect(
      await screen.findByRole("heading", { name: /What can move/ }),
    ).toBeInTheDocument();
    expect(screen.getByText("Build API")).toBeInTheDocument();
    expect(screen.getByText("1 ready")).toBeInTheDocument();
    expect(screen.getByText("Conflicts")).toBeInTheDocument();
  });
});
