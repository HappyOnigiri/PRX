import { Check, Copy, RotateCcw } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { formatBrowserDebugSection } from "../debug-text";
import type { DebugProblem } from "../gen/prx/v1/prx_pb";
import { useDebugReport, useQueryDiagnostics } from "../hooks";
import { debugProblemLabel, formatError } from "../i18n/domain";
import { IconButton } from "./IconButton";

const copyStatusResetMs = 1600;

export function DebugSettingsPanel() {
  const { t } = useTranslation();
  const report = useDebugReport(true);
  const queries = useQueryDiagnostics();
  const [copyStatus, setCopyStatus] = useState<"copied" | "failed">();

  async function copy(text: string) {
    try {
      // The server text is copied verbatim so a report pasted from the browser
      // is the same one `prx debug` prints, with this tab's own facts appended.
      await navigator.clipboard.writeText(
        text + formatBrowserDebugSection(queries),
      );
      setCopyStatus("copied");
    } catch {
      setCopyStatus("failed");
    }
    window.setTimeout(() => {
      setCopyStatus(undefined);
    }, copyStatusResetMs);
  }

  if (report.isError)
    return (
      <p className="settings-panel-state" role="alert">
        {formatError(report.error, t)}
      </p>
    );
  if (!report.data)
    return (
      <p className="settings-panel-state">{t("settings.debug.loading")}</p>
    );

  const generatedAt = report.data.report.runtime?.generatedAt ?? "";
  return (
    <>
      <p className="dialog-lead">{t("settings.debug.description")}</p>
      <div className="settings-debug-actions">
        <span className="settings-debug-taken">
          {t("settings.debug.generatedAt", { time: generatedAt })}
        </span>
        <IconButton
          icon={RotateCcw}
          label={t("settings.debug.reload")}
          variant="secondary"
          onClick={() => void report.refetch()}
        />
        <IconButton
          icon={copyStatus === "copied" ? Check : Copy}
          label={t("settings.debug.copy")}
          variant="secondary"
          onClick={() => void copy(report.data.text)}
        />
      </div>
      <p className="settings-debug-note" aria-live="polite">
        {copyStatus === "copied" && t("common.copied")}
        {copyStatus === "failed" && t("common.copyFailed")}
        {copyStatus === undefined && t("settings.debug.copyNote")}
      </p>
      <DebugProblems problems={report.data.report.problems} />
      <h3 className="settings-debug-heading">{t("settings.debug.report")}</h3>
      <pre className="settings-debug-text">{report.data.text}</pre>
    </>
  );
}

function DebugProblems({ problems }: { problems: DebugProblem[] }) {
  const { t } = useTranslation();
  return (
    <section>
      <h3 className="settings-debug-heading">{t("settings.debug.problems")}</h3>
      {problems.length === 0 ? (
        <p className="settings-empty">{t("settings.debug.noProblems")}</p>
      ) : (
        <ul className="settings-debug-problems">
          {problems.map((problem, index) => (
            <li
              className="settings-debug-problem"
              key={`${String(problem.code)}-${String(index)}`}
            >
              <strong>{debugProblemLabel(problem.code, t)}</strong>
              {problem.target && (
                <code className="settings-debug-target">{problem.target}</code>
              )}
              {problem.evidence && (
                <small>
                  {t("settings.debug.evidence")}: {problem.evidence}
                </small>
              )}
              {problem.nextCommand && (
                <small>
                  {t("settings.debug.next")}: <code>{problem.nextCommand}</code>
                </small>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
