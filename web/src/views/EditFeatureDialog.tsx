import { create } from "@bufbuild/protobuf";
import type { TFunction } from "i18next";
import { X } from "lucide-react";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { unfinishedTaskCount } from "../feature-status";
import {
  FeatureStatus,
  TaskLabelOverridesUpdateSchema,
  type Feature,
  type Project,
  type TaskLabelOverridesUpdate,
} from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { featureStatusLabel } from "../i18n/domain";
import { ConfirmationDialog } from "./ConfirmationDialog";
import { IconButton } from "./IconButton";
import { LifecycleActions, type LifecycleLabels } from "./LifecycleActions";
import { MutationError } from "./MutationError";
import { ProjectSelectField } from "./ProjectSelectField";
import {
  changedPromptValues,
  promptValuesChanged,
  promptValuesOf,
} from "./promptOverrideDraft";
import type { PromptValues } from "./PromptOverridesPanel";
import { PromptOverridesTabPanel } from "./PromptOverridesTabPanel";
import { TabList, TabPanel } from "./TabList";
import { TaskLabelOverridesTabPanel } from "./TaskLabelOverridesPanel";
import { TitleDescriptionFields } from "./TitleDescriptionFields";
import { DiscardChangesDialog, SaveButton } from "./UnsavedChanges";
import { useCloseOnEscape } from "./useCloseOnEscape";

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

type FeatureTab = "details" | "prompts" | "labels";

type PromptStateChange = (
  values: PromptValues,
  ready: boolean,
  invalid: boolean,
) => void;
type TaskLabelStateChange = (
  update: TaskLabelOverridesUpdate,
  ready: boolean,
  invalid: boolean,
) => void;

// 確認が送信の代わりになるため、送信時点のフォームの値は確認かキャンセルまで
// 保持する。
type FeatureUpdate = Parameters<typeof mutations.updateFeature>[0];

// eslint-disable-next-line max-lines-per-function -- 下書きと確認ダイアログを同じ編集フローで管理するため。
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
  const [activeTab, setActiveTab] = useState<FeatureTab>("details");
  const [promptsMounted, setPromptsMounted] = useState(false);
  const [labelsMounted, setLabelsMounted] = useState(false);
  const [promptValues, setPromptValues] = useState<PromptValues>(() =>
    promptValuesOf(feature.promptOverrides),
  );
  const [promptReady, setPromptReady] = useState(false);
  const [promptInvalid, setPromptInvalid] = useState(false);
  const [taskLabelUpdate, setTaskLabelUpdate] =
    useState<TaskLabelOverridesUpdate>(() =>
      create(TaskLabelOverridesUpdateSchema),
    );
  const [taskLabelReady, setTaskLabelReady] = useState(false);
  const [taskLabelInvalid, setTaskLabelInvalid] = useState(false);
  const updateFeature = useDomainMutation(mutations.updateFeature);
  const deleteFeature = useDomainMutation(mutations.deleteFeature);
  const unfinished = unfinishedTaskCount(feature);
  const dirty =
    draft.title !== feature.title ||
    draft.description !== feature.description ||
    draft.status !== feature.status ||
    draft.projectId !== feature.projectId ||
    (promptReady &&
      promptValuesChanged(
        promptValues,
        promptValuesOf(feature.promptOverrides),
      ));
  const taskLabelsDirty =
    taskLabelReady && Object.keys(taskLabelUpdate.values).length > 0;

  function openTab(tab: FeatureTab) {
    setActiveTab(tab);
    if (tab === "prompts") setPromptsMounted(true);
    if (tab === "labels") setLabelsMounted(true);
  }

  function requestClose() {
    if (dirty || taskLabelsDirty) setConfirmation("discard");
    else onClose();
  }

  useCloseOnEscape(requestClose);

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
    const promptOverrides = promptReady
      ? changedPromptValues(
          promptValues,
          promptValuesOf(feature.promptOverrides),
        )
      : undefined;
    const taskLabelOverrides = taskLabelsDirty ? taskLabelUpdate : undefined;
    const update: FeatureUpdate = {
      id: feature.id,
      ...draft,
      ...(promptOverrides ? { promptOverrides } : {}),
      ...(taskLabelOverrides ? { taskLabelOverrides } : {}),
    };
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
          taskLabelDirty={taskLabelsDirty}
          onDraftChange={setDraft}
          updatePending={updateFeature.isPending}
          deletePending={deleteFeature.isPending}
          updateError={updateFeature.error}
          activeTab={activeTab}
          promptsMounted={promptsMounted}
          labelsMounted={labelsMounted}
          promptProjectId={draft.projectId}
          promptInvalid={promptInvalid}
          taskLabelInvalid={taskLabelInvalid}
          onTab={openTab}
          onPromptStateChange={(values, ready, invalid) => {
            setPromptValues(values);
            setPromptReady(ready);
            setPromptInvalid(invalid);
          }}
          onTaskLabelStateChange={(update, ready, invalid) => {
            setTaskLabelUpdate(update);
            setTaskLabelReady(ready);
            setTaskLabelInvalid(invalid);
          }}
          onSubmit={submitFeature}
          onClose={requestClose}
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
  taskLabelDirty: boolean;
  onDraftChange: (next: FeatureDraft) => void;
  updatePending: boolean;
  deletePending: boolean;
  updateError: Error | null;
  activeTab: FeatureTab;
  promptsMounted: boolean;
  labelsMounted: boolean;
  promptProjectId: string;
  promptInvalid: boolean;
  taskLabelInvalid: boolean;
  onTab: (tab: FeatureTab) => void;
  onPromptStateChange: PromptStateChange;
  onTaskLabelStateChange: TaskLabelStateChange;
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
  activeTab,
  promptsMounted,
  labelsMounted,
  promptProjectId,
  onTab,
  onPromptStateChange,
  onTaskLabelStateChange,
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
      className="dialog feature-management-dialog entity-edit-dialog"
      role="dialog"
      aria-modal="true"
      aria-label={t("featureEdit.manageLabel")}
    >
      <header>
        <p className="section-label">{t("featureEdit.archivedEyebrow")}</p>
        <h2>{t("featureEdit.manageTitle")}</h2>
      </header>
      <TabList
        tabs={featureTabs(t)}
        active={activeTab}
        onSelect={onTab}
        idPrefix="feature-edit"
        className="settings-tabs entity-edit-tabs"
        tabClassName="settings-tab"
      />
      <TabPanel
        active={activeTab === "details"}
        className="entity-edit-tab-panel"
        idPrefix="feature-edit"
        tab="details"
      >
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
      </TabPanel>
      <PromptOverridesTabPanel
        active={activeTab === "prompts"}
        idPrefix="feature-edit"
        promptsMounted={promptsMounted}
        scope="feature"
        overrides={feature.promptOverrides}
        parentOverrides={
          projects.find((item) => item.id === promptProjectId)?.promptOverrides
        }
        editable={false}
        onStateChange={onPromptStateChange}
      />
      <TaskLabelOverridesTabPanel
        active={activeTab === "labels"}
        idPrefix="feature-edit"
        labelsMounted={labelsMounted}
        scope="feature"
        overrides={feature.taskLabelOverrides}
        parentOverrides={
          projects.find((item) => item.id === promptProjectId)
            ?.taskLabelOverrides
        }
        editable={false}
        onStateChange={onTaskLabelStateChange}
      />
      <MutationError error={updateError} />
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

