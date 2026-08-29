import { useNavigate } from "@tanstack/react-router";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { Feature, Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import type { TaskNodeDocument } from "./TaskNode";

interface FeatureWorkspaceActionsOptions {
  feature: Feature | undefined;
  featureId: string;
  tasks: Task[];
}

export function useFeatureWorkspaceActions({
  feature,
  featureId,
  tasks,
}: FeatureWorkspaceActionsOptions) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [selected, setSelected] = useState<string>();
  const [showTask, setShowTask] = useState(false);
  const [showFeatureEdit, setShowFeatureEdit] = useState(false);
  const [previewDocument, setPreviewDocument] = useState<TaskNodeDocument>();
  const sync = useDomainMutation((id: string) => mutations.sync(id));
  const updateFeature = useDomainMutation(mutations.updateFeature);
  const deleteFeature = useDomainMutation(mutations.deleteFeature);

  const editTask = useCallback((taskId: string) => {
    setSelected(taskId);
  }, []);
  const handlePreviewDocument = useCallback((document: TaskNodeDocument) => {
    setPreviewDocument(document);
  }, []);
  const openTaskDialog = useCallback(() => {
    setShowTask(true);
  }, []);
  const openFeatureEdit = useCallback(() => {
    setShowFeatureEdit(true);
  }, []);
  const syncFeature = useCallback(() => {
    sync.mutate(featureId);
  }, [featureId, sync]);
  const toggleArchive = useCallback(() => {
    if (!feature) return;
    updateFeature.mutate({ id: featureId, archived: !feature.archived });
  }, [feature, featureId, updateFeature]);
  const removeFeature = useCallback(async () => {
    if (!feature) return;
    if (
      !window.confirm(
        t("workspace.deleteFeatureConfirm", { title: feature.title }),
      )
    )
      return;
    try {
      await deleteFeature.mutateAsync(featureId);
    } catch {
      return;
    }
    await navigate({ to: "/" });
  }, [deleteFeature, feature, featureId, navigate, t]);
  const closeInspector = useCallback(() => {
    setSelected(undefined);
  }, []);
  const closePreview = useCallback(() => {
    setPreviewDocument(undefined);
  }, []);
  const closeTask = useCallback(() => {
    setShowTask(false);
  }, []);
  const closeFeatureEdit = useCallback(() => {
    setShowFeatureEdit(false);
  }, []);

  return {
    selectedTask: tasks.find((task) => task.id === selected),
    previewDocument,
    showTask,
    showFeatureEdit,
    editTask,
    handlePreviewDocument,
    openTaskDialog,
    openFeatureEdit,
    syncPending: sync.isPending,
    syncFeature,
    toggleArchive,
    removeFeature,
    deleteError: deleteFeature.error,
    closeInspector,
    closePreview,
    closeTask,
    closeFeatureEdit,
  };
}
