import { Link, Unlink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import type { PullRequest } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import {
  pullRequestDisplayStateLabel,
  pullRequestDisplayStateToken,
} from "../i18n/domain";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";

interface PullRequestSectionProps {
  taskId: string;
  pullRequest: PullRequest | undefined;
}

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
            {pullRequest.host && pullRequest.host !== "github.com"
              ? `${pullRequest.host}/`
              : ""}
            {pullRequest.owner}/{pullRequest.repository} #
            {String(pullRequest.number)}
          </a>
          <span>
            {pullRequestDisplayStateLabel(pullRequest.displayState, t)}
          </span>
          {pullRequest.stale && (
            <small className="pr-stale">{t("inspector.stale")}</small>
          )}
          {pullRequest.syncError && (
            <p className="sync-error">
              <strong>{t("inspector.githubSyncError")}</strong>
              {pullRequest.syncError}
            </p>
          )}
          <IconButton
            icon={Unlink}
            label={t("inspector.detach")}
            variant="quiet"
            size="compact"
            iconOnly
            className="text-action"
            onClick={() => {
              detach.mutate(taskId);
            }}
          />
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
          <IconButton
            icon={Link}
            label={t("inspector.attach")}
            variant="primary"
            type="submit"
          />
        </form>
      )}
      <MutationError error={attach.error} />
      <MutationError error={detach.error} />
    </section>
  );
}
