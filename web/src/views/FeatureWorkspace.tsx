import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import {
  Background,
  BackgroundVariant,
  Controls,
  MarkerType,
  ReactFlow,
  type Edge,
  type ReactFlowInstance,
} from "@xyflow/react";
import ELK from "elkjs/lib/elk-api.js";
import elkWorkerUrl from "elkjs/lib/elk-worker.min.js?url";
import { mutations } from "../api";
import {
  DocumentKind,
  FeatureStatus,
  PullRequestDisplayState,
  TaskDisplayState,
  TaskKind,
  TaskStatus,
  type BlockedReason,
} from "../gen/prx/v1/prx_pb";
import { useDomainMutation, useSnapshot } from "../hooks";
import {
  blockedReasonLabel,
  documentKindLabel,
  featureStatusLabel,
  formatError,
  pullRequestDisplayStateLabel,
  taskDisplayStateLabel,
  taskDisplayStateToken,
  taskKindLabel,
  taskStatusLabel,
} from "../i18n/domain";
import { TaskNode, type TaskFlowNode } from "./TaskNode";
import { formValue } from "../form";
import {
  maxGraphZoom,
  minGraphZoom,
  readGraphZoom,
  writeGraphZoom,
} from "../i18n/settings";

const nodeTypes = { task: TaskNode };

