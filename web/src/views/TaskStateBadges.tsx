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
import { TaskBlockLabel, TaskDisplayState } from "../gen/prx/v1/prx_pb";
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
export function TaskStatusBadge({ state }: { state: TaskDisplayState }) {
  const { t } = useTranslation();
  const presentation: TaskStatusPresentation = taskStatusPresentations[state];
  const label = taskDisplayStateLabel(state, t);
  return (
    <StatusBadge
      className={`state-${taskDisplayStateToken(state)}`}
      icon={presentation.icon}
      label={label}
      accessibleLabel={label}
      {...(presentation.stage
        ? { mainLabel: taskBadgeStageLabel(presentation.stage, t) }
        : {})}
      {...(presentation.state
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
}: {
  labels: TaskBlockLabel[];
  // dependencyDetail は依存未解決ラベルの補足で、どの blocker を待つかを示す。
  dependencyDetail?: string;
}) {
  const { t } = useTranslation();
  return (
    <>
      {labels.map((label) => {
        const detail =
          label === TaskBlockLabel.DEPENDENCY_UNRESOLVED
            ? dependencyDetail
            : "";
        return (
          <StatusBadge
            key={label}
            className={`block-${taskBlockLabelToken(label)}`}
            label={taskBlockLabelLabel(label, t)}
            {...(detail ? { title: detail } : {})}
          />
        );
      })}
    </>
  );
}
