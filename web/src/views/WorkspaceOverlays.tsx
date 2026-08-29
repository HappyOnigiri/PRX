import type { Feature } from "../gen/prx/v1/prx_pb";
import { CreateTaskDialog } from "./CreateTaskDialog";
import { EditFeatureDialog } from "./EditFeatureDialog";
import { MarkdownPreview } from "./MarkdownPreview";
import type { TaskNodeDocument } from "./TaskNode";

interface WorkspaceOverlaysProps {
  feature: Feature;
  featureId: string;
  previewDocument: TaskNodeDocument | undefined;
  showTask: boolean;
  showFeatureEdit: boolean;
  onClosePreview: () => void;
  onCloseTask: () => void;
  onCloseFeatureEdit: () => void;
}

export function WorkspaceOverlays({
  feature,
  featureId,
  previewDocument,
  showTask,
  showFeatureEdit,
  onClosePreview,
  onCloseTask,
  onCloseFeatureEdit,
}: WorkspaceOverlaysProps) {
  return (
    <>
      {previewDocument && (
        <MarkdownPreview
          key={previewDocument.id}
          document={previewDocument}
          onClose={onClosePreview}
        />
      )}
      {showTask && (
        <CreateTaskDialog featureId={featureId} onClose={onCloseTask} />
      )}
      {showFeatureEdit && (
        <EditFeatureDialog feature={feature} onClose={onCloseFeatureEdit} />
      )}
    </>
  );
}
