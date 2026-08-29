import { useTranslation } from "react-i18next";
import type { Snapshot } from "../gen/prx/v1/prx_pb";

const queueNames = [
  [
    "readyTasks",
    "queue-ready",
    "dashboard.queues.ready.title",
    "dashboard.queues.ready.detail",
  ],
  [
    "reviewWaitingTasks",
    "queue-review-waiting",
    "dashboard.queues.review.title",
    "dashboard.queues.review.detail",
  ],
  [
    "conflictTasks",
    "queue-conflict",
    "dashboard.queues.conflicts.title",
    "dashboard.queues.conflicts.detail",
  ],
  [
    "staleTasks",
    "queue-stale",
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
      {queueNames.map(([key, className, title, detail]) => (
        <article key={key} className={`queue-meter ${className}`}>
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
