import type { ReactElement } from "react";
import { useTranslation } from "react-i18next";
import type { Snapshot } from "../gen/prx/v1/prx_pb";
import { useSnapshot } from "../hooks";
import { formatError } from "../i18n/domain";
import { StateMessage } from "./Dashboard";

// The project list reports two interruptions while it reads the snapshot, and
// keeping the wording and the retry here leaves the page about the list itself.
type ProjectSnapshotState =
  | { message: ReactElement; data?: undefined }
  | { message?: undefined; data: Snapshot };

export function useProjectSnapshot(): ProjectSnapshotState {
  const { t } = useTranslation();
  const { data, isPending, error, refetch } = useSnapshot();
  if (isPending)
    return {
      message: (
        <StateMessage
          title={t("project.loadingTitle")}
          detail={t("project.loadingDetail")}
        />
      ),
    };
  if (error)
    return {
      message: (
        <StateMessage
          title={t("project.errorTitle")}
          detail={formatError(error, t)}
          action={() => void refetch()}
        />
      ),
    };
  return { data };
}
