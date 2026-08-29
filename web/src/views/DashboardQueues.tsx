import { useTranslation } from "react-i18next";
import type { Snapshot } from "../gen/prx/v1/prx_pb";

const queueNames = [
  [
    "readyTasks",
    "dashboard.queues.ready.title",
    "dashboard.queues.ready.detail",
  ],
  [
    "reviewWaitingTasks",
    "dashboard.queues.review.title",
    "dashboard.queues.review.detail",
  ],
  [
    "conflictTasks",
    "dashboard.queues.conflicts.title",
    "dashboard.queues.conflicts.detail",
  ],
  [
    "staleTasks",
    "dashboard.queues.stale.title",
    "dashboard.queues.stale.detail",
  ],
] as const;

interface DashboardQueuesProps {
  data: Snapshot;
}

export function DashboardQueues({ data }: DashboardQueuesProps) {
  const { t } = useTranslation();
  return (
    <section className="queue-strip" aria-label={t("dashboard.roadmapStatus")}>
      {queueNames.map(([key, title, detail]) => (
        <article key={key} className={`queue-meter ${key}`}>
          <span>{data[key].length}</span>
          <div>
            <h2>{t(title)}</h2>
            <p>{t(detail)}</p>
          </div>
        </article>
      ))}
    </section>
  );
}
