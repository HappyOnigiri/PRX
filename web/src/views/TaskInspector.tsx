import { Trash2, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { type PullRequest, type Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { blockedReasonLabel } from "../i18n/domain";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { EntityIcon } from "./EntityIcon";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { useTaskDraft } from "./taskDraft";
import { PullRequestSection } from "./TaskInspectorPullRequest";
import { ReferencesSection } from "./TaskInspectorReferences";
import { TaskInspectorTaskForm } from "./TaskInspectorTaskForm";
import { type TaskNodeDocument } from "./TaskNode";
import { TaskBlockLabels, TaskStatusBadge } from "./TaskStateBadges";
import { DiscardChangesDialog, SaveButton } from "./UnsavedChanges";

export interface TaskInspectorProps {
  task: Task;
  tasks: Task[];
  pullRequest: PullRequest | undefined;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  onClose: () => void;
  readOnly?: boolean;
}

function TaskInspectorBlocks({ task, tasks }: { task: Task; tasks: Task[] }) {
  const { t } = useTranslation();
  const detail = blockedReasonLabel(
    task.blockedReason,
    (id) => tasks.find((item) => item.id === id)?.title,
    t,
  );
  return (
    <p className="inspector-blocks">
      <TaskBlockLabels labels={task.blockLabels} dependencyDetail={detail} />
      {detail && <span className="inspector-blocks-detail">{detail}</span>}
    </p>
  );
}

function TaskInspectorHeader({
  task,
  onClose,
}: Pick<TaskInspectorProps, "task"> & { onClose: () => void }) {
  const { t } = useTranslation();
  return (
    <header>
      <div className="inspector-heading">
        {/* どのカードや行とも同じく、状態が見出しの先頭に来る。 */}
        <div className="inspector-heading-line">
          <TaskStatusBadge state={task.displayState} />
          <h2>
            <EntityIcon kind="task" size={16} />
            {task.title}
          </h2>
        </div>
        <CopyableIdentifier
          label={t("common.taskId")}
          value={task.id}
          valueOnly
        />
      </div>
      <IconButton
        icon={X}
        label={t("inspector.close")}
        variant="secondary"
        iconOnly
        onClick={onClose}
      />
    </header>
  );
}

export function TaskInspector({
  task,
  tasks,
  pullRequest,
  documents,
  onPreview,
  onClose,
  readOnly = false,
}: TaskInspectorProps) {
  const { t } = useTranslation();
  const deleteTask = useDomainMutation(mutations.deleteTask);
  const controller = useTaskDraft(task);
  const [discarding, setDiscarding] = useState(false);
  const dirty = !readOnly && controller.dirty;

  return (
    <aside className="inspector" aria-label={t("inspector.label")}>
      <TaskInspectorHeader
        task={task}
        onClose={() => {
          if (dirty) setDiscarding(true);
          else onClose();
        }}
      />
      {/* ステータスは見出しにあるので、このストリップには進行を妨げている事情
          だけが並ぶ。待ち相手はラベルの語だけでは特定できないため、可視の
          テキストとしても添える。 */}
      {task.blockLabels.length > 0 && (
        <TaskInspectorBlocks task={task} tasks={tasks} />
      )}
      {readOnly && (
        <p className="inspector-read-only">{t("inspector.readOnly")}</p>
      )}
      <TaskInspectorTaskForm
        task={task}
        controller={controller}
        readOnly={readOnly}
      />
      <PullRequestSection
        taskId={task.id}
        pullRequest={pullRequest}
        readOnly={readOnly}
      />
      <ReferencesSection
        documents={documents}
        onPreview={onPreview}
        readOnly={readOnly}
      />
      {!readOnly && (
        <>
          {/* 削除は task 自体を消す操作なので、下書きの保存とは別に置く。 */}
          <IconButton
            icon={Trash2}
            label={t("inspector.deleteTask")}
            variant="danger"
            className="danger-zone"
            onClick={() => {
              if (
                !window.confirm(
                  t("inspector.deleteTaskConfirm", { title: task.title }),
                )
              )
                return;
              void deleteTask
                .mutateAsync(task.id)
                .then(onClose, () => undefined);
            }}
          />
          <MutationError error={deleteTask.error} />
          <footer className="inspector-footer">
            <SaveButton
              dirty={dirty}
              label={t("inspector.saveTask")}
              pending={controller.pending}
              onClick={controller.save}
            />
          </footer>
        </>
      )}
      {discarding && (
        <DiscardChangesDialog
          onCancel={() => {
            setDiscarding(false);
          }}
          onConfirm={onClose}
        />
      )}
    </aside>
  );
}