export function FeatureWorkspace() {
  const { t } = useTranslation();
  const { featureId } = useParams({ from: "/features/$featureId" });
  const navigate = useNavigate();
  const snapshot = useSnapshot();
  const [selected, setSelected] = useState<string>();
  const [nodes, setNodes] = useState<TaskFlowNode[]>([]);
  const [flow, setFlow] = useState<ReactFlowInstance<TaskFlowNode, Edge>>();
  const [initialGraphZoom] = useState(readGraphZoom);
  const graphZoom = useRef(initialGraphZoom);
  const [showTask, setShowTask] = useState(false);
  const [showFeatureEdit, setShowFeatureEdit] = useState(false);
  // The raw message is kept untranslated so that changing the display language
  // does not re-run the layout effect and reset the viewport.
  const [layoutError, setLayoutError] = useState<{ message?: string }>();
  const [layoutAttempt, setLayoutAttempt] = useState(0);
  const data = snapshot.data;
  const feature = data?.features.find((f) => f.id === featureId);
  const tasks = useMemo(
    () => data?.tasks.filter((task) => task.featureId === featureId) ?? [],
    [data, featureId],
  );
  const taskIds = useMemo(() => new Set(tasks.map((task) => task.id)), [tasks]);
  const dependencies = useMemo(
    () =>
      data?.dependencies.filter((dep) => taskIds.has(dep.blockerTaskId)) ?? [],
    [data, taskIds],
  );
  const prs = useMemo(
    () => new Map(data?.pullRequests.map((pr) => [pr.taskId, pr]) ?? []),
    [data],
  );
  const edges: Edge[] = useMemo(
    () =>
      dependencies.map((dep) => ({
        id: `${dep.blockerTaskId}-${dep.blockedTaskId}`,
        source: dep.blockerTaskId,
        target: dep.blockedTaskId,
        type: "smoothstep",
        markerEnd: { type: MarkerType.ArrowClosed },
        className: "dependency-edge",
      })),
    [dependencies],
  );
  const ariaLabelConfig = useMemo(
    () => ({
      "node.a11yDescription.default": t("workspace.flow.nodeDescription"),
      "node.a11yDescription.keyboardDisabled": t(
        "workspace.flow.keyboardDisabled",
      ),
      "edge.a11yDescription.default": t("workspace.flow.edgeDescription"),
      "controls.ariaLabel": t("workspace.flow.controls"),
      "controls.zoomIn.ariaLabel": t("workspace.flow.zoomIn"),
      "controls.zoomOut.ariaLabel": t("workspace.flow.zoomOut"),
      "controls.fitView.ariaLabel": t("workspace.flow.fitView"),
      "handle.ariaLabel": t("workspace.flow.handle"),
    }),
    [t],
  );
  useEffect(() => {
    let current = true;
    const raw = tasks.map((task) => {
      const pr = prs.get(task.id);
      return {
        id: task.id,
        width: 244,
        height: 126,
        data: {
          title: task.title,
          repository: pr ? `${pr.owner}/${pr.repository} #${pr.number}` : "",
          assignee: task.assignee,
          state: task.displayState,
          ready: task.ready,
          stale: pr?.stale ?? false,
        },
      };
    });
    const elk = new ELK({ workerUrl: elkWorkerUrl });
    elk
      .layout({
        id: "root",
        layoutOptions: {
          "elk.algorithm": "layered",
          "elk.direction": "RIGHT",
          "elk.spacing.nodeNode": "72",
          "elk.layered.spacing.nodeNodeBetweenLayers": "110",
        },
        children: raw.map(({ id, width, height }) => ({ id, width, height })),
        edges: dependencies.map((dep, index) => ({
          id: String(index),
          sources: [dep.blockerTaskId],
          targets: [dep.blockedTaskId],
        })),
      })
      .then((layout) => {
        if (!current) return;
        setLayoutError(undefined);
        const positions = new Map(
          layout.children?.map((child) => [
            child.id,
            { x: child.x ?? 0, y: child.y ?? 0 },
          ]),
        );
        setNodes(
          raw.map((node) => ({
            ...node,
            type: "task",
            position: positions.get(node.id) ?? { x: 0, y: 0 },
          })),
        );
      })
      .catch((error: unknown) => {
        if (!current) return;
        setLayoutError({
          message: error instanceof Error ? error.message : undefined,
        });
      });
    return () => {
      current = false;
      elk.terminateWorker();
    };
  }, [tasks, dependencies, prs, layoutAttempt]);
  useEffect(() => {
    if (nodes.length && flow) {
      const bounds = flow.getNodesBounds(nodes);
      void flow.setCenter(
        bounds.x + bounds.width / 2,
        bounds.y + bounds.height / 2,
        {
          zoom: graphZoom.current,
          duration: window.matchMedia("(prefers-reduced-motion: reduce)")
            .matches
            ? 0
            : 260,
        },
      );
    }
  }, [nodes, flow]);
  const createTask = useDomainMutation(mutations.createTask);
  const sync = useDomainMutation((id: string) => mutations.sync(id));
  const updateFeature = useDomainMutation(mutations.updateFeature);
  const deleteFeature = useDomainMutation(mutations.deleteFeature);
  if (snapshot.isPending)
    return (
      <div className="state-message">
        <div className="spinner" />
        <h1>{t("workspace.loading")}</h1>
      </div>
    );
  if (!feature || !data)
    return (
      <div className="state-message">
        <h1>{t("workspace.notFound")}</h1>
        <button onClick={() => void navigate({ to: "/" })}>
          {t("workspace.returnOverview")}
        </button>
      </div>
    );
  async function submitTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    try {
      await createTask.mutateAsync({
        featureId,
        title: formValue(form, "title"),
        scope: formValue(form, "scope"),
        kind: Number(form.get("kind")),
        assignee: formValue(form, "assignee"),
      });
    } catch {
      return;
    }
    setShowTask(false);
  }
  async function submitFeature(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    try {
      await updateFeature.mutateAsync({
        id: featureId,
        slug: formValue(form, "slug"),
        title: formValue(form, "title"),
        description: formValue(form, "description"),
        status: Number(form.get("status")),
      });
    } catch {
      return;
    }
    setShowFeatureEdit(false);
  }
  async function removeFeature() {
    if (
      !window.confirm(
        t("workspace.deleteFeatureConfirm", { title: feature?.title ?? "" }),
      )
    )
      return;
    try {
      await deleteFeature.mutateAsync(featureId);
    } catch {
      return;
    }
    await navigate({ to: "/" });
  }
  const selectedTask = tasks.find((task) => task.id === selected);
  return (
    <div className="workspace">
      <header className="workspace-head">
        <div>
          <p className="eyebrow">
            {t("workspace.eyebrow", { slug: feature.slug })}
          </p>
          <h1>{feature.title}</h1>
          <p>{feature.description || t("workspace.noDescription")}</p>
        </div>
        <div className="workspace-actions">
          <button
            className="secondary"
            onClick={() => sync.mutate(featureId)}
            disabled={sync.isPending}
          >
            {sync.isPending
              ? t("workspace.syncing")
              : t("workspace.syncGithub")}
          </button>
          <button onClick={() => setShowTask(true)}>
            {t("workspace.addTask")}
          </button>
          <button
            className="icon-button"
            aria-label={t("workspace.editFeature")}
            onClick={() => setShowFeatureEdit(true)}
          >
            ✎
          </button>
          <button
            className="icon-button"
            aria-label={
              feature.archived
                ? t("workspace.unarchiveFeature")
                : t("workspace.archiveFeature")
            }
            onClick={() =>
              updateFeature.mutate({
                id: featureId,
                archived: !feature.archived,
              })
            }
          >
            ⌁
          </button>
          <button
            className="icon-button danger"
            aria-label={t("workspace.deleteFeature")}
            onClick={removeFeature}
          >
            ×
          </button>
        </div>
      </header>
      <MutationError error={deleteFeature.error} />
      <div className="graph-legend">
        <span>
          <i className="ready" />
          {t("workspace.legend.ready")}
        </span>
        <span>
          <i className="review" />
          {t("workspace.legend.review")}
        </span>
        <span>
          <i className="conflict" />
          {t("workspace.legend.conflict")}
        </span>
        <span>
          <i className="merged" />
          {t("workspace.legend.merged")}
        </span>
        <b>
          {t("workspace.graphSummary", {
            nodes: tasks.length,
            links: dependencies.length,
          })}
        </b>
      </div>
      <div className="graph-stage" data-testid="feature-graph">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onInit={setFlow}
          defaultViewport={{ x: 0, y: 0, zoom: initialGraphZoom }}
          onMoveEnd={(_, viewport) => {
            graphZoom.current = viewport.zoom;
            writeGraphZoom(viewport.zoom);
          }}
          onNodeClick={(_, node) => setSelected(node.id)}
          minZoom={minGraphZoom}
          maxZoom={maxGraphZoom}
          nodesDraggable={false}
          defaultEdgeOptions={{ animated: false }}
          ariaLabelConfig={ariaLabelConfig}
        >
          <Background variant={BackgroundVariant.Dots} gap={22} size={1} />
          <Controls showInteractive={false} />
        </ReactFlow>
        {tasks.length === 0 && !layoutError && (
          <div className="graph-empty">
            <span>＋</span>
            <h2>{t("workspace.graphEmptyTitle")}</h2>
            <p>{t("workspace.graphEmptyDetail")}</p>
            <button onClick={() => setShowTask(true)}>
              {t("workspace.addTaskPlain")}
            </button>
          </div>
        )}
        {layoutError && (
          <div className="graph-empty" role="alert">
            <span>⚠</span>
            <h2>{t("workspace.layoutErrorTitle")}</h2>
            <p>{layoutError.message ?? t("workspace.layoutErrorFallback")}</p>
            <button
              onClick={() => {
                setLayoutError(undefined);
                setLayoutAttempt((attempt) => attempt + 1);
              }}
            >
              {t("workspace.retryLayout")}
            </button>
          </div>
        )}
      </div>
      {selectedTask && (
        <TaskInspector
          task={selectedTask}
          tasks={tasks}
          dependencies={dependencies}
          pr={prs.get(selectedTask.id)}
          documents={data.documents.filter(
            (document) => document.taskId === selectedTask.id,
          )}
          onClose={() => setSelected(undefined)}
        />
      )}
      {showTask && (
        <div className="scrim">
          <form
            className="dialog"
            onSubmit={submitTask}
            aria-label={t("taskCreate.formLabel")}
          >
            <header>
              <p>{t("taskCreate.eyebrow")}</p>
              <h2>{t("taskCreate.title")}</h2>
            </header>
            <label>
              {t("common.title")}
              <input
                name="title"
                required
                placeholder={t("taskCreate.titlePlaceholder")}
              />
            </label>
            <label>
              {t("common.scope")}
              <textarea
                name="scope"
                placeholder={t("taskCreate.scopePlaceholder")}
              />
            </label>
            <div className="form-row">
              <label>
                {t("taskCreate.kind")}
                <select name="kind">
                  <option value={TaskKind.PULL_REQUEST}>
                    {taskKindLabel(TaskKind.PULL_REQUEST, t)}
                  </option>
                  <option value={TaskKind.MANUAL}>
                    {taskKindLabel(TaskKind.MANUAL, t)}
                  </option>
                </select>
              </label>
              <label>
                {t("common.assignee")}
                <input
                  name="assignee"
                  placeholder={t("taskCreate.assigneePlaceholder")}
                />
              </label>
            </div>
            {createTask.error && (
              <p className="form-error">{formatError(createTask.error, t)}</p>
            )}
            <footer>
              <button
                type="button"
                className="secondary"
                onClick={() => setShowTask(false)}
              >
                {t("common.cancel")}
              </button>
              <button disabled={createTask.isPending}>
                {t("taskCreate.submit")}
              </button>
            </footer>
            <MutationError error={createTask.error} />
          </form>
        </div>
      )}
      {showFeatureEdit && (
        <div className="scrim">
          <form
            className="dialog"
            onSubmit={submitFeature}
            aria-label={t("featureEdit.formLabel")}
          >
            <header>
              <p>{t("featureEdit.eyebrow")}</p>
              <h2>{t("featureEdit.title")}</h2>
            </header>
            <label>
              {t("common.slug")}
              <input name="slug" required defaultValue={feature.slug} />
            </label>
            <label>
              {t("common.title")}
              <input name="title" required defaultValue={feature.title} />
            </label>
            <label>
              {t("common.description")}
              <textarea name="description" defaultValue={feature.description} />
            </label>
            <label>
              {t("common.status")}
              <select name="status" defaultValue={feature.status}>
                {[
                  FeatureStatus.ACTIVE,
                  FeatureStatus.PAUSED,
                  FeatureStatus.COMPLETED,
                  FeatureStatus.CANCELLED,
                ].map((status) => (
                  <option value={status} key={status}>
                    {featureStatusLabel(status, t)}
                  </option>
                ))}
              </select>
            </label>
            <MutationError error={updateFeature.error} />
            <footer>
              <button
                type="button"
                className="secondary"
                onClick={() => setShowFeatureEdit(false)}
              >
                {t("common.cancel")}
              </button>
              <button disabled={updateFeature.isPending}>
                {t("featureEdit.submit")}
              </button>
            </footer>
          </form>
        </div>
      )}
    </div>
  );
}

