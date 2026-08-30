import { Code, ConnectError } from "@connectrpc/connect";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DomainErrorCode, ErrorDetailSchema } from "../src/gen/prx/v1/prx_pb";
import { FeatureGraph } from "../src/views/FeatureGraph";
import { makeDependency, makeTask } from "./factories";

const graphMocks = vi.hoisted(() => ({
  addDependencyApi: vi.fn().mockResolvedValue({}),
  removeDependencyApi: vi.fn().mockResolvedValue({}),
  addDependency: {
    mutate: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
  removeDependency: {
    mutate: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
  domainMutationCall: 0,
  edges: [] as Record<string, unknown>[],
  useGraphLayout: vi.fn(),
  writeGraphZoom: vi.fn(),
  onConnect: undefined as ((connection: unknown) => void) | undefined,
  onEdgesDelete: undefined as ((edges: unknown[]) => void) | undefined,
  onEdgeClick: undefined as
    ((event: unknown, edge: { id: string }) => void) | undefined,
  onPaneClick: undefined as (() => void) | undefined,
  onReconnect: undefined as (() => void) | undefined,
  onReconnectStart: undefined as
    ((event: unknown, edge: unknown, handleType: string) => void) | undefined,
  onReconnectEnd: undefined as
    | ((
        event: unknown,
        edge: unknown,
        handleType: string,
        connectionState: { isValid: boolean | null },
      ) => void)
    | undefined,
}));

vi.mock("../src/api", () => ({
  mutations: {
    addDependency: graphMocks.addDependencyApi,
    removeDependency: graphMocks.removeDependencyApi,
  },
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
    onEdgesDelete,
    onEdgeClick,
    onPaneClick,
    onReconnect,
    onReconnectStart,
    onReconnectEnd,
    nodesConnectable,
  }: {
    children?: ReactNode;
    edges?: unknown[];
    onMoveEnd?: (...args: unknown[]) => void;
    onConnect?: (connection: unknown) => void;
    onEdgesDelete?: (edges: unknown[]) => void;
    onEdgeClick?: (event: unknown, edge: { id: string }) => void;
    onPaneClick?: () => void;
    onReconnect?: () => void;
    onReconnectStart?: (
      event: unknown,
      edge: unknown,
      handleType: string,
    ) => void;
    onReconnectEnd?: (
      event: unknown,
      edge: unknown,
      handleType: string,
      connectionState: { isValid: boolean | null },
    ) => void;
    nodesConnectable?: boolean;
  }) => {
    graphMocks.edges = (edges ?? []) as Record<string, unknown>[];
    graphMocks.onConnect = onConnect;
    graphMocks.onEdgesDelete = onEdgesDelete;
    graphMocks.onEdgeClick = onEdgeClick;
    graphMocks.onPaneClick = onPaneClick;
    graphMocks.onReconnect = onReconnect;
    graphMocks.onReconnectStart = onReconnectStart;
    graphMocks.onReconnectEnd = onReconnectEnd;
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
    const mutation =
      graphMocks.domainMutationCall++ % 2 === 0
        ? graphMocks.addDependency
        : graphMocks.removeDependency;
    mutation.mutate.mockImplementation((input: unknown) => mutationFn(input));
    return mutation;
  },
}));

