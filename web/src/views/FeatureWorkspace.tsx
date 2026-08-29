import { useNavigate, useParams } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "../hooks";
import { MutationError } from "./MutationError";
import { WorkspaceBody } from "./WorkspaceBody";
import { WorkspaceHeader } from "./WorkspaceHeader";
import { WorkspaceOverlays } from "./WorkspaceOverlays";
import { useFeatureWorkspaceActions } from "./useFeatureWorkspaceActions";
import { useFeatureWorkspaceData } from "./useFeatureWorkspaceData";

export function FeatureWorkspace() {
  const { t } = useTranslation();
  const { featureId } = useParams({ from: "/features/$featureId" });
  const navigate = useNavigate();
  const snapshot = useSnapshot();
  const { feature, tasks, dependencies, pullRequests, documentsByTask } =
    useFeatureWorkspaceData(snapshot.data, featureId);
  const actions = useFeatureWorkspaceActions({ feature, featureId, tasks });

  if (snapshot.isPending)
    return (
      <div className="state-message">
        <div className="spinner" />
        <h1>{t("workspace.loading")}</h1>
      </div>
    );
  if (!feature || !snapshot.data)
    return (
      <div className="state-message">
        <h1>{t("workspace.notFound")}</h1>
        <button onClick={() => void navigate({ to: "/" })}>
          {t("workspace.returnOverview")}
        </button>
      </div>
    );

  return (
    <div className="workspace">
      <WorkspaceHeader
        feature={feature}
        syncPending={actions.syncPending}
        onSync={actions.syncFeature}
        onCreateTask={actions.openTaskDialog}
        onEdit={actions.openFeatureEdit}
        onToggleArchive={actions.toggleArchive}
        onDelete={actions.removeFeature}
      />
      <MutationError error={actions.deleteError} />
      <WorkspaceBody
        tasks={tasks}
        dependencies={dependencies}
        pullRequests={pullRequests}
        documentsByTask={documentsByTask}
        selectedTask={actions.selectedTask}
        onEditTask={actions.editTask}
        onPreviewDocument={actions.handlePreviewDocument}
        onCreateTask={actions.openTaskDialog}
        onCloseInspector={actions.closeInspector}
      />
      <WorkspaceOverlays
        feature={feature}
        featureId={featureId}
        previewDocument={actions.previewDocument}
        showTask={actions.showTask}
        showFeatureEdit={actions.showFeatureEdit}
        onClosePreview={actions.closePreview}
        onCloseTask={actions.closeTask}
        onCloseFeatureEdit={actions.closeFeatureEdit}
      />
    </div>
  );
}
