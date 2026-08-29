import { useTranslation } from "react-i18next";
import type { Task } from "../gen/prx/v1/prx_pb";
import {
  blockedReasonLabel,
  taskDisplayStateLabel,
  taskDisplayStateToken,
} from "../i18n/domain";

interface TaskInspectorHeaderProps {
  task: Task;
  tasks: Task[];
  onClose: () => void;
}

export function TaskInspectorHeader({
  task,
  tasks,
  onClose,
}: TaskInspectorHeaderProps) {
  const { t } = useTranslation();
  return (
    <>
      <header>
        <h2>{task.title}</h2>
        <button
          className="icon-button"
          aria-label={t("inspector.close")}
          onClick={onClose}
        >
          ×
        </button>
      </header>
      <div
        className={`inspector-state state-${taskDisplayStateToken(task.displayState)}`}
      >
        <i />
        {taskDisplayStateLabel(task.displayState, t)}
        {task.blockedReason && (
          <small>
            {blockedReasonLabel(
              task.blockedReason,
              (id) => tasks.find((item) => item.id === id)?.title,
              t,
            )}
          </small>
        )}
      </div>
    </>
  );
}
