import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useGraphLayout } from "../src/views/useGraphLayout";
import {
  makeDependency,
  makeDocument,
  makePullRequest,
  makeTask,
} from "./factories";

const layoutMocks = vi.hoisted(() => ({
  layout: vi.fn(),
  terminateWorker: vi.fn(),
}));

vi.mock("elkjs/lib/elk-api.js", () => ({
  default: class {
    layout(...args: unknown[]) {
      return layoutMocks.layout(...args) as Promise<unknown>;
    }

    terminateWorker() {
      layoutMocks.terminateWorker();
    }
  },
}));

describe("useGraphLayout", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("builds positioned task nodes with pull requests, documents, and edges", async () => {
    layoutMocks.layout.mockResolvedValue({
      children: [{ id: "task-1", x: 120, y: 48 }],
    });
    const onEditTask = vi.fn();
    const onPreviewDocument = vi.fn();
    const document = makeDocument({
      id: "document-1",
      kind: 3,
      title: "Plan",
      locator: "docs/plan.md",
    });
    const options = {
      tasks: [makeTask({ assignee: "Ren" })],
      dependencies: [makeDependency()],
      pullRequests: new Map([["task-1", makePullRequest({ stale: true })]]),
      documentsByTask: new Map([["task-1", [document]]]),
      onEditTask,
      onPreviewDocument,
    };
    const { result, unmount } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.nodes).toHaveLength(1);
    });
    const node = result.current.nodes[0];
    expect(node).toMatchObject({
      id: "task-1",
      type: "task",
      position: { x: 120, y: 48 },
      data: {
        title: "Build API",
        assignee: "Ren",
        stale: true,
        pullRequest: {
          label: "acme/prx #42",
          url: "https://github.com/acme/prx/pull/42",
        },
        documents: [document],
      },
    });
    node?.data.onEdit();
    expect(onEditTask).toHaveBeenCalledWith("task-1");
    expect(layoutMocks.layout).toHaveBeenCalledWith(
      expect.objectContaining({
        edges: [
          {
            id: "task-1-task-2",
            sources: ["task-1"],
            targets: ["task-2"],
          },
        ],
      }),
    );

    unmount();
    expect(layoutMocks.terminateWorker).toHaveBeenCalledOnce();
  });

  it("keeps ELK edge routes and assigns a distinct port to each endpoint", async () => {
    layoutMocks.layout.mockResolvedValue({
      children: [
        { id: "task-1", x: 12, y: 20 },
        { id: "task-2", x: 406, y: 80 },
      ],
      edges: [
        {
          id: "task-1-task-2",
          sources: ["task-1"],
          targets: ["task-2"],
          sections: [
            {
              id: "route",
              startPoint: { x: 296, y: 92 },
              bendPoints: [
                { x: 340, y: 92 },
                { x: 340, y: 154 },
              ],
              endPoint: { x: 406, y: 154 },
            },
          ],
        },
      ],
    });
    const options = {
      tasks: [makeTask(), makeTask({ id: "task-2" })],
      dependencies: [makeDependency()],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.edgeRoutes.size).toBe(1);
    });

    expect(result.current.edgeRoutes.get("task-1-task-2")).toEqual({
      points: [
        { x: 296, y: 92 },
        { x: 340, y: 92 },
        { x: 340, y: 154 },
        { x: 406, y: 154 },
      ],
      sourcePortId: "task-1-task-2-source",
      sourcePortTop: 72,
      targetPortId: "task-1-task-2-target",
      targetPortTop: 74,
    });
    expect(result.current.nodes[0]?.data.outgoingPorts).toEqual([
      { id: "task-1-task-2-source", top: 72 },
    ]);
    expect(result.current.nodes[1]?.data.incomingPorts).toEqual([
      { id: "task-1-task-2-target", top: 74 },
    ]);
    const layoutInput: unknown = layoutMocks.layout.mock.calls[0]?.[0];
    if (
      !layoutInput ||
      typeof layoutInput !== "object" ||
      !("layoutOptions" in layoutInput)
    )
      throw new Error("ELK layout options missing");
    expect(layoutInput.layoutOptions).toMatchObject({
      "elk.edgeRouting": "ORTHOGONAL",
      "elk.layered.mergeEdges": "false",
    });
  });

  it("reports layout errors and retries the layout", async () => {
    layoutMocks.layout
      .mockRejectedValueOnce(new Error("worker unavailable"))
      .mockResolvedValueOnce({ children: [] });
    const options = {
      tasks: [makeTask()],
      dependencies: [],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result, unmount } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.layoutError).toEqual({
        message: "worker unavailable",
      });
    });
    act(() => {
      result.current.retryLayout();
    });
    await waitFor(() => {
      expect(layoutMocks.layout).toHaveBeenCalledTimes(2);
    });
    expect(result.current.layoutError).toBeUndefined();
    unmount();
  });
});
