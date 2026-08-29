import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  AddDependencyRequestSchema,
  AddDocumentRequestSchema,
  AttachPullRequestRequestSchema,
  CreateFeatureRequestSchema,
  CreateTaskRequestSchema,
  DeleteDocumentRequestSchema,
  DeleteFeatureRequestSchema,
  DeleteImplementationPlanRequestSchema,
  DeleteTaskRequestSchema,
  DetachPullRequestRequestSchema,
  GetImplementationPlanRequestSchema,
  GetSnapshotRequestSchema,
  PRXService,
  ReadMarkdownDocumentRequestSchema,
  RemoveDependencyRequestSchema,
  SyncRequestSchema,
  UpdateFeatureRequestSchema,
  UpdateTaskRequestSchema,
  UpsertImplementationPlanRequestSchema,
  type DocumentKind,
  type FeatureStatus,
  type Snapshot,
  type TaskKind,
  type TaskStatus,
} from "./gen/prx/v1/prx_pb";

const transport = createConnectTransport({ baseUrl: window.location.origin });
const client = createClient(PRXService, transport);

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
  getImplementationPlan: (taskId: string) =>
    client.getImplementationPlan(
      create(GetImplementationPlanRequestSchema, { taskId }),
    ),
  upsertImplementationPlan: (input: { taskId: string; content: string }) =>
    client.upsertImplementationPlan(
      create(UpsertImplementationPlanRequestSchema, input),
    ),
  deleteImplementationPlan: (taskId: string) =>
    client.deleteImplementationPlan(
      create(DeleteImplementationPlanRequestSchema, { taskId }),
    ),
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
  sync: (featureId?: string, taskId?: string) => {
    const input: { featureId?: string; taskId?: string } = {};
    if (featureId !== undefined) input.featureId = featureId;
    if (taskId !== undefined) input.taskId = taskId;
    return client.sync(create(SyncRequestSchema, input));
  },
};

export async function readMarkdownDocument(id: string): Promise<string> {
  const response = await client.readMarkdownDocument(
    create(ReadMarkdownDocumentRequestSchema, { id }),
  );
  return response.content;
}
