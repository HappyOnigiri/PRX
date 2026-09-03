// A document belongs to exactly one of a project, a feature, or a task. Each
// arm marks the other keys as never so a caller cannot pass two parents at
// once, which the server would reject at the far end of the request.
export type DocumentParent =
  | { projectId: string; featureId?: never; taskId?: never }
  | { projectId?: never; featureId: string; taskId?: never }
  | { projectId?: never; featureId?: never; taskId: string };
