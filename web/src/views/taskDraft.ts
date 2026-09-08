import { useState } from "react";
import { mutations } from "../api";
import type { Task, TaskStatus } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";

interface TaskDraft {
  title: string;
  scope: string;
  status: TaskStatus;
  assignee: string;
}

export interface TaskDraftController {
  draft: TaskDraft;
  dirty: boolean;
  pending: boolean;
  error: Error | null;
  setDraft: (next: TaskDraft) => void;
  save: () => void;
}

function draftOf(task: Task): TaskDraft {
  return {
    title: task.title,
    scope: task.scope,
    status: task.status,
    assignee: task.assignee,
  };
}

// 保存はインスペクタのフッタ 1 つなので、下書きはフォームではなくインスペクタが
// 持つ。別の task を選んだときの入れ替えは、インスペクタを task の id で
// key しているので React が行う。docs/design/webui.md を参照。
export function useTaskDraft(task: Task): TaskDraftController {
  const updateTask = useDomainMutation(mutations.updateTask);
  const [draft, setDraft] = useState(() => draftOf(task));
  const saved = draftOf(task);
  return {
    draft,
    dirty:
      draft.title !== saved.title ||
      draft.scope !== saved.scope ||
      draft.status !== saved.status ||
      draft.assignee !== saved.assignee,
    pending: updateTask.isPending,
    error: updateTask.error,
    setDraft,
    save: () => {
      updateTask.mutate({ id: task.id, ...draft });
    },
  };
}
