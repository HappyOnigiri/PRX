import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { Feature, Task } from "../gen/prx/v1/prx_pb";

interface ReadyTaskBoardProps {
  tasks: Task[];
  features: Feature[];
}

export function ReadyTaskBoard({ tasks, features }: ReadyTaskBoardProps) {
  const { t } = useTranslation();
  return (
    <section className="ready-board">
      <header>
        <p className="section-label">{t("dashboard.executionQueue")}</p>
        <h2>{t("dashboard.readyToStart")}</h2>
      </header>
      {tasks.length === 0 ? (
        <div className="empty">
          <span>◇</span>
          <h3>{t("dashboard.noTaskTitle")}</h3>
          <p>{t("dashboard.noTaskDetail")}</p>
        </div>
      ) : (
        <ol>
          {tasks.map((task, index) => {
            const feature = features.find((item) => item.id === task.featureId);
            return (
              <li key={task.id}>
                <span className="queue-index">
                  {String(index + 1).padStart(2, "0")}
                </span>
                <div>
                  <Link
                    to="/features/$featureId"
                    params={{ featureId: task.featureId }}
                  >
                    {task.title}
                  </Link>
                  <p>
                    {feature?.title} · {task.assignee || t("common.unassigned")}
                  </p>
                </div>
                <i>{t("common.ready")}</i>
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}
