import { X } from "lucide-react";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { unfinishedTaskCount } from "../feature-status";
import {
  FeatureStatus,
  type Feature,
  type Project,
} from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { featureStatusLabel } from "../i18n/domain";
import { ConfirmationDialog } from "./ConfirmationDialog";
import { IconButton } from "./IconButton";
import { LifecycleActions, type LifecycleLabels } from "./LifecycleActions";
import { MutationError } from "./MutationError";
import { ProjectSelectField } from "./ProjectSelectField";
import { TitleDescriptionFields } from "./TitleDescriptionFields";
import { DiscardChangesDialog, SaveButton } from "./UnsavedChanges";

interface EditFeatureDialogProps {
  feature: Feature;
  projects: Project[];
  onClose: () => void;
  onDeleted: () => void;
}

type Confirmation = "archive" | "complete" | "delete" | "discard";

interface FeatureDraft {
  title: string;
  description: string;
  status: FeatureStatus;
  projectId: string;
}

// 確認が送信の代わりになるため、送信時点のフォームの値は確認かキャンセルまで
// 保持する。
type FeatureUpdate = Parameters<typeof mutations.updateFeature>[0];

export function EditFeatureDialog({
  feature,
  projects,
  onClose,
  onDeleted,
}: EditFeatureDialogProps) {
  const [confirmation, setConfirmation] = useState<Confirmation>();
  const [pendingUpdate, setPendingUpdate] = useState<FeatureUpdate>();
  const [draft, setDraft] = useState<FeatureDraft>(() => ({
    title: feature.title,
    description: feature.description,
    status: feature.status,
    projectId: feature.projectId,
  }));
  const updateFeature = useDomainMutation(mutations.updateFeature);
  const deleteFeature = useDomainMutation(mutations.deleteFeature);
  const unfinished = unfinishedTaskCount(feature);
  const dirty =
    draft.title !== feature.title ||
    draft.description !== feature.description ||
    draft.status !== feature.status ||
    draft.projectId !== feature.projectId;

  async function applyUpdate(update: FeatureUpdate) {
    try {
      await updateFeature.mutateAsync(update);
    } catch {
      return;
    }
    onClose();
  }

  async function submitFeature(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const update: FeatureUpdate = { id: feature.id, ...draft };
    if (draft.status === FeatureStatus.COMPLETED && unfinished > 0) {
      setPendingUpdate(update);
      setConfirmation("complete");
      return;
    }
    await applyUpdate(update);
  }

  return (
    <>
      <div
        className="scrim"
        aria-hidden={confirmation ? true : undefined}
        inert={confirmation !== undefined}
      >
        <FeatureDialogContent
          feature={feature}
          projects={projects}
          draft={draft}
          dirty={dirty}
          onDraftChange={setDraft}
          updatePending={updateFeature.isPending}
          deletePending={deleteFeature.isPending}
          updateError={updateFeature.error}
          onSubmit={submitFeature}
          onClose={() => {
            if (dirty) setConfirmation("discard");
            else onClose();
          }}
          onArchive={() => {
            setConfirmation("archive");
          }}
          onRestore={() => {
            void applyUpdate({ id: feature.id, archived: false });
          }}
          onDelete={() => {
            setConfirmation("delete");
          }}
        />
      </div>
      <LifecycleConfirmation
        confirmation={confirmation}
        feature={feature}
        unfinished={unfinished}
        updatePending={updateFeature.isPending}
        deletePending={deleteFeature.isPending}
        updateError={updateFeature.error}
        deleteError={deleteFeature.error}
        onCancel={() => {
          setConfirmation(undefined);
          setPendingUpdate(undefined);
        }}
        onArchive={() => {
          void applyUpdate({ id: feature.id, archived: true });
        }}
        onComplete={() => {
          if (pendingUpdate) void applyUpdate(pendingUpdate);
        }}
        onDelete={() => {
          void (async () => {
            try {
              await deleteFeature.mutateAsync(feature.id);
            } catch {
              return;
            }
            onDeleted();
          })();
        }}
        onDiscard={onClose}
      />
    </>
  );
}

interface FeatureDialogContentProps {
  feature: Feature;
  projects: Project[];
  draft: FeatureDraft;
  dirty: boolean;
  onDraftChange: (next: FeatureDraft) => void;
  updatePending: boolean;
  deletePending: boolean;
  updateError: Error | null;
  onSubmit: (event: SyntheticEvent<HTMLFormElement>) => void;
  onClose: () => void;
  onArchive: () => void;
  onRestore: () => void;
  onDelete: () => void;
}

// 読み取り専用の feature は、自身がアーカイブでもアーカイブ済みプロジェクト配下でも
// 編集できないので、編集フォームごと差し替える。
function FeatureDialogContent(props: FeatureDialogContentProps) {
  return props.feature.readOnly ? (
    <ReadOnlyFeatureDialog {...props} />
  ) : (
    <ActiveFeatureDialog {...props} />
  );
}

function useFeatureLifecycleLabels(archived: boolean): LifecycleLabels {
  const { t } = useTranslation();
  return {
    section: t("featureEdit.lifecycle"),
    detail: archived
      ? t("featureEdit.archivedLifecycleDetail")
      : t("featureEdit.activeLifecycleDetail"),
    archive: t("featureEdit.archive"),
    restore: t("featureEdit.restore"),
    remove: t("featureEdit.delete"),
  };
}

