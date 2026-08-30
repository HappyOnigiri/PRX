import { Code, ConnectError } from "@connectrpc/connect";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DomainErrorCode, ErrorDetailSchema } from "../src/gen/prx/v1/prx_pb";
import { FeatureGraph } from "../src/views/FeatureGraph";
import { makeDependency, makeTask } from "./factories";

const graphMocks = vi.hoisted(() => ({
  addDependencyApi: vi.fn().mockResolvedValue({}),
  addDependency: {
    mutate: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
  useGraphLayout: vi.fn(),
  writeGraphZoom: vi.fn(),
  onConnect: undefined as ((connection: unknown) => void) | undefined,
}));

vi.mock("../src/api", () => ({
  mutations: { addDependency: graphMocks.addDependencyApi },
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
    onConnect,
    nodesConnectable,
  }: {
    children?: ReactNode;
    edges?: unknown[];
    onMoveEnd?: (...args: unknown[]) => void;
    onConnect?: (connection: unknown) => void;
    nodesConnectable?: boolean;
  }) => {
    graphMocks.onConnect = onConnect;
    return (
      <button
        type="button"
        data-testid="mock-react-flow"
        data-edge-count={edges?.length ?? 0}
        data-nodes-connectable={String(nodesConnectable)}
        onClick={() => onMoveEnd?.({}, { zoom: 1.25 })}
      >
        {children}
      </button>
    );
  },
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
vi.mock("../src/hooks", () => ({
  useDomainMutation: (mutationFn: (input: unknown) => unknown) => {
    graphMocks.addDependency.mutate.mockImplementation((input: unknown) =>
      mutationFn(input),
    );
    return graphMocks.addDependency;
  },
}));

describe("FeatureGraph", () => {
  afterEach(cleanup);

  beforeEach(() => {
    vi.clearAllMocks();
    graphMocks.addDependency.isPending = false;
    graphMocks.addDependency.error = null;
    graphMocks.onConnect = undefined;
    graphMocks.useGraphLayout.mockReturnValue({
      nodes: [],
      layoutError: undefined,
      retryLayout: vi.fn(),
    });
  });

  it("sends a blocker-to-blocked connection without adding an optimistic edge", () => {
    const blocker = makeTask({ id: "blocker", title: "Blocker task" });
    const blocked = makeTask({ id: "blocked", title: "Blocked task" });
    graphMocks.useGraphLayout.mockReturnValue({
      nodes: [],
      layoutError: undefined,
      retryLayout: vi.fn(),
    });
    render(
      <FeatureGraph
        tasks={[blocker, blocked]}
        dependencies={[]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={vi.fn()}
      />,
    );

    if (!graphMocks.onConnect) throw new Error("connection handler missing");
    graphMocks.onConnect({
      source: blocker.id,
      target: blocked.id,
      sourceHandle: null,
      targetHandle: null,
    });

    expect(graphMocks.addDependency.mutate).toHaveBeenCalledWith({
      blocker: blocker.id,
      blocked: blocked.id,
    });
    expect(graphMocks.addDependencyApi).toHaveBeenCalledWith(
      blocker.id,
      blocked.id,
    );
    expect(screen.getByTestId("mock-react-flow")).toHaveAttribute(
      "data-edge-count",
      "0",
    );
  });

  it("does not call the mutation for incomplete or pending connections", () => {
    graphMocks.addDependency.isPending = true;
    render(
      <FeatureGraph
        tasks={[makeTask()]}
        dependencies={[]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={vi.fn()}
      />,
    );

    expect(screen.getByTestId("mock-react-flow")).toHaveAttribute(
      "data-nodes-connectable",
      "false",
    );
    if (!graphMocks.onConnect) throw new Error("connection handler missing");
    graphMocks.onConnect({
      source: "task-1",
      target: "task-2",
      sourceHandle: null,
      targetHandle: null,
    });
    graphMocks.onConnect({
      source: "",
      target: "task-2",
      sourceHandle: null,
      targetHandle: null,
    });

    expect(graphMocks.addDependency.mutate).not.toHaveBeenCalled();
  });

  it("shows a translated cycle error with task titles", () => {
    const blocker = makeTask({ id: "blocker", title: "Blocker task" });
    const blocked = makeTask({ id: "blocked", title: "Blocked task" });
    graphMocks.addDependency.error = new ConnectError(
      "cycle would be introduced",
      Code.FailedPrecondition,
      undefined,
      [
        {
          desc: ErrorDetailSchema,
          value: {
            code: DomainErrorCode.CYCLE,
            path: [blocker.id, blocked.id, blocker.id],
          },
        },
      ],
    );
    render(
      <FeatureGraph
        tasks={[blocker, blocked]}
        dependencies={[]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={vi.fn()}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent(
      "This dependency would create a cycle: Blocker task → Blocked task → Blocker task",
    );
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
