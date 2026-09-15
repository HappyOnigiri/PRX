import type { ELK as ElkInstance, ElkNode } from "elkjs/lib/elk-api.js";
import ELK from "elkjs/lib/elk-api.js";
import elkWorkerUrl from "elkjs/lib/elk-worker.min.js?url";
import { useEffect, useMemo, useState } from "react";
import type {
  Dependency,
  PullRequest,
  Task,
  TaskLabelAppearances,
} from "../gen/prx/v1/prx_pb";
import { isDependencyBlockedTask, isDormantTask } from "../task-attention";
import { dependencyEdgeId, type DependencyEdgeRoute } from "./dependencyGraph";
import { type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";
import {
  emptyHiddenDependencies,
  type HiddenDependencies,
} from "./visibleGraph";

interface GraphLayoutOptions {
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  hiddenDependencies?: Map<string, HiddenDependencies>;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onAddDocument?: (taskId: string, trigger: HTMLButtonElement) => void;
  taskLabelAppearances?: TaskLabelAppearances | undefined;
  readOnly?: boolean;
}

interface LayoutRequest extends GraphLayoutOptions {
  attempt: number;
}

// ノードの寸法はフィーチャー内で揃える。中身に合わせて 1 件ずつ高さを変える
// と、ELK はそのノードの周囲まで座標を動かし、ステータスが変わるだけでグラフ
// が動く。docs/design/webui.md を参照。
const nodeWidth = 284;
// ヘッダ・ステータス行・タイトル・担当者に、ブロックラベルが 2 行目へ折り返す
// ぶんを足した高さ。ラベルの数は同期で増減するので常に 2 行ぶんを確保する。
const nodeBaseHeight = 196;
const assetRowHeight = 34;
// これより多いアセットはノードの中でスクロールさせる。
const maxAssetRows = 4;

// 高さはフィーチャーで最も嵩むタスクに合わせる。全件を最大寸法にすると、
// アセットの少ないフィーチャーで必要以上に大きなカードになる。
function nodeHeightFor({
  tasks,
  pullRequests,
  documentsByTask,
  readOnly,
}: {
  tasks: Task[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  readOnly: boolean;
}) {
  const assets = Math.max(
    0,
    ...tasks.map(
      (task) =>
        (documentsByTask.get(task.id)?.length ?? 0) +
        (pullRequests.has(task.id) ? 1 : 0),
    ),
  );
  // 参照の追加ボタンは読み取り専用でなければどのノードにも並ぶ。
  const rows = Math.min(assets + (readOnly ? 0 : 1), maxAssetRows);
  return nodeBaseHeight + rows * assetRowHeight;
}

// タスク ID は T-<連番> なので、素の文字列比較では T-10 が T-9 より前に来る。
// ロケールを固定し、実行環境で並びが変わらないようにする。
const taskIdOrder = new Intl.Collator("en", { numeric: true });

function isSameLayoutRequest(
  completed: LayoutRequest | undefined,
  requested: LayoutRequest,
) {
  return (
    completed?.tasks === requested.tasks &&
    completed.dependencies === requested.dependencies &&
    completed.pullRequests === requested.pullRequests &&
    completed.documentsByTask === requested.documentsByTask &&
    completed.hiddenDependencies === requested.hiddenDependencies &&
    completed.onEditTask === requested.onEditTask &&
    completed.onPreviewDocument === requested.onPreviewDocument &&
    completed.onAddDocument === requested.onAddDocument &&
    completed.taskLabelAppearances === requested.taskLabelAppearances &&
    completed.readOnly === requested.readOnly &&
    completed.attempt === requested.attempt
  );
}

function buildRawNodes({
  tasks,
  pullRequests,
  documentsByTask,
  hiddenDependencies = emptyHiddenDependencies,
  onEditTask,
  onPreviewDocument,
  onAddDocument,
  taskLabelAppearances,
  readOnly = false,
}: GraphLayoutOptions) {
  const omitOwner = hasSingleOwner(pullRequests);
  const nodeHeight = nodeHeightFor({
    tasks,
    pullRequests,
    documentsByTask,
    readOnly,
  });
  // tasks はインスペクタや件数表示と同じ配列なので、複製してから並べ替える。
  return [...tasks]
    .sort((left, right) => taskIdOrder.compare(left.id, right.id))
    .map((task) => {
      const pr = pullRequests.get(task.id);
      const documents = documentsByTask.get(task.id) ?? [];
      const syncError = pr?.syncError ?? "";
      return {
        id: task.id,
        width: nodeWidth,
        height: nodeHeight,
        data: {
          title: task.title,
          assignee: task.assignee,
          state: task.displayState,
          ...(taskLabelAppearances ? { taskLabelAppearances } : {}),
          dormant: isDormantTask(task),
          blocked: isDependencyBlockedTask(task),
          blockLabels: task.blockLabels,
          hasImplementationPlan: task.hasImplementationPlan,
          stale: pr?.stale ?? false,
          syncError,
          pullRequest: pr
            ? { label: pullRequestLabel(pr, omitOwner), url: pr.url }
            : undefined,
          documents,
          ...hiddenDependencyData(hiddenDependencies.get(task.id)),
          readOnly,
          onEdit: () => {
            onEditTask(task.id);
          },
          onPreview: onPreviewDocument,
          onAddReference: (trigger: HTMLButtonElement) => {
            onAddDocument?.(task.id, trigger);
          },
        },
      };
    });
}

// フィーチャーの pull request がすべて同じ host と owner にあるなら、その部分は
// どの pull request かを分けない。ノードは幅が狭いので落とし、番号の側を残す。
function hasSingleOwner(pullRequests: Map<string, PullRequest>) {
  // 判定はフィーチャーの全 pull request で行う。完了済みを隠しても名前が
  // 変わらないようにするためである。
  const owners = new Set(
    [...pullRequests.values()].map((pr) => `${pr.host}/${pr.owner}`),
  );
  return owners.size === 1;
}

function pullRequestLabel(pr: PullRequest, omitOwner: boolean) {
  if (omitOwner) return `${pr.repository} #${String(pr.number)}`;
  const host = pr.host && pr.host !== "github.com" ? `${pr.host}/` : "";
  return `${host}${pr.owner}/${pr.repository} #${String(pr.number)}`;
}

// 背後に隠れているものがないタスクは、undefined を持たせずキー自体を省く。
// ノードのデータ型が undefined を受け付けないため。
function hiddenDependencyData(hidden: HiddenDependencies | undefined) {
  return hidden ? { hiddenDependencies: hidden } : {};
}

type RawNode = ReturnType<typeof buildRawNodes>[number];

// considerModelOrder は children だけでなくエッジの入力順も見る。サーバーが
// 返す依存の並びは created_at が同値だと揺れるので、ここで ID 順に揃える。
function orderedDependencies(dependencies: Dependency[]) {
  return [...dependencies].sort(
    (left, right) =>
      taskIdOrder.compare(left.blockerTaskId, right.blockerTaskId) ||
      taskIdOrder.compare(left.blockedTaskId, right.blockedTaskId),
  );
}

function buildLayoutGraph(raw: RawNode[], dependencies: Dependency[]): ElkNode {
  return {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.edgeRouting": "ORTHOGONAL",
      "elk.layered.mergeEdges": "false",
      // 同じ列の縦並びを入力順に合わせ、依存が許す範囲でタスク ID 順にする。
      "elk.layered.considerModelOrder.strategy": "NODES_AND_EDGES",
      "elk.spacing.componentComponent": "120",
      "elk.spacing.nodeNode": "72",
      "elk.layered.spacing.nodeNodeBetweenLayers": "110",
    },
    children: raw.map(({ id, width, height }) => ({ id, width, height })),
    edges: orderedDependencies(dependencies).map((dependency) => ({
      id: dependencyEdgeId(dependency.blockerTaskId, dependency.blockedTaskId),
      sources: [dependency.blockerTaskId],
      targets: [dependency.blockedTaskId],
    })),
  };
}

function readLayout(
  layout: ElkNode,
  raw: RawNode[],
  dependencies: Dependency[],
) {
  const positions = new Map(
    layout.children?.map((child) => [
      child.id,
      { x: child.x ?? 0, y: child.y ?? 0 },
    ]),
  );
  const ports = new Map<
    string,
    {
      incoming: { id: string; top: number }[];
      outgoing: { id: string; top: number }[];
    }
  >(raw.map((node) => [node.id, { incoming: [], outgoing: [] }]));
  const edgeRoutes = new Map<string, DependencyEdgeRoute>();

  for (const dependency of dependencies) {
    const id = dependencyEdgeId(
      dependency.blockerTaskId,
      dependency.blockedTaskId,
    );
    const section = layout.edges?.find((edge) => edge.id === id)?.sections?.[0];
    const sourcePosition = positions.get(dependency.blockerTaskId);
    const targetPosition = positions.get(dependency.blockedTaskId);
    if (!section || !sourcePosition || !targetPosition) continue;

    const sourcePortId = `${id}-source`;
    const targetPortId = `${id}-target`;
    const route = {
      points: [
        section.startPoint,
        ...(section.bendPoints ?? []),
        section.endPoint,
      ],
      sourcePortId,
      sourcePortTop: section.startPoint.y - sourcePosition.y,
      targetPortId,
      targetPortTop: section.endPoint.y - targetPosition.y,
    };
    edgeRoutes.set(id, route);
    ports.get(dependency.blockerTaskId)?.outgoing.push({
      id: sourcePortId,
      top: route.sourcePortTop,
    });
    ports.get(dependency.blockedTaskId)?.incoming.push({
      id: targetPortId,
      top: route.targetPortTop,
    });
  }

  const nodes: TaskFlowNode[] = raw.map((node) => ({
    ...node,
    data: {
      ...node.data,
      incomingPorts: ports.get(node.id)?.incoming ?? [],
      outgoingPorts: ports.get(node.id)?.outgoing ?? [],
    },
    type: "task",
    position: positions.get(node.id) ?? { x: 0, y: 0 },
  }));
  return { edgeRoutes, nodes };
}

export function useGraphLayout({
  tasks,
  dependencies,
  pullRequests,
  documentsByTask,
  hiddenDependencies = emptyHiddenDependencies,
  onEditTask,
  onPreviewDocument,
  onAddDocument,
  taskLabelAppearances,
  readOnly = false,
}: GraphLayoutOptions) {
  const [nodes, setNodes] = useState<TaskFlowNode[]>([]);
  const [edgeRoutes, setEdgeRoutes] = useState<
    Map<string, DependencyEdgeRoute>
  >(() => new Map());
  // 表示言語を変えてもレイアウトの effect が再実行されてビューポートが
  // リセットされないよう、生のエラーを保持する。
  const [layoutError, setLayoutError] = useState<
    { message: string | undefined } | undefined
  >();
  const [layoutAttempt, setLayoutAttempt] = useState(0);
  const [completedLayout, setCompletedLayout] = useState<LayoutRequest>();
  const layoutRequest = useMemo(
    () => ({
      tasks,
      dependencies,
      pullRequests,
      documentsByTask,
      hiddenDependencies,
      onEditTask,
      onPreviewDocument,
      ...(onAddDocument ? { onAddDocument } : {}),
      ...(taskLabelAppearances ? { taskLabelAppearances } : {}),
      readOnly,
      attempt: layoutAttempt,
    }),
    [
      tasks,
      dependencies,
      pullRequests,
      documentsByTask,
      hiddenDependencies,
      onEditTask,
      onPreviewDocument,
      onAddDocument,
      taskLabelAppearances,
      readOnly,
      layoutAttempt,
    ],
  );
  const layoutPending = !isSameLayoutRequest(completedLayout, layoutRequest);

  useEffect(() => {
    let current = true;
    const raw = buildRawNodes(layoutRequest);

    function failLayout(message: string | undefined) {
      if (!current) return;
      setEdgeRoutes(new Map());
      setLayoutError({ message });
      setCompletedLayout(layoutRequest);
    }

    // 起動に失敗した worker は応答せず、ELK はレイアウトの promise を保留した
    // ままにする。worker を自前で持ち、失敗イベントをグラフへ届けることで、
    // 読み込み中のまま止まるのを防ぐ。
    function handleWorkerFailure(event: Event) {
      failLayout(
        event instanceof ErrorEvent && event.message
          ? event.message
          : undefined,
      );
    }

    let worker: Worker | undefined;
    let elk: ElkInstance | undefined;
    try {
      elk = new ELK({
        workerFactory: () => {
          const created = new Worker(elkWorkerUrl);
          created.addEventListener("error", handleWorkerFailure);
          created.addEventListener("messageerror", handleWorkerFailure);
          worker = created;
          return created;
        },
      });
    } catch (error) {
      failLayout(error instanceof Error ? error.message : undefined);
    }
    elk
      ?.layout(buildLayoutGraph(raw, layoutRequest.dependencies))
      .then((layout) => {
        if (!current) return;
        setLayoutError(undefined);
        const result = readLayout(layout, raw, layoutRequest.dependencies);
        setEdgeRoutes(result.edgeRoutes);
        setNodes(result.nodes);
        setCompletedLayout(layoutRequest);
      })
      .catch((error: unknown) => {
        failLayout(error instanceof Error ? error.message : undefined);
      });
    return () => {
      current = false;
      worker?.removeEventListener("error", handleWorkerFailure);
      worker?.removeEventListener("messageerror", handleWorkerFailure);
      if (elk) elk.terminateWorker();
      else worker?.terminate();
    };
  }, [layoutRequest]);

  function retryLayout() {
    setLayoutError(undefined);
    setLayoutAttempt((attempt) => attempt + 1);
  }

  return { edgeRoutes, nodes, layoutError, layoutPending, retryLayout };
}
