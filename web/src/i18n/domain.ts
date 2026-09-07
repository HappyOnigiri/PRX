import { ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";
import {
  BlockedReasonCode,
  CheckState,
  DebugProblemCode,
  DocumentKind,
  DomainErrorCode,
  ErrorDetailSchema,
  FeatureStatus,
  PullRequestDisplayState,
  TaskBlockLabel,
  TaskDisplayState,
  TaskStatus,
  type BlockedReason,
} from "../gen/prx/v1/prx_pb";

export function featureStatusLabel(value: FeatureStatus, t: TFunction): string {
  return t(featureStatusKeys[value]);
}

export function taskStatusLabel(value: TaskStatus, t: TFunction): string {
  return t(taskStatusKeys[value]);
}

// token は CSS クラス名になるので、呼び出し側ごとに enum 名の綴りを導出させず、
// taskDisplayStateToken と同じ形に揃える。
export function featureStatusToken(value: FeatureStatus): string {
  return FeatureStatus[value].toLowerCase().replaceAll("_", "-");
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
  [TaskStatus.NOT_STARTED]: "taskStatus.notStarted",
  [TaskStatus.DESIGNING]: "taskStatus.designing",
  [TaskStatus.IN_PROGRESS]: "taskStatus.inProgress",
  [TaskStatus.COMPLETED]: "taskStatus.completed",
  [TaskStatus.CLOSED]: "taskStatus.closed",
  [TaskStatus.UNSPECIFIED]: "taskStatus.unknown",
} as const satisfies Record<TaskStatus, string>;

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
  [TaskDisplayState.DESIGNING]: "displayState.designing",
  [TaskDisplayState.DESIGNED]: "displayState.designed",
  [TaskDisplayState.IN_PROGRESS]: "displayState.inProgress",
  [TaskDisplayState.IMPLEMENTED]: "displayState.implemented",
  [TaskDisplayState.IN_REVIEW]: "displayState.inReview",
  [TaskDisplayState.APPROVED]: "displayState.approved",
  [TaskDisplayState.MERGED]: "displayState.merged",
  [TaskDisplayState.COMPLETED]: "displayState.completed",
  [TaskDisplayState.CLOSED]: "displayState.closed",
  [TaskDisplayState.UNKNOWN]: "displayState.unknown",
} as const satisfies Record<TaskDisplayState, string>;

// ブロックラベルはステータスとは別の軸なので、翻訳キーも別に持つ。
export const blockLabelKeys = {
  [TaskBlockLabel.UNSPECIFIED]: "blockLabel.unknown",
  [TaskBlockLabel.DEPENDENCY_UNRESOLVED]: "blockLabel.dependencyUnresolved",
  [TaskBlockLabel.CONFLICT]: "blockLabel.conflict",
  [TaskBlockLabel.CHANGES_REQUESTED]: "blockLabel.changesRequested",
  [TaskBlockLabel.CI_FAILED]: "blockLabel.ciFailed",
} as const satisfies Record<TaskBlockLabel, string>;

export function taskBlockLabelLabel(
  value: TaskBlockLabel,
  t: TFunction,
): string {
  return t(blockLabelKeys[value]);
}

export function taskBlockLabelToken(value: TaskBlockLabel): string {
  return TaskBlockLabel[value].toLowerCase().replaceAll("_", "-");
}

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

export const checkStateKeys = {
  [CheckState.UNSPECIFIED]: "checkState.unknown",
  [CheckState.UNKNOWN]: "checkState.unknown",
  [CheckState.NONE]: "checkState.none",
  [CheckState.PENDING]: "checkState.pending",
  [CheckState.SUCCESS]: "checkState.success",
  [CheckState.FAILURE]: "checkState.failure",
} as const satisfies Record<CheckState, string>;

export function checkStateLabel(value: CheckState, t: TFunction): string {
  return t(checkStateKeys[value]);
}

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

// 新しいサーバーのレポートにはこの bundle が知らない code が含まれうるので、
// 未対応の値は存在しない翻訳キーを表示せず汎用ラベルにフォールバックする。
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
  [DomainErrorCode.ARCHIVED_READ_ONLY]: "error.archivedReadOnly",
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
  [DomainErrorCode.INVALID_PARENT]: "error.invalidParent",
  [DomainErrorCode.INVALID_PULL_REQUEST_URL]: "error.invalidPullRequestUrl",
  [DomainErrorCode.INVALID_STATUS]: "error.invalidStatus",
  [DomainErrorCode.INVALID_TITLE]: "error.invalidTitle",
  [DomainErrorCode.NOT_FOUND]: "error.notFound",
  [DomainErrorCode.REFERENCES_EXIST]: "error.referencesExist",
} as const satisfies Record<DomainErrorCode, string>;

// 循環経路は task ID で届くので、画面上の task を知る呼び出し側は resolver を
// 渡して ID ではなくタイトルを表示する。
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
