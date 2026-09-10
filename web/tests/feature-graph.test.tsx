import { Code, ConnectError } from "@connectrpc/connect";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { useEffect, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DomainErrorCode, ErrorDetailSchema } from "../src/gen/prx/v1/prx_pb";
import { FeatureGraph } from "../src/views/FeatureGraph";
import { makeDependency, makeTask } from "./factories";

interface ConnectionStateStub {
  isValid: boolean | null;
  fromNode?: { id: string } | null;
  fromHandle?: { type: string } | null;
  toNode?: { id: string } | null;
}

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
  onConnectStart: undefined as (() => void) | undefined,
  onConnectEnd: undefined as
    | ((event: unknown, connectionState: ConnectionStateStub) => void)
    | undefined,
  onEdgesChange: undefined as ((changes: unknown[]) => void) | undefined,
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
  hideAttribution: undefined as boolean | undefined,
  // onInit で受け取る React Flow のインスタンス。空白判定は座標をこの stub で
  // flow 座標へ変換し、getNodes が返す矩形と突き合わせる。
  flow: {
    screenToFlowPosition: vi.fn((point: { x: number; y: number }) => point),
    getNodes: vi.fn((): unknown[] => []),
    getNodesBounds: vi.fn(() => ({ x: 0, y: 0, width: 0, height: 0 })),
    setCenter: vi.fn(),
  },
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
    onConnectStart,
    onConnectEnd,
    onEdgesChange,
    onEdgesDelete,
    onEdgeClick,
    onPaneClick,
    onReconnect,
    onReconnectStart,
    onReconnectEnd,
    onInit,
    nodesConnectable,
    proOptions,
  }: {
    children?: ReactNode;
    edges?: unknown[];
    onMoveEnd?: (...args: unknown[]) => void;
    onConnect?: (connection: unknown) => void;
    onConnectStart?: () => void;
    onConnectEnd?: (
      event: unknown,
      connectionState: ConnectionStateStub,
    ) => void;
    onEdgesChange?: (changes: unknown[]) => void;
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
    onInit?: (instance: unknown) => void;
    nodesConnectable?: boolean;
    proOptions?: { hideAttribution?: boolean };
  }) => {
    graphMocks.edges = (edges ?? []) as Record<string, unknown>[];
    graphMocks.onConnect = onConnect;
    graphMocks.onConnectStart = onConnectStart;
    graphMocks.onConnectEnd = onConnectEnd;
    graphMocks.onEdgesChange = onEdgesChange;
    graphMocks.onEdgesDelete = onEdgesDelete;
    graphMocks.onEdgeClick = onEdgeClick;
    graphMocks.onPaneClick = onPaneClick;
    graphMocks.onReconnect = onReconnect;
    graphMocks.onReconnectStart = onReconnectStart;
    graphMocks.onReconnectEnd = onReconnectEnd;
    graphMocks.hideAttribution = proOptions?.hideAttribution;
    useEffect(() => {
      onInit?.(graphMocks.flow);
    }, [onInit]);
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
    // onInit 後の setCenter が参照する。jsdom は matchMedia を持たない。
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches: true }));
    graphMocks.addDependency.isPending = false;
    graphMocks.addDependency.error = null;
    graphMocks.removeDependency.isPending = false;
    graphMocks.removeDependency.error = null;
    graphMocks.domainMutationCall = 0;
    graphMocks.onConnect = undefined;
    graphMocks.onConnectStart = undefined;
    graphMocks.onConnectEnd = undefined;
    graphMocks.onEdgesChange = undefined;
    graphMocks.onEdgesDelete = undefined;
    graphMocks.onEdgeClick = undefined;
    graphMocks.onPaneClick = undefined;
    graphMocks.onReconnect = undefined;
    graphMocks.onReconnectStart = undefined;
    graphMocks.onReconnectEnd = undefined;
    graphMocks.hideAttribution = undefined;
    graphMocks.flow.getNodes.mockReturnValue([]);
    graphMocks.useGraphLayout.mockReturnValue({
      edgeRoutes: new Map(),
      nodes: [],
      layoutError: undefined,
      retryLayout: vi.fn(),
    });
  });

  it("hides the React Flow attribution", () => {
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

    expect(graphMocks.hideAttribution).toBe(true);
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

  it.each([
    ["source", "blockedBy"],
    ["target", "blocks"],
  ] as const)(
    "opens task creation from a %s handle dropped on empty space",
    (handleType, direction) => {
      const onCreateTask = vi.fn();
      render(
        <FeatureGraph
          tasks={[makeTask({ id: "task-1", title: "Blocker task" })]}
          dependencies={[]}
          pullRequests={new Map()}
          documentsByTask={new Map()}
          onEditTask={vi.fn()}
          onPreviewDocument={vi.fn()}
          onCreateTask={onCreateTask}
        />,
      );

      if (!graphMocks.onConnectStart || !graphMocks.onConnectEnd) {
        throw new Error("connection handlers missing");
      }
      act(() => {
        graphMocks.onConnectStart?.();
      });
      expect(
        screen.getByText(
          "Drop in empty space to create a new task with this dependency.",
        ),
      ).toBeInTheDocument();

      act(() => {
        graphMocks.onConnectEnd?.(
          {},
          {
            isValid: null,
            fromNode: { id: "task-1" },
            fromHandle: { type: handleType },
            toNode: null,
          },
        );
      });

      expect(onCreateTask).toHaveBeenCalledWith({
        taskId: "task-1",
        direction,
      });
      expect(graphMocks.addDependency.mutate).not.toHaveBeenCalled();
    },
  );

  const rejectedDrops: [string, ConnectionStateStub][] = [
    [
      "a completed connection",
      {
        isValid: true,
        fromNode: { id: "task-1" },
        fromHandle: { type: "source" },
        toNode: { id: "task-2" },
      },
    ],
    [
      "a drop on an existing node",
      {
        isValid: null,
        fromNode: { id: "task-1" },
        fromHandle: { type: "source" },
        toNode: { id: "task-2" },
      },
    ],
    [
      "a drag without an origin handle",
      {
        isValid: null,
        fromNode: { id: "task-1" },
        fromHandle: null,
        toNode: null,
      },
    ],
    [
      "a drag without an origin node",
      {
        isValid: null,
        fromNode: null,
        fromHandle: { type: "source" },
        toNode: null,
      },
    ],
  ];

  it.each(rejectedDrops)(
    "keeps task creation closed after %s",
    (_name, connectionState) => {
      const onCreateTask = vi.fn();
      render(
        <FeatureGraph
          tasks={[makeTask({ id: "task-1" })]}
          dependencies={[]}
          pullRequests={new Map()}
          documentsByTask={new Map()}
          onEditTask={vi.fn()}
          onPreviewDocument={vi.fn()}
          onCreateTask={onCreateTask}
        />,
      );

      act(() => {
        graphMocks.onConnectEnd?.({}, connectionState);
      });

      expect(onCreateTask).not.toHaveBeenCalled();
    },
  );

  it("keeps task creation closed when the drop lands on a node body", () => {
    const onCreateTask = vi.fn();
    graphMocks.flow.getNodes.mockReturnValue([
      {
        id: "task-2",
        position: { x: 0, y: 0 },
        measured: { width: 284, height: 120 },
      },
    ]);
    render(
      <FeatureGraph
        tasks={[makeTask({ id: "task-1" }), makeTask({ id: "task-2" })]}
        dependencies={[]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={onCreateTask}
      />,
    );

    if (!graphMocks.onConnectEnd) {
      throw new Error("connection handlers missing");
    }
    act(() => {
      graphMocks.onConnectEnd?.(
        { clientX: 140, clientY: 60 },
        {
          isValid: null,
          fromNode: { id: "task-1" },
          fromHandle: { type: "source" },
          toNode: null,
        },
      );
    });

    expect(graphMocks.flow.screenToFlowPosition).toHaveBeenCalledWith({
      x: 140,
      y: 60,
    });
    expect(onCreateTask).not.toHaveBeenCalled();
  });

  it("detaches an existing edge without opening task creation", () => {
    const dependency = makeDependency({
      blockerTaskId: "blocker",
      blockedTaskId: "blocked",
    });
    const edge = {
      id: "blocker-blocked",
      source: "blocker",
      target: "blocked",
    };
    const onCreateTask = vi.fn();
    render(
      <FeatureGraph
        tasks={[makeTask({ id: "blocker" }), makeTask({ id: "blocked" })]}
        dependencies={[dependency]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={onCreateTask}
      />,
    );

    // 掴み直しでは onConnectEnd が onReconnectEnd より先に呼ばれる。
    act(() => {
      graphMocks.onReconnectStart?.({}, edge, "source");
      graphMocks.onConnectEnd?.(
        {},
        {
          isValid: null,
          fromNode: { id: "blocker" },
          fromHandle: { type: "source" },
          toNode: null,
        },
      );
      graphMocks.onReconnectEnd?.({}, edge, "source", { isValid: null });
    });

    expect(onCreateTask).not.toHaveBeenCalled();
    expect(graphMocks.removeDependency.mutate).toHaveBeenCalledWith({
      blocker: "blocker",
      blocked: "blocked",
    });
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
        document.querySelector(".graph-status-notice"),
      ).toBeInTheDocument();
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

  it("uses routed endpoint ports and removes the selected edge by keyboard", async () => {
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

    // ポートの計測はノード確定の 1 フレーム後なので、初回描画では React Flow
    // がまだ知らない handle を指定してはいけない。
    expect(graphMocks.edges[0]).toMatchObject({
      id: "blocker-blocked",
      sourceHandle: null,
      targetHandle: null,
    });
    await waitFor(() => {
      expect(graphMocks.edges[0]?.["sourceHandle"]).toBe(
        "blocker-blocked-source",
      );
    });
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

  it("selects an edge from a React Flow selection change and deletes it", () => {
    const dependency = makeDependency({
      blockerTaskId: "blocker",
      blockedTaskId: "blocked",
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

    act(() => {
      graphMocks.onEdgesChange?.([
        { type: "select", id: "blocker-blocked", selected: true },
      ]);
    });
    expect(graphMocks.edges[0]?.["selected"]).toBe(true);
    act(() => {
      graphMocks.onEdgesDelete?.([{ source: "blocker", target: "blocked" }]);
    });
    expect(graphMocks.removeDependency.mutate).toHaveBeenCalledWith({
      blocker: "blocker",
      blocked: "blocked",
    });
    act(() => {
      graphMocks.onEdgesChange?.([
        { type: "select", id: "blocker-blocked", selected: false },
      ]);
    });
    expect(graphMocks.edges[0]?.["selected"]).toBe(false);
  });

  it("does not reselect a dependency that is removed and added again", () => {
    const dependency = makeDependency({
      blockerTaskId: "blocker",
      blockedTaskId: "blocked",
    });
    const tasks = [
      makeTask({ id: "blocker", title: "Blocker task" }),
      makeTask({ id: "blocked", title: "Blocked task" }),
    ];
    const graph = (dependencies: (typeof dependency)[]) => (
      <FeatureGraph
        tasks={tasks}
        dependencies={dependencies}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={vi.fn()}
      />
    );
    const { rerender } = render(graph([dependency]));

    act(() => {
      graphMocks.onEdgeClick?.({}, { id: "blocker-blocked" });
    });
    expect(graphMocks.edges[0]?.["selected"]).toBe(true);
    rerender(graph([]));
    rerender(graph([dependency]));
    expect(graphMocks.edges[0]?.["selected"]).toBe(false);
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

  it("explains an empty canvas that hiding produced instead of offering a first task", () => {
    const onCreateTask = vi.fn();
    render(
      <FeatureGraph
        tasks={[]}
        dependencies={[]}
        pullRequests={new Map()}
        documentsByTask={new Map()}
        hiddenTaskCount={3}
        onEditTask={vi.fn()}
        onPreviewDocument={vi.fn()}
        onCreateTask={onCreateTask}
      />,
    );

    expect(
      screen.getByRole("heading", { name: "Every task is completed" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "3 completed tasks are hidden. Turn off Hide completed to see them.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Add task" }),
    ).not.toBeInTheDocument();
  });

  it("keeps an archived empty graph read-only", () => {
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
      graphMocks.onConnectStart?.();
      graphMocks.onConnectEnd?.(
        {},
        {
          isValid: null,
          fromNode: { id: "task-1" },
          fromHandle: { type: "source" },
          toNode: null,
        },
      );
    });
    expect(graphMocks.addDependency.mutate).not.toHaveBeenCalled();
    expect(graphMocks.removeDependency.mutate).not.toHaveBeenCalled();
    expect(onCreateTask).not.toHaveBeenCalled();
    expect(
      document.querySelector(".graph-status-notice"),
    ).not.toBeInTheDocument();
  });

  it("builds dependency edges, persists zoom, and retries a failed layout", () => {
    const retryLayout = vi.fn();
    graphMocks.useGraphLayout.mockReturnValue({
      edgeRoutes: new Map(),
      nodes: [makeTask()],
      layoutError: { message: undefined },
      retryLayout,
    });
    const { container } = render(
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

    expect(
      container.querySelector(".graph-status-notice"),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("1 nodes · 1 links")).not.toBeInTheDocument();
    for (const label of ["Ready", "Review", "Conflict", "Merged"]) {
      expect(screen.queryByText(label)).not.toBeInTheDocument();
    }
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
