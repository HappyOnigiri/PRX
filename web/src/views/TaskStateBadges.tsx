import { useTranslation } from "react-i18next";
import { TaskBlockLabel, type TaskDisplayState } from "../gen/prx/v1/prx_pb";
import {
  taskBlockLabelLabel,
  taskBlockLabelToken,
  taskDisplayStateLabel,
  taskDisplayStateToken,
} from "../i18n/domain";
import { StatusBadge } from "./StatusBadge";

// タスクの状態は 2 系統ある。ステータスは 1 つで進み具合を、ブロックラベルは
// 0〜3 個で進行を妨げている事情を示す。docs/design/webui.md を参照。
export function TaskStatusBadge({ state }: { state: TaskDisplayState }) {
  const { t } = useTranslation();
  return (
    <StatusBadge
      className={`state-${taskDisplayStateToken(state)}`}
      label={taskDisplayStateLabel(state, t)}
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
