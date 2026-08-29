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
      kind: 2,
      title: "Plan",
      value: "docs/plan.md",
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
            id: "0",
            sources: ["task-1"],
            targets: ["task-2"],
          },
        ],
      }),
    );

    unmount();
    expect(layoutMocks.terminateWorker).toHaveBeenCalledOnce();
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
