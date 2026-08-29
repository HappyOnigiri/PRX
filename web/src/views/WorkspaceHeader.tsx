import { useTranslation } from "react-i18next";
import type { Feature } from "../gen/prx/v1/prx_pb";

interface WorkspaceHeaderProps {
  feature: Feature;
  syncPending: boolean;
  onSync: () => void;
  onCreateTask: () => void;
  onEdit: () => void;
  onToggleArchive: () => void;
  onDelete: () => void;
}

export function WorkspaceHeader({
  feature,
  syncPending,
  onSync,
  onCreateTask,
  onEdit,
  onToggleArchive,
  onDelete,
}: WorkspaceHeaderProps) {
  const { t } = useTranslation();
  return (
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
        <button className="secondary" onClick={onSync} disabled={syncPending}>
          {syncPending ? t("workspace.syncing") : t("workspace.syncGithub")}
        </button>
        <button onClick={onCreateTask}>{t("workspace.addTask")}</button>
        <button
          className="icon-button"
          aria-label={t("workspace.editFeature")}
          onClick={onEdit}
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
          onClick={onToggleArchive}
        >
          ⌁
        </button>
        <button
          className="icon-button danger"
          aria-label={t("workspace.deleteFeature")}
          onClick={onDelete}
        >
          ×
        </button>
      </div>
    </header>
  );
}
