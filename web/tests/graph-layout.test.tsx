import { render, screen } from "@testing-library/react";
import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { router } from "../src/router";

// React Flow measures its container, which jsdom does not implement.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver =
  ResizeObserverStub as unknown as typeof ResizeObserver;

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
      taskCount: 1,
      readyCount: 1,
      reviewWaitingCount: 0,
      conflictCount: 0,
      mergedCount: 0,
      $typeName: "prx.v1.Feature",
    },
  ],
  tasks: [
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
      $typeName: "prx.v1.Task",
    },
  ],
  dependencies: [],
  pullRequests: [],
  documents: [],
  readyTasks: [],
  reviewWaitingTasks: [],
  conflictTasks: [],
  staleTasks: [],
  $typeName: "prx.v1.Snapshot",
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

vi.mock("elkjs/lib/elk-api.js", () => ({
  default: class {
    layout() {
      return Promise.reject(new Error("worker unavailable"));
    }
    terminateWorker() {}
  },
}));

describe("FeatureWorkspace graph layout", () => {
  beforeEach(async () => {
    router.update({
      history: createMemoryHistory({ initialEntries: ["/features/feature-1"] }),
    });
    await router.load();
  });

  it("surfaces a layout failure instead of leaving the canvas blank", async () => {
    render(<RouterProvider router={router} />);
    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Graph layout failed");
    expect(alert).toHaveTextContent("worker unavailable");
    expect(
      screen.getByRole("button", { name: "Retry layout" }),
    ).toBeInTheDocument();
  });
});
