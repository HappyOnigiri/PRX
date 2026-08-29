import { useTranslation } from "react-i18next";

interface FeatureGraphStateProps {
  taskCount: number;
  layoutError: { message: string | undefined } | undefined;
  onCreateTask: () => void;
  onRetryLayout: () => void;
}

export function FeatureGraphState({
  taskCount,
  layoutError,
  onCreateTask,
  onRetryLayout,
}: FeatureGraphStateProps) {
  const { t } = useTranslation();
  return (
    <>
      {taskCount === 0 && !layoutError && (
        <div className="graph-empty">
          <span>＋</span>
          <h2>{t("workspace.graphEmptyTitle")}</h2>
          <p>{t("workspace.graphEmptyDetail")}</p>
          <button onClick={onCreateTask}>{t("workspace.addTaskPlain")}</button>
        </div>
      )}
      {layoutError && (
        <div className="graph-empty" role="alert">
          <span>⚠</span>
          <h2>{t("workspace.layoutErrorTitle")}</h2>
          <p>{layoutError.message ?? t("workspace.layoutErrorFallback")}</p>
          <button onClick={onRetryLayout}>{t("workspace.retryLayout")}</button>
        </div>
      )}
    </>
  );
}
