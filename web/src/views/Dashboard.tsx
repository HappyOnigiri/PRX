import { useTranslation } from "react-i18next";
import { useSnapshot } from "../hooks";
import { formatError } from "../i18n/domain";
import { DashboardQueues } from "./DashboardQueues";
import { DashboardStateMessage } from "./DashboardStateMessage";
import { FeatureBoard } from "./FeatureBoard";
import { ReadyTaskBoard } from "./ReadyTaskBoard";

export function Dashboard() {
  const { t } = useTranslation();
  const { data, isPending, error, refetch } = useSnapshot();
  if (isPending)
    return (
      <DashboardStateMessage
        title={t("dashboard.loadingTitle")}
        detail={t("dashboard.loadingDetail")}
      />
    );
  if (error)
    return (
      <DashboardStateMessage
        title={t("dashboard.errorTitle")}
        detail={formatError(error, t)}
        action={() => void refetch()}
      />
    );
  return (
    <div className="dashboard">
      <header className="page-head">
        <div>
          <h1>
            {t("dashboard.titleStart")}
            <em>{t("dashboard.titleEmphasis")}</em>
          </h1>
          <p>{t("dashboard.description")}</p>
        </div>
        <div className="clock">
          <span>{data.tasks.length}</span>
          <small>{t("dashboard.nodesUnderControl")}</small>
        </div>
      </header>
      <DashboardQueues data={data} />
      <div className="dashboard-grid">
        <ReadyTaskBoard tasks={data.readyTasks} features={data.features} />
        <FeatureBoard features={data.features} />
      </div>
    </div>
  );
}