// eslint-disable-next-line max-lines-per-function -- 詳細・プロンプト・ラベルを同じフォームで保存するため。
function ActiveFeatureDialog({
  feature,
  projects,
  draft,
  dirty,
  taskLabelDirty,
  onDraftChange,
  updatePending,
  deletePending,
  updateError,
  onSubmit,
  onClose,
  onArchive,
  onDelete,
  activeTab,
  promptsMounted,
  labelsMounted,
  promptProjectId,
  promptInvalid,
  taskLabelInvalid,
  onTab,
  onPromptStateChange,
  onTaskLabelStateChange,
}: FeatureDialogContentProps) {
  const { t } = useTranslation();
  const labels = useFeatureLifecycleLabels(false);
  return (
    <form
      className="dialog feature-management-dialog entity-edit-dialog"
      onSubmit={onSubmit}
      aria-label={t("featureEdit.formLabel")}
    >
      <header>
        <h2>{t("featureEdit.title")}</h2>
      </header>
      <TabList
        tabs={featureTabs(t)}
        active={activeTab}
        onSelect={onTab}
        idPrefix="feature-edit"
        className="settings-tabs entity-edit-tabs"
        tabClassName="settings-tab"
      />
      <TabPanel
        active={activeTab === "details"}
        className="entity-edit-tab-panel"
        idPrefix="feature-edit"
        tab="details"
      >
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
        <LifecycleActions
          labels={labels}
          updatePending={updatePending}
          deletePending={deletePending}
          onArchive={onArchive}
          onDelete={onDelete}
        />
      </TabPanel>
      <PromptOverridesTabPanel
        active={activeTab === "prompts"}
        idPrefix="feature-edit"
        promptsMounted={promptsMounted}
        scope="feature"
        overrides={feature.promptOverrides}
        parentOverrides={
          projects.find((item) => item.id === promptProjectId)?.promptOverrides
        }
        editable
        onStateChange={onPromptStateChange}
      />
      <TaskLabelOverridesTabPanel
        active={activeTab === "labels"}
        idPrefix="feature-edit"
        labelsMounted={labelsMounted}
        scope="feature"
        overrides={feature.taskLabelOverrides}
        parentOverrides={
          projects.find((item) => item.id === promptProjectId)
            ?.taskLabelOverrides
        }
        editable
        onStateChange={onTaskLabelStateChange}
      />
      <MutationError error={updateError} />
      <footer>
        <IconButton
          icon={X}
          label={t("common.cancel")}
          variant="secondary"
          onClick={onClose}
        />
        <SaveButton
          dirty={
            (dirty || taskLabelDirty) && !promptInvalid && !taskLabelInvalid
          }
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

function featureTabs(t: TFunction) {
  return [
    { id: "details" as const, label: t("featureEdit.tabs.details") },
    { id: "prompts" as const, label: t("featureEdit.tabs.prompts") },
    { id: "labels" as const, label: t("featureEdit.tabs.labels") },
  ];
}
