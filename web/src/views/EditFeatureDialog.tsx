import { Archive, ArchiveRestore, Save, Trash2, X } from "lucide-react";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { unfinishedTaskCount } from "../feature-status";
import { formValue } from "../form";
import { FeatureStatus, type Feature } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { featureStatusLabel } from "../i18n/domain";
import { ConfirmationDialog } from "./ConfirmationDialog";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";

interface EditFeatureDialogProps {
  feature: Feature;
  onClose: () => void;
  onDeleted: () => void;
}

type Confirmation = "archive" | "complete" | "delete";

// The confirmation replaces the submit, so the values the form held when it was
// submitted are kept until the person confirms or cancels.
type FeatureUpdate = Parameters<typeof mutations.updateFeature>[0];

export function EditFeatureDialog({
  feature,
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
    };
    if (status === FeatureStatus.COMPLETED && unfinished > 0) {
      setPendingUpdate(update);
      setConfirmation("complete");
      return;
    }
    await applyUpdate(update);
  }

  async function archiveFeature() {
    await applyUpdate({ id: feature.id, archived: true });
  }

  async function restoreFeature() {
    await applyUpdate({ id: feature.id, archived: false });
  }

  async function removeFeature() {
    try {
      await deleteFeature.mutateAsync(feature.id);
    } catch {
      return;
    }
    onDeleted();
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
          updatePending={updateFeature.isPending}
          deletePending={deleteFeature.isPending}
          updateError={updateFeature.error}
          onSubmit={submitFeature}
          onClose={onClose}
          onArchive={() => {
            setConfirmation("archive");
          }}
          onRestore={() => {
            void restoreFeature();
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
          void archiveFeature();
        }}
        onComplete={() => {
          if (pendingUpdate) void applyUpdate(pendingUpdate);
        }}
        onDelete={() => {
          void removeFeature();
        }}
      />
    </>
  );
}

interface FeatureDialogContentProps {
  feature: Feature;
  updatePending: boolean;
  deletePending: boolean;
  updateError: Error | null;
  onSubmit: (event: SyntheticEvent<HTMLFormElement>) => void;
  onClose: () => void;
  onArchive: () => void;
  onRestore: () => void;
  onDelete: () => void;
}

function FeatureDialogContent(props: FeatureDialogContentProps) {
  return props.feature.archived ? (
    <ArchivedFeatureDialog {...props} />
  ) : (
    <ActiveFeatureDialog {...props} />
  );
}

function ArchivedFeatureDialog({
  feature,
  updatePending,
  deletePending,
  updateError,
  onClose,
  onRestore,
  onDelete,
}: FeatureDialogContentProps) {
  const { t } = useTranslation();
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
        archived
        updatePending={updatePending}
        deletePending={deletePending}
        onRestore={onRestore}
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
  updatePending,
  deletePending,
  updateError,
  onSubmit,
  onClose,
  onArchive,
  onDelete,
}: FeatureDialogContentProps) {
  const { t } = useTranslation();
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
      <MutationError error={updateError} />
      <LifecycleActions
        archived={false}
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

function LifecycleActions({
  archived,
  updatePending,
  deletePending,
  onArchive,
  onRestore,
  onDelete,
}: {
  archived: boolean;
  updatePending: boolean;
  deletePending: boolean;
  onArchive?: () => void;
  onRestore?: () => void;
  onDelete: () => void;
}) {
  const { t } = useTranslation();
  return (
    <section
      className="feature-lifecycle"
      aria-label={t("featureEdit.lifecycle")}
    >
      <div>
        <h3 className="feature-lifecycle-title">
          {t("featureEdit.lifecycle")}
        </h3>
        <p className="feature-lifecycle-detail">
          {archived
            ? t("featureEdit.archivedLifecycleDetail")
            : t("featureEdit.activeLifecycleDetail")}
        </p>
      </div>
      <div className="feature-lifecycle-actions">
        {archived ? (
          <IconButton
            icon={ArchiveRestore}
            label={t("featureEdit.restore")}
            variant="primary"
            disabled={updatePending}
            onClick={onRestore}
          />
        ) : (
          <IconButton
            icon={Archive}
            label={t("featureEdit.archive")}
            variant="secondary"
            disabled={updatePending}
            onClick={onArchive}
          />
        )}
        <IconButton
          icon={Trash2}
          label={t("featureEdit.delete")}
          variant="danger"
          disabled={deletePending}
          onClick={onDelete}
        />
      </div>
    </section>
  );
}
