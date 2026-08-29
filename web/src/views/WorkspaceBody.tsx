import type { Dependency, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { FeatureGraph } from "./FeatureGraph";
import { TaskInspector } from "./TaskInspector";
import type { TaskNodeDocument } from "./TaskNode";

interface WorkspaceBodyProps {
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  selectedTask: Task | undefined;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onCreateTask: () => void;
  onCloseInspector: () => void;
}

export function WorkspaceBody({
  tasks,
  dependencies,
  pullRequests,
  documentsByTask,
  selectedTask,
  onEditTask,
  onPreviewDocument,
  onCreateTask,
  onCloseInspector,
}: WorkspaceBodyProps) {
  return (
    <div className="workspace-body">
      <FeatureGraph
        tasks={tasks}
        dependencies={dependencies}
        pullRequests={pullRequests}
        documentsByTask={documentsByTask}
        onEditTask={onEditTask}
        onPreviewDocument={onPreviewDocument}
        onCreateTask={onCreateTask}
      />
      {selectedTask && (
        <TaskInspector
          task={selectedTask}
          tasks={tasks}
          dependencies={dependencies}
          pullRequest={pullRequests.get(selectedTask.id)}
          documents={documentsByTask.get(selectedTask.id) ?? []}
          onPreview={onPreviewDocument}
          onClose={onCloseInspector}
        />
      )}
    </div>
  );
}
