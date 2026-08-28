import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  AddDependencyRequestSchema,
  AddDocumentRequestSchema,
  AttachPullRequestRequestSchema,
  CreateFeatureRequestSchema,
  CreateTaskRequestSchema,
  DeleteFeatureRequestSchema,
  DeleteDocumentRequestSchema,
  DeleteTaskRequestSchema,
  DetachPullRequestRequestSchema,
  GetSnapshotRequestSchema,
  PRXService,
  RemoveDependencyRequestSchema,
  SyncRequestSchema,
  UpdateFeatureRequestSchema,
  UpdateTaskRequestSchema,
  type Snapshot,
  DocumentKind,
  FeatureStatus,
  TaskKind,
  TaskStatus,
} from "./gen/prx/v1/prx_pb";

const transport = createConnectTransport({ baseUrl: window.location.origin });
export const client = createClient(PRXService, transport);

export async function getSnapshot(): Promise<Snapshot> {
  const response = await client.getSnapshot(create(GetSnapshotRequestSchema));
  if (!response.snapshot)
    throw new Error("The server returned an empty snapshot.");
  return response.snapshot;
}

export const mutations = {
  createFeature: (input: {
    slug: string;
    title: string;
    description: string;
  }) => client.createFeature(create(CreateFeatureRequestSchema, input)),
  updateFeature: (input: {
    id: string;
    slug?: string;
    title?: string;
    description?: string;
    status?: FeatureStatus;
    archived?: boolean;
  }) => client.updateFeature(create(UpdateFeatureRequestSchema, input)),
  deleteFeature: (id: string) =>
    client.deleteFeature(
      create(DeleteFeatureRequestSchema, { id, cascade: true }),
    ),
  createTask: (input: {
    featureId: string;
    title: string;
    scope: string;
    kind: TaskKind;
    assignee: string;
  }) => client.createTask(create(CreateTaskRequestSchema, input)),
  updateTask: (input: {
    id: string;
    title?: string;
    scope?: string;
    status?: TaskStatus;
    assignee?: string;
  }) => client.updateTask(create(UpdateTaskRequestSchema, input)),
  deleteTask: (id: string) =>
    client.deleteTask(create(DeleteTaskRequestSchema, { id, cascade: true })),
  addDependency: (blockerTaskId: string, blockedTaskId: string) =>
    client.addDependency(
      create(AddDependencyRequestSchema, { blockerTaskId, blockedTaskId }),
    ),
  removeDependency: (blockerTaskId: string, blockedTaskId: string) =>
    client.removeDependency(
      create(RemoveDependencyRequestSchema, { blockerTaskId, blockedTaskId }),
    ),
  attachPR: (taskId: string, url: string) =>
    client.attachPullRequest(
      create(AttachPullRequestRequestSchema, { taskId, url }),
    ),
  detachPR: (taskId: string) =>
    client.detachPullRequest(
      create(DetachPullRequestRequestSchema, { taskId }),
    ),
  addDocument: (input: {
    featureId?: string;
    taskId?: string;
    kind: DocumentKind;
    title: string;
    value: string;
  }) => client.addDocument(create(AddDocumentRequestSchema, input)),
  deleteDocument: (id: string) =>
    client.deleteDocument(create(DeleteDocumentRequestSchema, { id })),
  sync: (featureId?: string, taskId?: string) =>
    client.sync(create(SyncRequestSchema, { featureId, taskId })),
};