describe("FeatureGraph", () => {
  afterEach(cleanup);

  beforeEach(() => {
    vi.clearAllMocks();
    graphMocks.addDependency.isPending = false;
    graphMocks.addDependency.error = null;
    graphMocks.removeDependency.isPending = false;
    graphMocks.removeDependency.error = null;
    graphMocks.domainMutationCall = 0;
    graphMocks.onConnect = undefined;
    graphMocks.onEdgesDelete = undefined;
    graphMocks.onEdgeClick = undefined;
    graphMocks.onPaneClick = undefined;
    graphMocks.onReconnect = undefined;
    graphMocks.onReconnectStart = undefined;
    graphMocks.onReconnectEnd = undefined;
    graphMocks.useGraphLayout.mockReturnValue({
      edgeRoutes: new Map(),
      nodes: [],
      layoutError: undefined,
      retryLayout: vi.fn(),
    });
  });

  it("sends a blocker-to-blocked connection without adding an optimistic edge", () => {
    const blocker = makeTask({ id: "blocker", title: "Blocker task" });
    const blocked = makeTask({ id: "blocked", title: "Blocked task" });
    graphMocks.useGraphLayout.mockReturnValue({
      edgeRoutes: new Map(),
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

  it.each(["source", "target"] as const)(
    "removes a dependency dropped from its %s end into empty space",
    (handleType) => {
      const dependency = makeDependency({
        blockerTaskId: "blocker",
        blockedTaskId: "blocked",
      });
      const edge = {
        id: "blocker-blocked",
        source: "blocker",
        target: "blocked",
      };
      render(
        <FeatureGraph
          tasks={[
            makeTask({ id: "blocker", title: "Blocker task" }),
            makeTask({ id: "blocked", title: "Blocked task" }),
          ]}
          dependencies={[dependency]}
          pullRequests={new Map()}
          documentsByTask={new Map()}
          onEditTask={vi.fn()}
          onPreviewDocument={vi.fn()}
          onCreateTask={vi.fn()}
        />,
      );

      if (!graphMocks.onReconnectStart || !graphMocks.onReconnectEnd) {
        throw new Error("reconnect handlers missing");
      }
      act(() => {
        graphMocks.onReconnectStart?.({}, edge, handleType);
      });
      expect(
        screen.getByText(
          "Drop in empty space to remove Blocker task → Blocked task.",
        ),
      ).toBeInTheDocument();

      act(() => {
        graphMocks.onReconnectEnd?.({}, edge, handleType, {
          isValid: null,
        });
      });

      expect(graphMocks.removeDependency.mutate).toHaveBeenCalledWith({
        blocker: "blocker",
        blocked: "blocked",
      });
      expect(graphMocks.removeDependencyApi).toHaveBeenCalledWith(
        "blocker",
        "blocked",
      );
    },
  );

  it("keeps a dependency when its source is dropped on a valid handle", () => {
    const dependency = makeDependency();
    const edge = {
      id: "task-1-task-2",
      source: dependency.blockerTaskId,
      target: dependency.blockedTaskId,
    };
    render(
      <FeatureGraph
        tasks={[makeTask()]}
        dependencies={[dependency]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={vi.fn()}
      />,
    );

    if (!graphMocks.onReconnect || !graphMocks.onReconnectEnd) {
      throw new Error("reconnect handlers missing");
    }
    act(() => {
      graphMocks.onReconnect?.();
      graphMocks.onReconnectEnd?.({}, edge, "target", {
        isValid: true,
      });
    });

    expect(graphMocks.removeDependency.mutate).not.toHaveBeenCalled();
  });

  it("uses routed endpoint ports and removes the selected edge by keyboard", () => {
    const dependency = makeDependency({
      blockerTaskId: "blocker",
      blockedTaskId: "blocked",
    });
    graphMocks.useGraphLayout.mockReturnValue({
      edgeRoutes: new Map([
        [
          "blocker-blocked",
          {
            points: [
              { x: 284, y: 72 },
              { x: 394, y: 72 },
            ],
            sourcePortId: "blocker-blocked-source",
            sourcePortTop: 72,
            targetPortId: "blocker-blocked-target",
            targetPortTop: 72,
          },
        ],
      ]),
      nodes: [],
      layoutError: undefined,
      layoutPending: false,
      retryLayout: vi.fn(),
    });
    render(
      <FeatureGraph
        tasks={[
          makeTask({ id: "blocker", title: "Blocker task" }),
          makeTask({ id: "blocked", title: "Blocked task" }),
        ]}
        dependencies={[dependency]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={vi.fn()}
      />,
    );

    expect(graphMocks.edges[0]).toMatchObject({
      id: "blocker-blocked",
      sourceHandle: "blocker-blocked-source",
      targetHandle: "blocker-blocked-target",
      type: "dependency",
      reconnectable: true,
      deletable: true,
      data: {
        label: "Blocker task → Blocked task",
        removeLabel: "Remove dependency Blocker task → Blocked task",
      },
      selected: false,
    });
    act(() => {
      graphMocks.onEdgeClick?.({}, { id: "blocker-blocked" });
    });
    expect(graphMocks.edges[0]?.["selected"]).toBe(true);
    act(() => {
      graphMocks.onPaneClick?.();
    });
    expect(graphMocks.edges[0]?.["selected"]).toBe(false);
    act(() => {
      const data = graphMocks.edges[0]?.["data"] as
        { onRemove: () => void } | undefined;
      data?.onRemove();
    });
    expect(graphMocks.removeDependency.mutate).toHaveBeenCalledWith({
      blocker: "blocker",
      blocked: "blocked",
    });
    graphMocks.removeDependency.mutate.mockClear();
    act(() => {
      graphMocks.onEdgesDelete?.([{ source: "blocker", target: "blocked" }]);
    });
    expect(graphMocks.removeDependency.mutate).toHaveBeenCalledWith({
      blocker: "blocker",
      blocked: "blocked",
    });
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

  it("keeps an archived empty graph read-only", () => {
    render(
      <FeatureGraph
        tasks={[]}
        dependencies={[]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={vi.fn()}
        readOnly
      />,
    );

    expect(
      screen.getByRole("heading", { name: "Draw the first node" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Add task" }),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("mock-react-flow")).toHaveAttribute(
      "data-nodes-connectable",
      "false",
    );
    expect(
      screen.queryByText(/Add: drag right output/),
    ).not.toBeInTheDocument();
    expect(graphMocks.useGraphLayout).toHaveBeenCalledWith(
      expect.objectContaining({ readOnly: true }),
    );

    act(() => {
      graphMocks.onConnect?.({ source: "task-1", target: "task-2" });
      graphMocks.onReconnectStart?.(
        {},
        { source: "task-1", target: "task-2" },
        "source",
      );
      graphMocks.onReconnectEnd?.(
        {},
        { source: "task-1", target: "task-2" },
        "source",
        { isValid: null },
      );
    });
    expect(graphMocks.addDependency.mutate).not.toHaveBeenCalled();
    expect(graphMocks.removeDependency.mutate).not.toHaveBeenCalled();
  });

  it("builds dependency edges, persists zoom, and retries a failed layout", () => {
    const retryLayout = vi.fn();
    graphMocks.useGraphLayout.mockReturnValue({
      edgeRoutes: new Map(),
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
