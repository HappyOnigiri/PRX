import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureGraph } from "../src/views/FeatureGraph";
import { makeDependency, makeTask } from "./factories";

const graphMocks = vi.hoisted(() => ({
  useGraphLayout: vi.fn(),
  writeGraphZoom: vi.fn(),
}));

vi.mock("@xyflow/react", () => ({
  Background: () => null,
  BackgroundVariant: { Dots: "dots" },
  Controls: () => null,
  MarkerType: { ArrowClosed: "arrowclosed" },
  ReactFlow: ({
    children,
    edges,
    onMoveEnd,
  }: {
    children?: ReactNode;
    edges?: unknown[];
    onMoveEnd?: (...args: unknown[]) => void;
  }) => (
    <button
      type="button"
      data-testid="mock-react-flow"
      data-edge-count={edges?.length ?? 0}
      onClick={() => onMoveEnd?.({}, { zoom: 1.25 })}
    >
      {children}
    </button>
  ),
}));
vi.mock("../src/i18n/settings", () => ({
  maxGraphZoom: 1.7,
  minGraphZoom: 0.08,
  readGraphZoom: vi.fn(() => 1),
  writeGraphZoom: graphMocks.writeGraphZoom,
}));
vi.mock("../src/views/useGraphLayout", () => ({
  useGraphLayout: graphMocks.useGraphLayout,
}));

describe("FeatureGraph", () => {
  afterEach(cleanup);

  beforeEach(() => {
    vi.clearAllMocks();
    graphMocks.useGraphLayout.mockReturnValue({
      nodes: [],
      layoutError: undefined,
      retryLayout: vi.fn(),
    });
  });

  it("shows the empty graph action and creates a task", () => {
    const onCreateTask = vi.fn();
    render(
      <FeatureGraph
        tasks={[]}
        dependencies={[]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={onCreateTask}
      />,
    );

    expect(
      screen.getByRole("heading", { name: "Draw the first node" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Add task" }));
    expect(onCreateTask).toHaveBeenCalledOnce();
  });

  it("builds dependency edges, persists zoom, and retries a failed layout", () => {
    const retryLayout = vi.fn();
    graphMocks.useGraphLayout.mockReturnValue({
      nodes: [makeTask()],
      layoutError: { message: undefined },
      retryLayout,
    });
    render(
      <FeatureGraph
        tasks={[makeTask()]}
        dependencies={[makeDependency()]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={vi.fn()}
      />,
    );

    expect(screen.getByText("1 nodes · 1 links")).toBeInTheDocument();
    expect(screen.getByTestId("mock-react-flow")).toHaveAttribute(
      "data-edge-count",
      "1",
    );
    expect(screen.getByText("Graph layout failed.")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("mock-react-flow"));
    expect(graphMocks.writeGraphZoom).toHaveBeenCalledWith(1.25);
    fireEvent.click(screen.getByRole("button", { name: "Retry layout" }));
    expect(retryLayout).toHaveBeenCalledOnce();
  });
});
