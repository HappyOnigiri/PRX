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
  CreateProjectRequestSchema,
  CreateTaskRequestSchema,
  DeleteDocumentRequestSchema,
  DeleteFeatureRequestSchema,
  DeleteGitHubAuthMethodRequestSchema,
  DeleteGitHubHostRequestSchema,
  DeleteProjectRequestSchema,
  DeleteTaskRequestSchema,
  DetachPullRequestRequestSchema,
  DocumentKind,
  GetBatchPromptRequestSchema,
  GetConfigRequestSchema,
  GetDebugReportRequestSchema,
  GetDocumentRequestSchema,
  GetGitHubSyncStatusRequestSchema,
  GetPromptTemplatesRequestSchema,
  GetSnapshotRequestSchema,
  GetTaskPromptRequestSchema,
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
  UpdateProjectRequestSchema,
  UpdatePromptTemplatesRequestSchema,
  UpdateTaskRequestSchema,
  ValidateConfigRequestSchema,
  type DebugReport,
  type FeatureStatus,
  type GetBatchPromptResponse,
  type GetTaskPromptResponse,
  type GithubAuthMethodType,
  type GitHubConfig,
  type GitHubSyncStatus,
  type PromptTemplates,
  type Snapshot,
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

// レポートと整形済みテキストを一緒に受け取ることで、WebUI はセクションを表示
// しつつ `prx debug` が出力するテキストをそのままコピーできる。
export async function getDebugReport(): Promise<{
  report: DebugReport;
  text: string;
}> {
  const response = await client.getDebugReport(
    create(GetDebugReportRequestSchema),
  );
  if (!response.report)
    throw new Error("The server returned an empty debug report.");
  return { report: response.report, text: response.text };
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
  createProject: (input: { title: string; description: string }) =>
    client.createProject(create(CreateProjectRequestSchema, input)),
  updateProject: (input: {
    id: string;
    title?: string;
    description?: string;
    archived?: boolean;
  }) => client.updateProject(create(UpdateProjectRequestSchema, input)),
  // cascade は project の feature を削除せず切り離すので、WebUI に必要な形は
  // これだけ。
  deleteProject: (id: string) =>
    client.deleteProject(
      create(DeleteProjectRequestSchema, { id, cascade: true }),
    ),
  createFeature: (input: {
    title: string;
    description: string;
    projectId?: string;
  }) => client.createFeature(create(CreateFeatureRequestSchema, input)),
  updateFeature: (input: {
    id: string;
    title?: string;
    description?: string;
    status?: FeatureStatus;
    archived?: boolean;
    projectId?: string;
  }) => client.updateFeature(create(UpdateFeatureRequestSchema, input)),
  deleteFeature: (id: string) =>
    client.deleteFeature(
      create(DeleteFeatureRequestSchema, { id, cascade: true }),
    ),
  createTask: (input: {
    featureId: string;
    title: string;
    scope: string;
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
    projectId?: string;
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
      projectId?: string;
      featureId?: string;
      taskId?: string;
      title: string;
      source: typeof source;
      isImplementationPlan?: boolean;
    } = { title: input.title, source };
    if (input.projectId !== undefined) request.projectId = input.projectId;
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

// PromptTemplateSettings は保存済みテンプレートに加え、サーバーが受け付ける語彙
// と同梱テンプレートを持つ。エディタが自前の複製を使って検証・復元しないため。
export interface PromptTemplateSettings extends PromptTemplates {
  supportedPlaceholders: string[];
  requiredPlaceholder: string;
  // batch テンプレートは複数の task を扱うため語彙が別。単一 task の
  // プレースホルダは展開できない。
  batchSupportedPlaceholders: string[];
  batchRequiredPlaceholder: string;
  builtIn: PromptTemplates;
}

export async function getPromptTemplates(): Promise<PromptTemplateSettings> {
  const response = await client.getPromptTemplates(
    create(GetPromptTemplatesRequestSchema),
  );
  if (!response.templates || !response.builtIn)
    throw new Error("The server returned empty prompt templates.");
  return {
    ...response.templates,
    supportedPlaceholders: response.supportedPlaceholders,
    requiredPlaceholder: response.requiredPlaceholder,
    batchSupportedPlaceholders: response.batchSupportedPlaceholders,
    batchRequiredPlaceholder: response.batchRequiredPlaceholder,
    builtIn: response.builtIn,
  };
}

// プロンプトは snapshot と一緒ではなく都度描画する。サーバーが選ぶテンプレート
// は task に実装計画が今あるかで変わり、キャッシュ済みの snapshot はすでに実態
// とずれている可能性があるため。
export async function getTaskPrompt(
  taskId: string,
): Promise<GetTaskPromptResponse> {
  return client.getTaskPrompt(create(GetTaskPromptRequestSchema, { taskId }));
}

// batch プロンプトも単一 task と同じ理由で、選択された task から都度描画する。
// テンプレートも task もサーバー側にあり、ブラウザが保持する snapshot はどちら
// ともずれている可能性があるため。
export async function getBatchPrompt(
  featureId: string,
  taskIds: string[],
): Promise<GetBatchPromptResponse> {
  return client.getBatchPrompt(
    create(GetBatchPromptRequestSchema, { featureId, taskIds }),
  );
}

export const promptMutations = {
  updateTemplates: (input: {
    design: string;
    implementation: string;
    batch: string;
  }) =>
    client.updatePromptTemplates(
      create(UpdatePromptTemplatesRequestSchema, input),
    ),
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
