import { ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";
import {
  BlockedReasonCode,
  DebugProblemCode,
  DocumentKind,
  DomainErrorCode,
  ErrorDetailSchema,
  FeatureStatus,
  PullRequestDisplayState,
  TaskDisplayState,
  TaskKind,
  TaskStatus,
  type BlockedReason,
} from "../gen/prx/v1/prx_pb";

export function featureStatusLabel(value: FeatureStatus, t: TFunction): string {
  return t(featureStatusKeys[value]);
}

export function taskStatusLabel(value: TaskStatus, t: TFunction): string {
  return t(taskStatusKeys[value]);
}

export const featureStatusKeys = {
  [FeatureStatus.AUTO]: "featureStatus.auto",
  [FeatureStatus.ACTIVE]: "featureStatus.active",
  [FeatureStatus.PAUSED]: "featureStatus.paused",
  [FeatureStatus.COMPLETED]: "featureStatus.completed",
  [FeatureStatus.CANCELLED]: "featureStatus.cancelled",
  [FeatureStatus.UNSPECIFIED]: "featureStatus.unknown",
} as const satisfies Record<FeatureStatus, string>;

export const taskStatusKeys = {
  [TaskStatus.AUTO]: "taskStatus.auto",
  [TaskStatus.NOT_STARTED]: "taskStatus.notStarted",
  [TaskStatus.IN_PROGRESS]: "taskStatus.inProgress",
  [TaskStatus.COMPLETED]: "taskStatus.completed",
  [TaskStatus.CLOSED]: "taskStatus.closed",
  [TaskStatus.UNSPECIFIED]: "taskStatus.unknown",
} as const satisfies Record<TaskStatus, string>;

export const taskKindKeys = {
  [TaskKind.UNSPECIFIED]: "kind.unknown",
  [TaskKind.PULL_REQUEST]: "kind.pullRequest",
  [TaskKind.MANUAL]: "kind.manual",
} as const satisfies Record<TaskKind, string>;

export function taskKindLabel(value: TaskKind, t: TFunction): string {
  return t(taskKindKeys[value]);
}

export const documentKindKeys = {
  [DocumentKind.UNSPECIFIED]: "documentKind.unknown",
  [DocumentKind.URL]: "documentKind.url",
  [DocumentKind.LOCAL_FILE]: "documentKind.localFile",
  [DocumentKind.MARKDOWN]: "documentKind.markdown",
} as const satisfies Record<DocumentKind, string>;

export function documentKindLabel(value: DocumentKind, t: TFunction): string {
  return t(documentKindKeys[value]);
}

export const displayStateKeys = {
  [TaskDisplayState.UNSPECIFIED]: "displayState.unknown",
  [TaskDisplayState.NOT_STARTED]: "displayState.notStarted",
  [TaskDisplayState.IN_PROGRESS]: "displayState.inProgress",
  [TaskDisplayState.COMPLETED]: "displayState.completed",
  [TaskDisplayState.CLOSED]: "displayState.closed",
  [TaskDisplayState.MERGED]: "displayState.merged",
  [TaskDisplayState.DRAFT]: "displayState.draft",
  [TaskDisplayState.CONFLICT]: "displayState.conflict",
  [TaskDisplayState.CHANGES_REQUESTED]: "displayState.changesRequested",
  [TaskDisplayState.APPROVED]: "displayState.approved",
  [TaskDisplayState.REVIEW_WAITING]: "displayState.reviewWaiting",
  [TaskDisplayState.OPEN]: "displayState.open",
  [TaskDisplayState.UNKNOWN]: "displayState.unknown",
} as const satisfies Record<TaskDisplayState, string>;

export function taskDisplayStateLabel(
  value: TaskDisplayState,
  t: TFunction,
): string {
  return t(displayStateKeys[value]);
}

export function taskDisplayStateToken(value: TaskDisplayState): string {
  return TaskDisplayState[value].toLowerCase().replaceAll("_", "-");
}

export function pullRequestDisplayStateLabel(
  value: PullRequestDisplayState,
  t: TFunction,
): string {
  return t(pullRequestDisplayStateKeys[value]);
}

export function pullRequestDisplayStateToken(
  value: PullRequestDisplayState,
): string {
  return PullRequestDisplayState[value].toLowerCase().replaceAll("_", "-");
}

export const pullRequestDisplayStateKeys = {
  [PullRequestDisplayState.UNSPECIFIED]: "displayState.unknown",
  [PullRequestDisplayState.MERGED]: "displayState.merged",
  [PullRequestDisplayState.CLOSED]: "displayState.closed",
  [PullRequestDisplayState.DRAFT]: "displayState.draft",
  [PullRequestDisplayState.CONFLICT]: "displayState.conflict",
  [PullRequestDisplayState.CHANGES_REQUESTED]: "displayState.changesRequested",
  [PullRequestDisplayState.APPROVED]: "displayState.approved",
  [PullRequestDisplayState.REVIEW_WAITING]: "displayState.reviewWaiting",
  [PullRequestDisplayState.OPEN]: "displayState.open",
  [PullRequestDisplayState.UNKNOWN]: "displayState.unknown",
} as const satisfies Record<PullRequestDisplayState, string>;

export const blockedReasonKeys = {
  [BlockedReasonCode.UNSPECIFIED]: "blockedReason.unknown",
  [BlockedReasonCode.DEPENDENCY_DATA_INCOMPLETE]:
    "blockedReason.dependencyDataIncomplete",
  [BlockedReasonCode.WAITING_FOR_BLOCKER]: "blockedReason.waitingForBlocker",
} as const satisfies Record<BlockedReasonCode, string>;

export function blockedReasonLabel(
  reason: BlockedReason | undefined,
  taskTitle: (id: string) => string | undefined,
  t: TFunction,
): string {
  if (!reason) return "";
  const title =
    taskTitle(reason.blockerTaskId) ?? t("blockedReason.unknownBlocker");
  if (reason.code === BlockedReasonCode.WAITING_FOR_BLOCKER)
    return t(blockedReasonKeys[BlockedReasonCode.WAITING_FOR_BLOCKER], {
      title,
    });
  return t(blockedReasonKeys[reason.code]);
}

export const debugProblemKeys = {
  [DebugProblemCode.UNSPECIFIED]: "debugProblem.unknown",
  [DebugProblemCode.STORAGE_UNAVAILABLE]: "debugProblem.storageUnavailable",
  [DebugProblemCode.SCHEMA_VERSION_AHEAD_OF_BINARY]:
    "debugProblem.schemaVersionAheadOfBinary",
  [DebugProblemCode.DATABASE_NOT_WRITABLE]: "debugProblem.databaseNotWritable",
  [DebugProblemCode.DATABASE_INTEGRITY_ERRORS]:
    "debugProblem.databaseIntegrityErrors",
  [DebugProblemCode.CONFIG_UNREADABLE]: "debugProblem.configUnreadable",
  [DebugProblemCode.CONFIG_PERMISSIONS_TOO_OPEN]:
    "debugProblem.configPermissionsTooOpen",
  [DebugProblemCode.CONFIG_UNKNOWN_FIELDS]: "debugProblem.configUnknownFields",
  [DebugProblemCode.NO_AUTH_METHOD_FOR_HOST]:
    "debugProblem.noAuthMethodForHost",
  [DebugProblemCode.GITHUB_SYNC_RUN_ERROR]: "debugProblem.githubSyncRunError",
  [DebugProblemCode.GITHUB_SYNC_OVERDUE]: "debugProblem.githubSyncOverdue",
  [DebugProblemCode.GITHUB_SYNC_NEVER_COMPLETED]:
    "debugProblem.githubSyncNeverCompleted",
  [DebugProblemCode.PULL_REQUESTS_STALE]: "debugProblem.pullRequestsStale",
} as const satisfies Record<DebugProblemCode, string>;

// A report from a newer server may carry a code this bundle does not know, so
// an unmapped value falls back to the generic label instead of rendering a
// missing translation key.
export function debugProblemLabel(
  value: DebugProblemCode,
  t: TFunction,
): string {
  const known =
    value in debugProblemKeys ? value : DebugProblemCode.UNSPECIFIED;
  return t(debugProblemKeys[known]);
}

export const errorKeys = {
  [DomainErrorCode.UNSPECIFIED]: "error.unknown",
  [DomainErrorCode.CROSS_FEATURE_DEPENDENCY]: "error.crossFeatureDependency",
  [DomainErrorCode.CYCLE]: "error.cycle",
  [DomainErrorCode.DOCUMENT_READ_FAILED]: "error.documentReadFailed",
  [DomainErrorCode.DOCUMENT_TOO_LARGE]: "error.documentTooLarge",
  [DomainErrorCode.DOCUMENT_NOT_TEXT]: "error.documentNotText",
  [DomainErrorCode.DUPLICATE_DEPENDENCY]: "error.duplicateDependency",
  [DomainErrorCode.DUPLICATE_PULL_REQUEST]: "error.duplicatePullRequest",
  [DomainErrorCode.DUPLICATE_IMPLEMENTATION_PLAN]:
    "error.duplicateImplementationPlan",
  [DomainErrorCode.GITHUB_AUTH]: "error.githubAuth",
  [DomainErrorCode.INVALID_CONFIG]: "error.invalidConfig",
  [DomainErrorCode.INVALID_DATABASE]: "error.invalidDatabase",
  [DomainErrorCode.INVALID_DOCUMENT]: "error.invalidDocument",
  [DomainErrorCode.INVALID_DOCUMENT_KIND]: "error.invalidDocumentKind",
  [DomainErrorCode.INVALID_DOCUMENT_URL]: "error.invalidDocumentUrl",
  [DomainErrorCode.INVALID_IMPLEMENTATION_PLAN]:
    "error.invalidImplementationPlan",
  [DomainErrorCode.IMPLEMENTATION_PLAN_TOO_LARGE]:
    "error.implementationPlanTooLarge",
  [DomainErrorCode.INVALID_KIND]: "error.invalidKind",
  [DomainErrorCode.INVALID_PARENT]: "error.invalidParent",
  [DomainErrorCode.INVALID_PULL_REQUEST_URL]: "error.invalidPullRequestUrl",
  [DomainErrorCode.INVALID_SLUG]: "error.invalidSlug",
  [DomainErrorCode.INVALID_STATUS]: "error.invalidStatus",
  [DomainErrorCode.INVALID_TITLE]: "error.invalidTitle",
  [DomainErrorCode.NOT_FOUND]: "error.notFound",
  [DomainErrorCode.PULL_REQUEST_ON_MANUAL_TASK]:
    "error.pullRequestOnManualTask",
  [DomainErrorCode.REFERENCES_EXIST]: "error.referencesExist",
} as const satisfies Record<DomainErrorCode, string>;

// The cycle path arrives as task IDs, so callers that know the tasks on screen
// pass a resolver to show titles instead of raw identifiers.
export function formatError(
  error: Error,
  t: TFunction,
  taskTitle?: (id: string) => string | undefined,
): string {
  const connectError = ConnectError.from(error);
  const detail = connectError.findDetails(ErrorDetailSchema)[0];
  if (!detail || detail.code === DomainErrorCode.UNSPECIFIED)
    return connectError.rawMessage;
  if (detail.code === DomainErrorCode.CYCLE)
    return t("error.cycle", {
      path: detail.path.map((id) => taskTitle?.(id) ?? id).join(" → "),
    });
  const key = errorKeys[detail.code];
  return t(key);
}
