import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { Dependency, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { MutationError } from "./MutationError";
import { DependencySection } from "./TaskInspectorDependencies";
import { TaskInspectorForm } from "./TaskInspectorForm";
import { TaskInspectorHeader } from "./TaskInspectorHeader";
import { PullRequestSection } from "./TaskInspectorPullRequest";
import { ReferencesSection } from "./TaskInspectorReferences";
import type { TaskNodeDocument } from "./TaskNode";

export interface TaskInspectorProps {
  task: Task;
  tasks: Task[];
  dependencies: Dependency[];
  pullRequest: PullRequest | undefined;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  onClose: () => void;
}

export function TaskInspector({
  task,
  tasks,
  dependencies,
  pullRequest,
  documents,
  onPreview,
  onClose,
}: TaskInspectorProps) {
  const { t } = useTranslation();
  const deleteTask = useDomainMutation(mutations.deleteTask);

  return (
    <aside className="inspector" aria-label={t("inspector.label")}>
      <TaskInspectorHeader task={task} tasks={tasks} onClose={onClose} />
      <TaskInspectorForm task={task} />
      <PullRequestSection taskId={task.id} pullRequest={pullRequest} />
      <DependencySection
        taskId={task.id}
        tasks={tasks}
        dependencies={dependencies}
      />
      <ReferencesSection
        taskId={task.id}
        documents={documents}
        onPreview={onPreview}
      />
      <button
        className="danger-zone"
        onClick={() => {
          if (
            !window.confirm(
              t("inspector.deleteTaskConfirm", { title: task.title }),
            )
          )
            return;
          void deleteTask.mutateAsync(task.id).then(onClose, () => undefined);
        }}
      >
        {t("inspector.deleteTask")}
      </button>
      <MutationError error={deleteTask.error} />
    </aside>
  );
}
