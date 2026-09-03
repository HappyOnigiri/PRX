import { Save, X } from "lucide-react";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { unfinishedTaskCount } from "../feature-status";
import { formValue } from "../form";
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

interface EditFeatureDialogProps {
  feature: Feature;
  projects: Project[];
  onClose: () => void;
  onDeleted: () => void;
}

type Confirmation = "archive" | "complete" | "delete";

// The confirmation replaces the submit, so the values the form held when it was
// submitted are kept until the person confirms or cancels.
type FeatureUpdate = Parameters<typeof mutations.updateFeature>[0];

export function EditFeatureDialog({
  feature,
  projects,
  onClose,
  onDeleted,
}: EditFeatureDialogProps) {
  const [confirmation, setConfirmation] = useState<Confirmation>();
  const [pendingUpdate, setPendingUpdate] = useState<FeatureUpdate>();
  const updateFeature = useDomainMutation(mutations.updateFeature);
  const deleteFeature = useDomainMutation(mutations.deleteFeature);
  const unfinished = unfinishedTaskCount(feature);

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
    const form = new FormData(event.currentTarget);
    const status: FeatureStatus = Number(form.get("status"));
    const update: FeatureUpdate = {
      id: feature.id,
      slug: formValue(form, "slug"),
      title: formValue(form, "title"),
      description: formValue(form, "description"),
      status,
      projectId: formValue(form, "projectId"),
    };
    if (status === FeatureStatus.COMPLETED && unfinished > 0) {
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
          updatePending={updateFeature.isPending}
          deletePending={deleteFeature.isPending}
          updateError={updateFeature.error}
          onSubmit={submitFeature}
          onClose={onClose}
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
      />
    </>
  );
}

interface FeatureDialogContentProps {
  feature: Feature;
  projects: Project[];
  updatePending: boolean;
  deletePending: boolean;
  updateError: Error | null;
  onSubmit: (event: SyntheticEvent<HTMLFormElement>) => void;
  onClose: () => void;
  onArchive: () => void;
  onRestore: () => void;
  onDelete: () => void;
}

// A read-only feature cannot be edited, whether it is archived itself or sits
// inside an archived project, so the editable form is replaced entirely.
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
  updatePending,
  deletePending,
  updateError,
  onClose,
  onRestore,
  onDelete,
}: FeatureDialogContentProps) {
  const { t } = useTranslation();
  const labels = useFeatureLifecycleLabels(true);
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
          <dt>{t("common.slug")}</dt>
          <dd>{feature.slug}</dd>
        </div>
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
          feature.archived
            ? labels
            : { ...labels, detail: t("featureEdit.projectArchivedDetail") }
        }
        updatePending={updatePending}
        deletePending={deletePending}
        // Restoring is only meaningful when the feature carries the archive
        // itself; one inherited from a project is lifted on the project.
        {...(feature.archived ? { onRestore } : {})}
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
        <IconButton
          icon={Save}
          label={t("featureEdit.submit")}
          variant="primary"
          type="submit"
          disabled={updatePending}
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
}) {
  const { t } = useTranslation();
  if (!confirmation) return null;
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
