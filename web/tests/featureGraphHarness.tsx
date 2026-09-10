import { useContext, useEffect, type ReactNode } from "react";
import { vi } from "vitest";
import { HoveredEdgeContext } from "../src/views/dependencyGraph";

// FeatureGraph のテストは React Flow ごと差し替えるので、モックの束はここに置く。
// 差し替えたキャンバスが受け取ったハンドラは graphMocks 経由でテストから呼ぶ。
export interface ConnectionStateStub {
  isValid: boolean | null;
  fromNode?: { id: string } | null;
  fromHandle?: { type: string } | null;
  toNode?: { id: string } | null;
}

export const graphMocks = {
  addDependencyApi: vi.fn().mockResolvedValue({}),
  removeDependencyApi: vi.fn().mockResolvedValue({}),
  addDependency: {
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
  removeDependency: {
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
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
  onEdgeMouseEnter: undefined as
    ((event: unknown, edge: { id: string }) => void) | undefined,
  onEdgeMouseLeave: undefined as
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
};

export const apiMock = () => ({
  mutations: {
    addDependency: graphMocks.addDependencyApi,
    removeDependency: graphMocks.removeDependencyApi,
  },
});

export const xyflowMock = () => ({
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
    onEdgeMouseEnter,
    onEdgeMouseLeave,
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
    onEdgeMouseEnter?: (event: unknown, edge: { id: string }) => void;
    onEdgeMouseLeave?: (event: unknown, edge: { id: string }) => void;
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
    graphMocks.onEdgeMouseEnter = onEdgeMouseEnter;
    graphMocks.onEdgeMouseLeave = onEdgeMouseLeave;
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
        data-hovered-edge={useContext(HoveredEdgeContext) ?? ""}
        data-nodes-connectable={String(nodesConnectable)}
        onClick={() => onMoveEnd?.({}, { zoom: 1.25 })}
      >
        {children}
      </button>
    );
  },
});

export const settingsMock = () => ({
  maxGraphZoom: 1.7,
  minGraphZoom: 0.08,
  readGraphZoom: vi.fn(() => 1),
  writeGraphZoom: graphMocks.writeGraphZoom,
});

export const graphLayoutMock = () => ({
  useGraphLayout: graphMocks.useGraphLayout,
});

export const hooksMock = () => ({
  useDomainMutation: (mutationFn: (input: unknown) => unknown) => {
    const mutation =
      graphMocks.domainMutationCall++ % 2 === 0
        ? graphMocks.addDependency
        : graphMocks.removeDependency;
    mutation.mutate.mockImplementation((input: unknown) => mutationFn(input));
    mutation.mutateAsync.mockImplementation((input: unknown) =>
      Promise.resolve(mutationFn(input)),
    );
    return mutation;
  },
});

export function resetGraphMocks() {
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
  graphMocks.onEdgeMouseEnter = undefined;
  graphMocks.onEdgeMouseLeave = undefined;
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
}
