import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { PullRequest } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { formValue } from "../form";
import {
  pullRequestDisplayStateLabel,
  pullRequestDisplayStateToken,
} from "../i18n/domain";
import { MutationError } from "./MutationError";

type PullRequestSectionProps = {
  taskId: string;
  pullRequest?: PullRequest;
};

export function PullRequestSection({
  taskId,
  pullRequest,
}: PullRequestSectionProps) {
  const { t } = useTranslation();
  const attach = useDomainMutation(
    ({ taskId: id, url }: { taskId: string; url: string }) =>
      mutations.attachPR(id, url),
  );
  const detach = useDomainMutation(mutations.detachPR);

  return (
    <section>
      <h3>{t("inspector.pullRequest")}</h3>
      {pullRequest ? (
        <div
          className={`linked-pr state-${pullRequestDisplayStateToken(pullRequest.displayState)} ${pullRequest.stale ? "is-stale" : ""}`}
        >
          <a href={pullRequest.url} target="_blank" rel="noreferrer">
            {pullRequest.owner}/{pullRequest.repository} #
            {String(pullRequest.number)}
          </a>
          <span>
            {pullRequest.stale
              ? t("inspector.stale")
              : pullRequestDisplayStateLabel(pullRequest.displayState, t)}
          </span>
          {pullRequest.syncError && <p>{pullRequest.syncError}</p>}
          <button className="text-action" onClick={() => detach.mutate(taskId)}>
            {t("inspector.detach")}
          </button>
        </div>
      ) : (
        <form
          className="inline-form"
          onSubmit={(event) => {
            event.preventDefault();
            attach.mutate({
              taskId,
              url: formValue(new FormData(event.currentTarget), "url"),
            });
          }}
        >
          <input
            name="url"
            required
            placeholder="https://github.com/org/repo/pull/42"
          />
          <button>{t("inspector.attach")}</button>
        </form>
      )}
      <MutationError error={attach.error} />
      <MutationError error={detach.error} />
    </section>
  );
}
