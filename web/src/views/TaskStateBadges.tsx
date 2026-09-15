import {
  CircleCheck,
  CircleDashed,
  CircleHelp,
  CircleX,
  Code2,
  MessagesSquare,
  PencilRuler,
  type LucideIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  TaskBlockLabel,
  TaskDisplayState,
  type TaskLabelAppearance,
  type TaskLabelAppearances,
} from "../gen/prx/v1/prx_pb";
import type { taskBadgeStageKeys, taskBadgeStateKeys } from "../i18n/domain";
import {
  taskBadgeStageLabel,
  taskBadgeStateLabel,
  taskBlockLabelLabel,
  taskBlockLabelToken,
  taskDisplayStateLabel,
  taskDisplayStateToken,
} from "../i18n/domain";
import { StatusBadge } from "./StatusBadge";

type TaskBadgeStage = keyof typeof taskBadgeStageKeys;
type TaskBadgeState = keyof typeof taskBadgeStateKeys;

interface TaskStatusPresentation {
  icon: LucideIcon;
  stage?: TaskBadgeStage;
  state?: TaskBadgeState;
}

const taskStatusPresentations = {
  [TaskDisplayState.UNSPECIFIED]: { icon: CircleHelp },
  [TaskDisplayState.NOT_STARTED]: { icon: CircleDashed },
  [TaskDisplayState.DESIGNING]: {
    icon: PencilRuler,
    stage: "design",
    state: "working",
  },
  [TaskDisplayState.DESIGNED]: {
    icon: PencilRuler,
    stage: "design",
    state: "done",
  },
  [TaskDisplayState.IN_PROGRESS]: {
    icon: Code2,
    stage: "implementation",
    state: "working",
  },
  [TaskDisplayState.IMPLEMENTED]: {
    icon: Code2,
    stage: "implementation",
    state: "done",
  },
  [TaskDisplayState.IN_REVIEW]: {
    icon: MessagesSquare,
    stage: "review",
    state: "waiting",
  },
  [TaskDisplayState.APPROVED]: {
    icon: MessagesSquare,
    stage: "review",
    state: "approved",
  },
  [TaskDisplayState.COMPLETED]: { icon: CircleCheck },
  [TaskDisplayState.MERGED]: { icon: CircleCheck },
  [TaskDisplayState.CLOSED]: { icon: CircleX },
  [TaskDisplayState.UNKNOWN]: { icon: CircleHelp },
} as const satisfies Record<TaskDisplayState, TaskStatusPresentation>;

// タスクの状態は 2 系統ある。ステータスは 1 つで進み具合を、ブロックラベルは
// 0〜4 個で進行を妨げている事情を示す。docs/design/webui.md を参照。
export function TaskStatusBadge({
  state,
  appearances,
}: {
  state: TaskDisplayState;
  appearances?: TaskLabelAppearances | undefined;
}) {
  const { t } = useTranslation();
  const presentation: TaskStatusPresentation = taskStatusPresentations[state];
  const appearance = findAppearance(
    appearances,
    taskDisplayStateLabelKey(state),
  );
  const customText = appearance?.textOverridden ? appearance.text : "";
  const label = customText || taskDisplayStateLabel(state, t);
  return (
    <StatusBadge
      className={`state-${taskDisplayStateToken(state)}`}
      color={appearance?.colorOverridden ? appearance.color : undefined}
      icon={presentation.icon}
      label={label}
      accessibleLabel={label}
      {...(customText
        ? {}
        : presentation.stage
          ? { mainLabel: taskBadgeStageLabel(presentation.stage, t) }
          : {})}
      {...(customText
        ? {}
        : presentation.state
          ? { secondaryLabel: taskBadgeStateLabel(presentation.state, t) }
          : {})}
    />
  );
}

// ラベルの並びはサーバーが決めた順のまま出す。並べ替えると同じタスクが場所に
// よって違う順で見えてしまう。
export function TaskBlockLabels({
  labels,
  dependencyDetail,
  appearances,
}: {
  labels: TaskBlockLabel[];
  // dependencyDetail は依存未解決ラベルの補足で、どの blocker を待つかを示す。
  dependencyDetail?: string | undefined;
  appearances?: TaskLabelAppearances | undefined;
}) {
  const { t } = useTranslation();
  return (
    <>
      {labels.map((label) => {
        const detail =
          label === TaskBlockLabel.DEPENDENCY_UNRESOLVED
            ? dependencyDetail
            : "";
        const appearance = findAppearance(
          appearances,
          taskBlockLabelKey(label),
        );
        const customText = appearance?.textOverridden ? appearance.text : "";
        return (
          <StatusBadge
            key={label}
            className={`block-${taskBlockLabelToken(label)}`}
            color={appearance?.colorOverridden ? appearance.color : undefined}
            label={customText || taskBlockLabelLabel(label, t)}
            {...(customText ? { accessibleLabel: customText } : {})}
            {...(detail ? { title: detail } : {})}
          />
        );
      })}
    </>
  );
}

function findAppearance(
  appearances: TaskLabelAppearances | undefined,
  key: string,
): TaskLabelAppearance | undefined {
  return appearances?.values.find((appearance) => appearance.key === key);
}

function taskDisplayStateLabelKey(state: TaskDisplayState): string {
  return `status.${taskDisplayStateKeyToken(state)}`;
}

function taskDisplayStateKeyToken(state: TaskDisplayState): string {
  switch (state) {
    case TaskDisplayState.UNSPECIFIED:
      return "unknown";
    case TaskDisplayState.NOT_STARTED:
      return "not_started";
    case TaskDisplayState.DESIGNING:
      return "designing";
    case TaskDisplayState.DESIGNED:
      return "designed";
    case TaskDisplayState.IN_PROGRESS:
      return "in_progress";
    case TaskDisplayState.IMPLEMENTED:
      return "implemented";
    case TaskDisplayState.IN_REVIEW:
      return "in_review";
    case TaskDisplayState.APPROVED:
      return "approved";
    case TaskDisplayState.COMPLETED:
      return "completed";
    case TaskDisplayState.MERGED:
      return "merged";
    case TaskDisplayState.CLOSED:
      return "closed";
    case TaskDisplayState.UNKNOWN:
      return "unknown";
  }
}

function taskBlockLabelKey(label: TaskBlockLabel): string {
  switch (label) {
    case TaskBlockLabel.UNSPECIFIED:
      return "block.unknown";
    case TaskBlockLabel.DEPENDENCY_UNRESOLVED:
      return "block.dependency_unresolved";
    case TaskBlockLabel.CONFLICT:
      return "block.conflict";
    case TaskBlockLabel.CHANGES_REQUESTED:
      return "block.changes_requested";
    case TaskBlockLabel.CI_FAILED:
      return "block.ci_failed";
  }
}