function ReadOnlyFeatureDialog({
  feature,
  projects,
  updatePending,
  deletePending,
  updateError,
  onClose,
  onRestore,
  onDelete,
}: FeatureDialogContentProps) {
  const { t } = useTranslation();
  const labels = useFeatureLifecycleLabels(true);
  // プロジェクトがアーカイブ済みだと feature 側のフラグを解除しても読み取り
  // 専用のままなので、原因が feature 自身だけのときにだけ復元を出す。
  const projectArchived =
    projects.find((item) => item.id === feature.projectId)?.archived ?? false;
  const restorable = feature.archived && !projectArchived;
  return (
    <section
      className="dialog feature-management-dialog"
      role="dialog"
      aria-modal="true"
      aria-label={t("featureEdit.manageLabel")}
    >
      <header>
        <p className="section-label">{t("featureEdit.archivedEyebrow")}</p>
        <h2>{t("featureEdit.manageTitle")}</h2>
      </header>
      <dl className="read-only-values">
        <div>
          <dt>{t("common.title")}</dt>
          <dd>{feature.title}</dd>
        </div>
        <div>
          <dt>{t("common.description")}</dt>
          <dd>{feature.description || t("workspace.noDescription")}</dd>
        </div>
        <div>
          <dt>{t("common.status")}</dt>
          <dd>{featureStatusLabel(feature.displayStatus, t)}</dd>
        </div>
      </dl>
      <MutationError error={updateError} />
      <LifecycleActions
        labels={
          restorable
            ? labels
            : { ...labels, detail: t("featureEdit.projectArchivedDetail") }
        }
        updatePending={updatePending}
        deletePending={deletePending}
        // 復元が意味を持つのは feature 自身がアーカイブされている場合だけ。
        // プロジェクトから継いだものはプロジェクト側で解除する。
        {...(restorable ? { onRestore } : {})}
        onDelete={onDelete}
      />
      <footer>
        <IconButton
          icon={X}
          label={t("common.close")}
          variant="secondary"
          onClick={onClose}
        />
      </footer>
    </section>
  );
}

function ActiveFeatureDialog({
  feature,
  projects,
  draft,
  dirty,
  onDraftChange,
  updatePending,
  deletePending,
  updateError,
  onSubmit,
  onClose,
  onArchive,
  onDelete,
}: FeatureDialogContentProps) {
  const { t } = useTranslation();
  const labels = useFeatureLifecycleLabels(false);
  return (
    <form
      className="dialog feature-management-dialog"
      onSubmit={onSubmit}
      aria-label={t("featureEdit.formLabel")}
    >
      <header>
        <h2>{t("featureEdit.title")}</h2>
      </header>
      <TitleDescriptionFields draft={draft} onChange={onDraftChange} />
      <label>
        {t("common.status")}
        <select
          name="status"
          value={draft.status}
          onChange={(event) => {
            onDraftChange({ ...draft, status: Number(event.target.value) });
          }}
        >
          {[
            FeatureStatus.AUTO,
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
      <ProjectSelectField
        projects={projects}
        currentProjectId={feature.projectId}
        value={draft.projectId}
        onChange={(projectId) => {
          onDraftChange({ ...draft, projectId });
        }}
      />
      <MutationError error={updateError} />
      <LifecycleActions
        labels={labels}
        updatePending={updatePending}
        deletePending={deletePending}
        onArchive={onArchive}
        onDelete={onDelete}
      />
      <footer>
        <IconButton
          icon={X}
          label={t("common.cancel")}
          variant="secondary"
          onClick={onClose}
        />
        <SaveButton
          dirty={dirty}
          label={t("featureEdit.submit")}
          pending={updatePending}
          type="submit"
        />
      </footer>
    </form>
  );
}

function LifecycleConfirmation({
  confirmation,
  feature,
  unfinished,
  updatePending,
  deletePending,
  updateError,
  deleteError,
  onCancel,
  onArchive,
  onComplete,
  onDelete,
  onDiscard,
}: {
  confirmation: Confirmation | undefined;
  feature: Feature;
  unfinished: number;
  updatePending: boolean;
  deletePending: boolean;
  updateError: Error | null;
  deleteError: Error | null;
  onCancel: () => void;
  onArchive: () => void;
  onComplete: () => void;
  onDelete: () => void;
  onDiscard: () => void;
}) {
  const { t } = useTranslation();
  if (!confirmation) return null;
  if (confirmation === "discard")
    return <DiscardChangesDialog onCancel={onCancel} onConfirm={onDiscard} />;
  if (confirmation === "archive")
    return (
      <ConfirmationDialog
        title={t("featureEdit.archiveTitle", { title: feature.title })}
        description={t("featureEdit.archiveDescription")}
        confirmLabel={t("featureEdit.confirmArchive")}
        pending={updatePending}
        error={updateError}
        onCancel={onCancel}
        onConfirm={onArchive}
      />
    );
  if (confirmation === "complete")
    return (
      <ConfirmationDialog
        title={t("featureEdit.completeTitle", { title: feature.title })}
        description={t("featureEdit.completeDescription", {
          count: unfinished,
        })}
        confirmLabel={t("featureEdit.confirmComplete")}
        pending={updatePending}
        error={updateError}
        onCancel={onCancel}
        onConfirm={onComplete}
      />
    );
  return (
    <ConfirmationDialog
      title={t("featureEdit.deleteTitle", { title: feature.title })}
      description={t("featureEdit.deleteDescription", {
        count: feature.taskCount,
      })}
      confirmLabel={t("featureEdit.confirmDelete")}
      danger
      pending={deletePending}
      error={deleteError}
      onCancel={onCancel}
      onConfirm={onDelete}
    />
  );
}
