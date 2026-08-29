import { useMemo } from "react";
import type {
  Dependency,
  Feature,
  PullRequest,
  Snapshot,
  Task,
} from "../gen/prx/v1/prx_pb";
import type { TaskNodeDocument } from "./TaskNode";

interface FeatureWorkspaceData {
  feature: Feature | undefined;
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
}

export function useFeatureWorkspaceData(
  data: Snapshot | undefined,
  featureId: string,
): FeatureWorkspaceData {
  const feature = data?.features.find((item) => item.id === featureId);
  const tasks = useMemo(
    () => data?.tasks.filter((task) => task.featureId === featureId) ?? [],
    [data, featureId],
  );
  const taskIds = useMemo(() => new Set(tasks.map((task) => task.id)), [tasks]);
  const dependencies = useMemo(
    () =>
      data?.dependencies.filter((dependency) =>
        taskIds.has(dependency.blockerTaskId),
      ) ?? [],
    [data, taskIds],
  );
  const pullRequests = useMemo(
    () => new Map(data?.pullRequests.map((pr) => [pr.taskId, pr]) ?? []),
    [data],
  );
  const documentsByTask = useMemo(() => {
    const result = new Map<string, TaskNodeDocument[]>();
    for (const document of data?.documents ?? []) {
      if (!document.taskId) continue;
      const documents = result.get(document.taskId) ?? [];
      documents.push(document);
      result.set(document.taskId, documents);
    }
    return result;
  }, [data]);
  return { feature, tasks, dependencies, pullRequests, documentsByTask };
}
