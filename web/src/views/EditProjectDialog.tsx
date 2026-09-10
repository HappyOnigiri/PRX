import { X } from "lucide-react";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { Project } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { ConfirmationDialog } from "./ConfirmationDialog";
import { IconButton } from "./IconButton";
import { LifecycleActions, type LifecycleLabels } from "./LifecycleActions";
import { MutationError } from "./MutationError";
import { TitleDescriptionFields } from "./TitleDescriptionFields";
import { DiscardChangesDialog, SaveButton } from "./UnsavedChanges";
import { useCloseOnEscape } from "./useCloseOnEscape";

interface EditProjectDialogProps {
  project: Project;
  // referenceCount は連鎖削除の対象、つまりプロジェクト自身のドキュメント数。
  // 配下の feature は削除ではなく所属解除なので、確認が示すこの数には含めない。
  referenceCount: number;
  onClose: () => void;
  onDeleted: () => void;
}

type Confirmation = "archive" | "delete" | "discard";

interface ProjectDraft {
  title: string;
  description: string;
}

type ProjectUpdate = Parameters<typeof mutations.updateProject>[0];

export function EditProjectDialog({
  project,
  referenceCount,
  onClose,
  onDeleted,
}: EditProjectDialogProps) {
  const [confirmation, setConfirmation] = useState<Confirmation>();
  const [draft, setDraft] = useState<ProjectDraft>(() => ({
    title: project.title,
    description: project.description,
  }));
  const updateProject = useDomainMutation(mutations.updateProject);
  const deleteProject = useDomainMutation(mutations.deleteProject);
  const dirty =
    draft.title !== project.title || draft.description !== project.description;

  function requestClose() {
    if (dirty) setConfirmation("discard");
    else onClose();
  }

  useCloseOnEscape(requestClose);

  async function applyUpdate(update: ProjectUpdate) {
    try {
      await updateProject.mutateAsync(update);
    } catch {
      return;
    }
    onClose();
  }

  async function submitProject(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    await applyUpdate({ id: project.id, ...draft });
  }

  return (
    <>
      <div
        className="scrim"
        aria-hidden={confirmation ? true : undefined}
        inert={confirmation !== undefined}
      >
        <ProjectDialogContent
          project={project}
          draft={draft}
          dirty={dirty}
          onDraftChange={setDraft}
          updatePending={updateProject.isPending}
          deletePending={deleteProject.isPending}
          updateError={updateProject.error}
          onSubmit={submitProject}
          onClose={requestClose}
          onArchive={() => {
            setConfirmation("archive");
          }}
          onRestore={() => {
            void applyUpdate({ id: project.id, archived: false });
          }}
          onDelete={() => {
            setConfirmation("delete");
          }}
        />
      </div>
      <ProjectLifecycleConfirmation
        confirmation={confirmation}
        project={project}
        referenceCount={referenceCount}
        updatePending={updateProject.isPending}
        deletePending={deleteProject.isPending}
        updateError={updateProject.error}
        deleteError={deleteProject.error}
        onCancel={() => {
          setConfirmation(undefined);
        }}
        onDiscard={onClose}
        onArchive={() => {
          void applyUpdate({ id: project.id, archived: true });
        }}
        onDelete={() => {
          void (async () => {
            try {
              await deleteProject.mutateAsync(project.id);
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

function ProjectLifecycleConfirmation({
  confirmation,
  project,
  referenceCount,
  updatePending,
  deletePending,
  updateError,
  deleteError,
  onCancel,
  onDiscard,
  onArchive,
  onDelete,
}: {
  confirmation: Confirmation | undefined;
  project: Project;
  referenceCount: number;
  updatePending: boolean;
  deletePending: boolean;
  updateError: Error | null;
  deleteError: Error | null;
  onCancel: () => void;
  onDiscard: () => void;
  onArchive: () => void;
  onDelete: () => void;
}) {
  const { t } = useTranslation();
  if (!confirmation) return null;
  if (confirmation === "discard")
    return <DiscardChangesDialog onCancel={onCancel} onConfirm={onDiscard} />;
  if (confirmation === "archive")
    return (
      <ConfirmationDialog
        title={t("projectEdit.archiveTitle", { title: project.title })}
        description={t("projectEdit.archiveDescription")}
        confirmLabel={t("projectEdit.confirmArchive")}
        pending={updatePending}
        error={updateError}
        onCancel={onCancel}
        onConfirm={onArchive}
      />
    );
  return (
    <ConfirmationDialog
      title={t("projectEdit.deleteTitle", { title: project.title })}
      description={t("projectEdit.deleteDescription", {
        count: referenceCount,
      })}
      confirmLabel={t("projectEdit.confirmDelete")}
      danger
      pending={deletePending}
      error={deleteError}
      onCancel={onCancel}
      onConfirm={onDelete}
    />
  );
}

interface ProjectDialogContentProps {
  project: Project;
  draft: ProjectDraft;
  dirty: boolean;
  onDraftChange: (next: ProjectDraft) => void;
  updatePending: boolean;
  deletePending: boolean;
  updateError: Error | null;
  onSubmit: (event: SyntheticEvent<HTMLFormElement>) => void;
  onClose: () => void;
  onArchive: () => void;
  onRestore: () => void;
  onDelete: () => void;
}

function ProjectDialogContent(props: ProjectDialogContentProps) {
  return props.project.archived ? (
    <ArchivedProjectDialog {...props} />
  ) : (
    <ActiveProjectDialog {...props} />
  );
}

function useProjectLifecycleLabels(archived: boolean): LifecycleLabels {
  const { t } = useTranslation();
  return {
    section: t("projectEdit.lifecycle"),
    detail: archived
      ? t("projectEdit.archivedLifecycleDetail")
      : t("projectEdit.activeLifecycleDetail"),
    archive: t("projectEdit.archive"),
    restore: t("projectEdit.restore"),
    remove: t("projectEdit.delete"),
  };
}

function ArchivedProjectDialog({
  project,
  updatePending,
  deletePending,
  updateError,
  onClose,
  onRestore,
  onDelete,
}: ProjectDialogContentProps) {
  const { t } = useTranslation();
  const labels = useProjectLifecycleLabels(true);
  return (
    <section
      className="dialog feature-management-dialog"
      role="dialog"
      aria-modal="true"
      aria-label={t("projectEdit.manageLabel")}
    >
      <header>
        <p className="section-label">{t("projectEdit.archivedEyebrow")}</p>
        <h2>{t("projectEdit.manageTitle")}</h2>
      </header>
      <dl className="read-only-values">
        <div>
          <dt>{t("common.title")}</dt>
          <dd>{project.title}</dd>
        </div>
        <div>
          <dt>{t("common.description")}</dt>
          <dd>{project.description || t("project.noDescription")}</dd>
        </div>
      </dl>
      <MutationError error={updateError} />
      <LifecycleActions
        labels={labels}
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

function ActiveProjectDialog({
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
}: ProjectDialogContentProps) {
  const { t } = useTranslation();
  const labels = useProjectLifecycleLabels(false);
  return (
    <form
      className="dialog feature-management-dialog"
      onSubmit={onSubmit}
      aria-label={t("projectEdit.formLabel")}
    >
      <header>
        <h2>{t("projectEdit.title")}</h2>
      </header>
      <TitleDescriptionFields draft={draft} onChange={onDraftChange} />
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
          label={t("projectEdit.submit")}
          pending={updatePending}
          type="submit"
        />
      </footer>
    </form>
  );
}
