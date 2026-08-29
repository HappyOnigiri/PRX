import { useCallback, useMemo, useState } from "react";
import { useNavigate, useParams } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { useDomainMutation, useSnapshot } from "../hooks";
import { type TaskNodeDocument } from "./TaskNode";
import { CreateTaskDialog } from "./CreateTaskDialog";
import { EditFeatureDialog } from "./EditFeatureDialog";
import { FeatureGraph } from "./FeatureGraph";
import { MutationError } from "./MutationError";
import { TaskInspector } from "./TaskInspector";
import { MarkdownPreview } from "./MarkdownPreview";

export function FeatureWorkspace() {
  const { t } = useTranslation();
  const { featureId } = useParams({ from: "/features/$featureId" });
  const navigate = useNavigate();
  const snapshot = useSnapshot();
  const [selected, setSelected] = useState<string>();
  const [showTask, setShowTask] = useState(false);
  const [showFeatureEdit, setShowFeatureEdit] = useState(false);
  const [previewDocument, setPreviewDocument] = useState<TaskNodeDocument>();
  const data = snapshot.data;
  const feature = data?.features.find((item) => item.id === featureId);
  const tasks = useMemo(
    () => data?.tasks.filter((task) => task.featureId === featureId) ?? [],
    [data, featureId],
  );
  const taskIds = useMemo(() => new Set(tasks.map((task) => task.id)), [tasks]);
  const dependencies = useMemo(
    () =>
      data?.dependencies.filter((dependency) =>
        taskIds.has(dependency.blockerTaskId),
      ) ?? [],
    [data, taskIds],
  );
  const pullRequests = useMemo(
    () => new Map(data?.pullRequests.map((pr) => [pr.taskId, pr]) ?? []),
    [data],
  );
  const documentsByTask = useMemo(() => {
    const result = new Map<string, TaskNodeDocument[]>();
    for (const document of data?.documents ?? []) {
      if (!document.taskId) continue;
      const documents = result.get(document.taskId) ?? [];
      documents.push(document);
      result.set(document.taskId, documents);
    }
    return result;
  }, [data]);
  const editTask = useCallback((taskId: string) => {
    setSelected(taskId);
  }, []);
  const handlePreviewDocument = useCallback((document: TaskNodeDocument) => {
    setPreviewDocument(document);
  }, []);
  const openTaskDialog = useCallback(() => {
    setShowTask(true);
  }, []);
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
        <div
          className="workspace-title"
          title={feature.description || t("workspace.noDescription")}
        >
          <h1>{feature.title}</h1>
          <p className="eyebrow">
            {t("workspace.eyebrow", { slug: feature.slug })}
          </p>
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
          <button onClick={openTaskDialog}>{t("workspace.addTask")}</button>
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
      <div className="workspace-body">
        <FeatureGraph
          tasks={tasks}
          dependencies={dependencies}
          pullRequests={pullRequests}
          documentsByTask={documentsByTask}
          onEditTask={editTask}
          onPreviewDocument={handlePreviewDocument}
          onCreateTask={openTaskDialog}
        />
        {selectedTask && (
          <TaskInspector
            task={selectedTask}
            tasks={tasks}
            dependencies={dependencies}
            pullRequest={pullRequests.get(selectedTask.id)}
            documents={documentsByTask.get(selectedTask.id) ?? []}
            onPreview={handlePreviewDocument}
            onClose={() => setSelected(undefined)}
          />
        )}
      </div>
      {previewDocument && (
        <MarkdownPreview
          key={previewDocument.id}
          document={previewDocument}
          onClose={() => setPreviewDocument(undefined)}
        />
      )}
      {showTask && (
        <CreateTaskDialog
          featureId={featureId}
          onClose={() => setShowTask(false)}
        />
      )}
      {showFeatureEdit && (
        <EditFeatureDialog
          feature={feature}
          onClose={() => setShowFeatureEdit(false)}
        />
      )}
    </div>
  );
}
