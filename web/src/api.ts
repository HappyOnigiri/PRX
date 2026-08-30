import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  AddDependencyRequestSchema,
  AddDocumentRequestSchema,
  AddGitHubAuthMethodRequestSchema,
  AddGitHubHostRequestSchema,
  AttachPullRequestRequestSchema,
  CreateFeatureRequestSchema,
  CreateTaskRequestSchema,
  DeleteDocumentRequestSchema,
  DeleteFeatureRequestSchema,
  DeleteGitHubAuthMethodRequestSchema,
  DeleteGitHubHostRequestSchema,
  DeleteTaskRequestSchema,
  DetachPullRequestRequestSchema,
  DocumentKind,
  GetConfigRequestSchema,
  GetDocumentRequestSchema,
  GetGitHubSyncStatusRequestSchema,
  GetSnapshotRequestSchema,
  PRXService,
  ReadDocumentContentRequestSchema,
  RemoveDependencyRequestSchema,
  ReorderGitHubAuthMethodsRequestSchema,
  SelectLocalFileRequestSchema,
  SyncGitHubIfDueRequestSchema,
  SyncRequestSchema,
  UpdateDocumentRequestSchema,
  UpdateFeatureRequestSchema,
  UpdateGitHubAuthMethodRequestSchema,
  UpdateGitHubHostRequestSchema,
  UpdateGitHubSyncConfigRequestSchema,
  UpdateTaskRequestSchema,
  ValidateConfigRequestSchema,
  type FeatureStatus,
  type GithubAuthMethodType,
  type GitHubConfig,
  type GitHubSyncStatus,
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

export async function getConfig(): Promise<GitHubConfig> {
  const response = await client.getConfig(create(GetConfigRequestSchema));
  if (!response.config)
    throw new Error("The server returned an empty GitHub configuration.");
  return response.config;
}

export async function getSyncStatus(): Promise<GitHubSyncStatus> {
  const response = await client.getGitHubSyncStatus(
    create(GetGitHubSyncStatusRequestSchema),
  );
  if (!response.status)
    throw new Error("The server returned an empty sync status.");
  return response.status;
}

export async function syncIfDue() {
  return client.syncGitHubIfDue(create(SyncGitHubIfDueRequestSchema));
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
    title: string;
    kind: DocumentKind;
    value: string;
    isImplementationPlan?: boolean;
  }) => {
    const source =
      input.kind === DocumentKind.URL
        ? { case: "url" as const, value: input.value }
        : input.kind === DocumentKind.LOCAL_FILE
          ? { case: "localFile" as const, value: input.value }
          : { case: "markdown" as const, value: input.value };
    const request: {
      featureId?: string;
      taskId?: string;
      title: string;
      source: typeof source;
      isImplementationPlan?: boolean;
    } = { title: input.title, source };
    if (input.featureId !== undefined) request.featureId = input.featureId;
    if (input.taskId !== undefined) request.taskId = input.taskId;
    if (input.isImplementationPlan !== undefined)
      request.isImplementationPlan = input.isImplementationPlan;
    return client.addDocument(create(AddDocumentRequestSchema, request));
  },
  getDocument: (id: string) =>
    client.getDocument(create(GetDocumentRequestSchema, { id })),
  updateDocument: (input: {
    id: string;
    title?: string;
    source?: { case: "url" | "localFile" | "markdown"; value: string };
    isImplementationPlan?: boolean;
  }) => client.updateDocument(create(UpdateDocumentRequestSchema, input)),
  deleteDocument: (id: string) =>
    client.deleteDocument(create(DeleteDocumentRequestSchema, { id })),
  sync: (featureId?: string, taskId?: string) => {
    const input: { featureId?: string; taskId?: string } = {};
    if (featureId !== undefined) input.featureId = featureId;
    if (taskId !== undefined) input.taskId = taskId;
    return client.sync(create(SyncRequestSchema, input));
  },
};

export const configMutations = {
  addHost: (input: {
    host: string;
    webUrl?: string;
    apiUrl?: string;
    uploadUrl?: string;
    graphqlUrl?: string;
  }) => client.addGitHubHost(create(AddGitHubHostRequestSchema, input)),
  updateHost: (input: {
    host: string;
    newHost?: string;
    webUrl?: string;
    apiUrl?: string;
    uploadUrl?: string;
    graphqlUrl?: string;
  }) => client.updateGitHubHost(create(UpdateGitHubHostRequestSchema, input)),
  deleteHost: (host: string) =>
    client.deleteGitHubHost(create(DeleteGitHubHostRequestSchema, { host })),
  addAuth: (input: {
    id: string;
    host: string;
    type: GithubAuthMethodType;
    account?: string;
    service?: string;
    variable?: string;
    user?: string;
    token?: string;
  }) =>
    client.addGitHubAuthMethod(create(AddGitHubAuthMethodRequestSchema, input)),
  updateAuth: (input: {
    id: string;
    newId?: string;
    host?: string;
    type?: GithubAuthMethodType;
    account?: string;
    service?: string;
    variable?: string;
    user?: string;
    token?: string;
  }) =>
    client.updateGitHubAuthMethod(
      create(UpdateGitHubAuthMethodRequestSchema, input),
    ),
  deleteAuth: (id: string) =>
    client.deleteGitHubAuthMethod(
      create(DeleteGitHubAuthMethodRequestSchema, { id }),
    ),
  reorderAuth: (ids: string[]) =>
    client.reorderGitHubAuthMethods(
      create(ReorderGitHubAuthMethodsRequestSchema, { ids }),
    ),
  updateSync: (intervalSeconds: bigint) =>
    client.updateGitHubSyncConfig(
      create(UpdateGitHubSyncConfigRequestSchema, { intervalSeconds }),
    ),
  validate: () => client.validateConfig(create(ValidateConfigRequestSchema)),
};

export async function readDocumentContent(id: string): Promise<string> {
  const response = await client.readDocumentContent(
    create(ReadDocumentContentRequestSchema, { id }),
  );
  return response.content;
}

export async function selectLocalFile() {
  return client.selectLocalFile(create(SelectLocalFileRequestSchema));
}
