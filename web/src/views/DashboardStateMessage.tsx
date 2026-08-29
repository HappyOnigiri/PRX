import { useTranslation } from "react-i18next";

interface DashboardStateMessageProps {
  title: string;
  detail: string;
  action?: () => void;
}

export function DashboardStateMessage({
  title,
  detail,
  action,
}: DashboardStateMessageProps) {
  const { t } = useTranslation();
  return (
    <div className="state-message">
      <div className="spinner" />
      <h1>{title}</h1>
      <p>{detail}</p>
      {action && <button onClick={action}>{t("common.retry")}</button>}
    </div>
  );
}