type InspectorProps = {
  task: {
    id: string;
    title: string;
    scope: string;
    kind: TaskKind;
    status: TaskStatus;
    assignee: string;
    displayState: TaskDisplayState;
    blockedReason?: BlockedReason;
  };
  tasks: Array<{ id: string; title: string }>;
  dependencies: Array<{ blockerTaskId: string; blockedTaskId: string }>;
  pr?: {
    url: string;
    owner: string;
    repository: string;
    number: bigint;
    displayState: PullRequestDisplayState;
    syncError: string;
    stale: boolean;
  };
  documents: Array<{
    id: string;
    kind: DocumentKind;
    title: string;
    value: string;
  }>;
  onClose: () => void;
};
function MutationError({
  error,
  taskTitle,
}: {
  error: Error | null;
  taskTitle?: (id: string) => string | undefined;
}) {
  const { t } = useTranslation();
  if (!error) return null;
  return (
    <p className="form-error" role="alert">
      {formatError(error, t, taskTitle)}
    </p>
  );
}

function TaskInspector({
  task,
  tasks,
  dependencies,
  pr,
  documents,
  onClose,
}: InspectorProps) {
  const { t } = useTranslation();
  const update = useDomainMutation(mutations.updateTask);
  const remove = useDomainMutation(mutations.deleteTask);
  const attach = useDomainMutation(
    ({ taskId, url }: { taskId: string; url: string }) =>
      mutations.attachPR(taskId, url),
  );
  const detach = useDomainMutation(mutations.detachPR);
  const addDep = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.addDependency(blocker, blocked),
  );
  const removeDep = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.removeDependency(blocker, blocked),
  );
  const addDoc = useDomainMutation(mutations.addDocument);
  const deleteDoc = useDomainMutation(mutations.deleteDocument);
  const blockers = dependencies.filter((d) => d.blockedTaskId === task.id);
  return (
    <aside className="inspector" aria-label={t("inspector.label")}>
      <header>
        <div>
          <p>{t("inspector.eyebrow")}</p>
          <h2>{task.title}</h2>
        </div>
        <button aria-label={t("inspector.close")} onClick={onClose}>
          ×
        </button>
      </header>
      <div
        className={`inspector-state state-${taskDisplayStateToken(task.displayState)}`}
      >
        <i />
        {taskDisplayStateLabel(task.displayState, t)}
        {task.blockedReason && (
          <small>
            {blockedReasonLabel(
              task.blockedReason,
              (id) => tasks.find((item) => item.id === id)?.title,
              t,
            )}
          </small>
        )}
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const f = new FormData(e.currentTarget);
          update.mutate({
            id: task.id,
            title: formValue(f, "title"),
            scope: formValue(f, "scope"),
            status: Number(f.get("status")),
            assignee: formValue(f, "assignee"),
          });
        }}
      >
        <label>
          {t("common.title")}
          <input name="title" defaultValue={task.title} />
        </label>
        <label>
          {t("common.scope")}
          <textarea name="scope" defaultValue={task.scope} />
        </label>
        <div className="form-row">
          <label>
            {t("common.status")}
            <select name="status" defaultValue={task.status}>
              <option value={TaskStatus.PLANNED}>
                {taskStatusLabel(TaskStatus.PLANNED, t)}
              </option>
              <option value={TaskStatus.IN_PROGRESS}>
                {taskStatusLabel(TaskStatus.IN_PROGRESS, t)}
              </option>
              {/* A PR task completes when its pull request merges. */}
              {task.kind !== TaskKind.PULL_REQUEST && (
                <option value={TaskStatus.COMPLETED}>
                  {taskStatusLabel(TaskStatus.COMPLETED, t)}
                </option>
              )}
              <option value={TaskStatus.CANCELLED}>
                {taskStatusLabel(TaskStatus.CANCELLED, t)}
              </option>
            </select>
          </label>
          <label>
            {t("common.assignee")}
            <input name="assignee" defaultValue={task.assignee} />
          </label>
        </div>
        <button disabled={update.isPending}>{t("inspector.saveTask")}</button>
        <MutationError error={update.error} />
      </form>
      <section>
        <h3>{t("inspector.pullRequest")}</h3>
        {pr ? (
          <div className="linked-pr">
            <a href={pr.url} target="_blank" rel="noreferrer">
              {pr.owner}/{pr.repository} #{String(pr.number)}
            </a>
            <span>
              {pr.stale
                ? t("inspector.stale")
                : pullRequestDisplayStateLabel(pr.displayState, t)}
            </span>
            {pr.syncError && <p>{pr.syncError}</p>}
            <button
              className="text-action"
              onClick={() => detach.mutate(task.id)}
            >
              {t("inspector.detach")}
            </button>
          </div>
        ) : (
          <form
            className="inline-form"
            onSubmit={(e) => {
              e.preventDefault();
              attach.mutate({
                taskId: task.id,
                url: formValue(new FormData(e.currentTarget), "url"),
              });
            }}
          >
            <input
              name="url"
              required
              placeholder="https://github.com/org/repo/pull/42"
            />
            <button>{t("inspector.attach")}</button>
          </form>
        )}
        <MutationError error={attach.error} />
        <MutationError error={detach.error} />
      </section>
      <section>
        <h3>{t("inspector.blockedBy")}</h3>
        {blockers.map((dep) => (
          <div className="dependency-chip" key={dep.blockerTaskId}>
            <span>{tasks.find((t) => t.id === dep.blockerTaskId)?.title}</span>
            <button
              aria-label={t("inspector.removeDependency")}
              onClick={() =>
                removeDep.mutate({
                  blocker: dep.blockerTaskId,
                  blocked: task.id,
                })
              }
            >
              ×
            </button>
          </div>
        ))}
        <form
          className="inline-form"
          onSubmit={(e) => {
            e.preventDefault();
            addDep.mutate({
              blocker: formValue(new FormData(e.currentTarget), "blocker"),
              blocked: task.id,
            });
          }}
        >
          <select
            name="blocker"
            aria-label={t("inspector.blockerTask")}
            defaultValue=""
          >
            <option value="" disabled>
              {t("inspector.selectBlocker")}
            </option>
            {tasks
              .filter(
                (candidate) =>
                  candidate.id !== task.id &&
                  !blockers.some((d) => d.blockerTaskId === candidate.id),
              )
              .map((candidate) => (
                <option key={candidate.id} value={candidate.id}>
                  {candidate.title}
                </option>
              ))}
          </select>
          <button>{t("common.add")}</button>
        </form>
        <MutationError
          error={addDep.error}
          taskTitle={(id) => tasks.find((item) => item.id === id)?.title}
        />
        <MutationError error={removeDep.error} />
      </section>
      <section>
        <h3>{t("inspector.references")}</h3>
        {documents.map((document) => (
          <div className="document-chip" key={document.id}>
            <span>
              <b>{document.title || documentKindLabel(document.kind, t)}</b>
              <small>{document.value}</small>
            </span>
            <button
              aria-label={t("inspector.deleteReference", {
                title: document.title || t("inspector.referenceFallback"),
              })}
              onClick={() => deleteDoc.mutate(document.id)}
            >
              ×
            </button>
          </div>
        ))}
        <form
          className="stack-form"
          onSubmit={(e) => {
            e.preventDefault();
            const f = new FormData(e.currentTarget);
            addDoc.mutate({
              taskId: task.id,
              kind: Number(f.get("kind")),
              title: formValue(f, "title"),
              value: formValue(f, "value"),
            });
          }}
        >
          <div className="form-row">
            <select name="kind">
              <option value={DocumentKind.URL}>
                {documentKindLabel(DocumentKind.URL, t)}
              </option>
              <option value={DocumentKind.MARKDOWN_PATH}>
                {documentKindLabel(DocumentKind.MARKDOWN_PATH, t)}
              </option>
            </select>
            <input name="title" placeholder={t("inspector.designNotes")} />
          </div>
          <input
            name="value"
            required
            placeholder={t("inspector.referenceValue")}
          />
          <button>{t("inspector.addReference")}</button>
        </form>
        <MutationError error={addDoc.error} />
        <MutationError error={deleteDoc.error} />
      </section>
      <button
        className="danger-zone"
        onClick={() => {
          if (
            !window.confirm(
              t("inspector.deleteTaskConfirm", { title: task.title }),
            )
          )
            return;
          remove.mutateAsync(task.id).then(onClose, () => {});
        }}
      >
        {t("inspector.deleteTask")}
      </button>
      <MutationError error={remove.error} />
    </aside>
  );
}
