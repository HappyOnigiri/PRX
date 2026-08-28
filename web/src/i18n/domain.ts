import { ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";
import {
  BlockedReasonCode,
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
  const keys = {
    [FeatureStatus.ACTIVE]: "featureStatus.active",
    [FeatureStatus.PAUSED]: "featureStatus.paused",
    [FeatureStatus.COMPLETED]: "featureStatus.completed",
    [FeatureStatus.CANCELLED]: "featureStatus.cancelled",
    [FeatureStatus.UNSPECIFIED]: "featureStatus.unknown",
  } as const;
  return t(keys[value] ?? "featureStatus.unknown");
}

export function taskStatusLabel(value: TaskStatus, t: TFunction): string {
  const keys = {
    [TaskStatus.PLANNED]: "taskStatus.planned",
    [TaskStatus.IN_PROGRESS]: "taskStatus.inProgress",
    [TaskStatus.COMPLETED]: "taskStatus.completed",
    [TaskStatus.CANCELLED]: "taskStatus.cancelled",
    [TaskStatus.UNSPECIFIED]: "taskStatus.unknown",
  } as const;
  return t(keys[value] ?? "taskStatus.unknown");
}

export function taskKindLabel(value: TaskKind, t: TFunction): string {
  if (value === TaskKind.PULL_REQUEST) return t("kind.pullRequest");
  if (value === TaskKind.MANUAL) return t("kind.manual");
  return t("kind.unknown");
}

export function documentKindLabel(value: DocumentKind, t: TFunction): string {
  if (value === DocumentKind.URL) return t("documentKind.url");
  if (value === DocumentKind.MARKDOWN_PATH)
    return t("documentKind.markdownPath");
  return t("documentKind.unknown");
}

const displayStateKeys = {
  [TaskDisplayState.UNSPECIFIED]: "displayState.unknown",
  [TaskDisplayState.PLANNED]: "displayState.planned",
  [TaskDisplayState.IN_PROGRESS]: "displayState.inProgress",
  [TaskDisplayState.COMPLETED]: "displayState.completed",
  [TaskDisplayState.CANCELLED]: "displayState.cancelled",
  [TaskDisplayState.UNLINKED]: "displayState.unlinked",
  [TaskDisplayState.MERGED]: "displayState.merged",
  [TaskDisplayState.CLOSED]: "displayState.closed",
  [TaskDisplayState.DRAFT]: "displayState.draft",
  [TaskDisplayState.CONFLICT]: "displayState.conflict",
  [TaskDisplayState.CHANGES_REQUESTED]: "displayState.changesRequested",
  [TaskDisplayState.APPROVED]: "displayState.approved",
  [TaskDisplayState.REVIEW_WAITING]: "displayState.reviewWaiting",
  [TaskDisplayState.OPEN]: "displayState.open",
  [TaskDisplayState.UNKNOWN]: "displayState.unknown",
} as const;

export function taskDisplayStateLabel(
  value: TaskDisplayState,
  t: TFunction,
): string {
  return t(displayStateKeys[value] ?? "displayState.unknown");
}

export function taskDisplayStateToken(value: TaskDisplayState): string {
  return TaskDisplayState[value]?.toLowerCase() ?? "unknown";
}

export function pullRequestDisplayStateLabel(
  value: PullRequestDisplayState,
  t: TFunction,
): string {
  const keys = {
    [PullRequestDisplayState.UNSPECIFIED]: "displayState.unknown",
    [PullRequestDisplayState.MERGED]: "displayState.merged",
    [PullRequestDisplayState.CLOSED]: "displayState.closed",
    [PullRequestDisplayState.DRAFT]: "displayState.draft",
    [PullRequestDisplayState.CONFLICT]: "displayState.conflict",
    [PullRequestDisplayState.CHANGES_REQUESTED]:
      "displayState.changesRequested",
    [PullRequestDisplayState.APPROVED]: "displayState.approved",
    [PullRequestDisplayState.REVIEW_WAITING]: "displayState.reviewWaiting",
    [PullRequestDisplayState.OPEN]: "displayState.open",
    [PullRequestDisplayState.UNKNOWN]: "displayState.unknown",
  } as const;
  return t(keys[value] ?? "displayState.unknown");
}

export function blockedReasonLabel(
  reason: BlockedReason | undefined,
  taskTitle: (id: string) => string | undefined,
  t: TFunction,
): string {
  if (!reason) return "";
  const title =
    taskTitle(reason.blockerTaskId) ?? t("blockedReason.unknownBlocker");
  switch (reason.code) {
    case BlockedReasonCode.DEPENDENCY_DATA_INCOMPLETE:
      return t("blockedReason.dependencyDataIncomplete");
    case BlockedReasonCode.BLOCKER_STALE:
      return t("blockedReason.blockerStale", { title });
    case BlockedReasonCode.WAITING_FOR_BLOCKER:
      return t("blockedReason.waitingForBlocker", { title });
    default:
      return "";
  }
}

const errorKeys = {
  [DomainErrorCode.CROSS_FEATURE_DEPENDENCY]: "error.crossFeatureDependency",
  [DomainErrorCode.DUPLICATE_DEPENDENCY]: "error.duplicateDependency",
  [DomainErrorCode.DUPLICATE_PULL_REQUEST]: "error.duplicatePullRequest",
  [DomainErrorCode.GITHUB_AUTH]: "error.githubAuth",
  [DomainErrorCode.INVALID_DATABASE]: "error.invalidDatabase",
  [DomainErrorCode.INVALID_DOCUMENT]: "error.invalidDocument",
  [DomainErrorCode.INVALID_DOCUMENT_KIND]: "error.invalidDocumentKind",
  [DomainErrorCode.INVALID_KIND]: "error.invalidKind",
  [DomainErrorCode.INVALID_PARENT]: "error.invalidParent",
  [DomainErrorCode.INVALID_PULL_REQUEST_URL]: "error.invalidPullRequestUrl",
  [DomainErrorCode.INVALID_SEED]: "error.invalidSeed",
  [DomainErrorCode.INVALID_SLUG]: "error.invalidSlug",
  [DomainErrorCode.INVALID_STATUS]: "error.invalidStatus",
  [DomainErrorCode.INVALID_TITLE]: "error.invalidTitle",
  [DomainErrorCode.NOT_FOUND]: "error.notFound",
  [DomainErrorCode.REFERENCES_EXIST]: "error.referencesExist",
} as const;

export function formatError(error: Error, t: TFunction): string {
  const connectError = ConnectError.from(error);
  const detail = connectError.findDetails(ErrorDetailSchema)[0];
  if (
    !detail ||
    detail.code === DomainErrorCode.UNSPECIFIED ||
    detail.code === DomainErrorCode.INTERNAL
  )
    return connectError.rawMessage;
  if (detail.code === DomainErrorCode.CYCLE)
    return t("error.cycle", { path: detail.path.join(" → ") });
  const key = errorKeys[detail.code as keyof typeof errorKeys];
  return key ? t(key) : connectError.rawMessage;
}
