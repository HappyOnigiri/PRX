import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  configMutations,
  getConfig,
  getDebugReport,
  getSnapshot,
  getSyncStatus,
  mutations,
  readDocumentContent,
  selectLocalFile,
  syncIfDue,
} from "../src/api";
import { makeSnapshot } from "./factories";

const apiMocks = vi.hoisted(() => {
  const client = {
    getSnapshot: vi.fn(),
    getConfig: vi.fn(),
    getGitHubSyncStatus: vi.fn(),
    getDebugReport: vi.fn(),
    syncGitHubIfDue: vi.fn(),
    createFeature: vi.fn(),
    updateFeature: vi.fn(),
    deleteFeature: vi.fn(),
    createTask: vi.fn(),
    updateTask: vi.fn(),
    deleteTask: vi.fn(),
    addDependency: vi.fn(),
    removeDependency: vi.fn(),
    attachPullRequest: vi.fn(),
    detachPullRequest: vi.fn(),
    addDocument: vi.fn(),
    getDocument: vi.fn(),
    updateDocument: vi.fn(),
    deleteDocument: vi.fn(),
    sync: vi.fn(),
    readDocumentContent: vi.fn(),
    selectLocalFile: vi.fn(),
    addGitHubHost: vi.fn(),
    updateGitHubHost: vi.fn(),
    deleteGitHubHost: vi.fn(),
    addGitHubAuthMethod: vi.fn(),
    updateGitHubAuthMethod: vi.fn(),
    deleteGitHubAuthMethod: vi.fn(),
    reorderGitHubAuthMethods: vi.fn(),
    validateConfig: vi.fn(),
    updateGitHubSyncConfig: vi.fn(),
  };
  return {
    client,
    createClient: vi.fn(() => client),
    createConnectTransport: vi.fn(() => ({})),
  };
});

vi.mock("@connectrpc/connect", () => ({
  createClient: apiMocks.createClient,
}));
vi.mock("@connectrpc/connect-web", () => ({
  createConnectTransport: apiMocks.createConnectTransport,
}));

describe("RPC API wrappers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    for (const method of Object.values(apiMocks.client))
      method.mockResolvedValue({});
  });

  it("returns a snapshot and rejects an empty server response", async () => {
    apiMocks.client.getSnapshot.mockResolvedValueOnce({
      snapshot: makeSnapshot(),
    });
    await expect(getSnapshot()).resolves.toEqual(makeSnapshot());

    apiMocks.client.getSnapshot.mockResolvedValueOnce({ snapshot: undefined });
    await expect(getSnapshot()).rejects.toThrow(
      "The server returned an empty snapshot.",
    );
  });

  it("wraps configuration reads and mutations", async () => {
    const config = { version: 1 };
    apiMocks.client.getConfig.mockResolvedValueOnce({ config });
    await expect(getConfig()).resolves.toBe(config);
    apiMocks.client.getConfig.mockResolvedValueOnce({ config: undefined });
    await expect(getConfig()).rejects.toThrow(
      "The server returned an empty GitHub configuration.",
    );

    await configMutations.addHost({ host: "ghe.example.com" });
    await configMutations.updateHost({
      host: "ghe.example.com",
      newHost: "ghe.internal",
    });
    await configMutations.deleteHost("ghe.internal");
    await configMutations.addAuth({
      id: "token",
      host: "github.com",
      type: 3,
      token: "secret",
    });
    await configMutations.updateAuth({ id: "token", user: "Mika" });
    await configMutations.deleteAuth("token");
    await configMutations.reorderAuth(["token"]);
    await configMutations.validate();
    await configMutations.updateSync(600n);
    expect(apiMocks.client.addGitHubHost).toHaveBeenCalled();
    expect(apiMocks.client.updateGitHubHost).toHaveBeenCalled();
    expect(apiMocks.client.deleteGitHubHost).toHaveBeenCalled();
    expect(apiMocks.client.addGitHubAuthMethod).toHaveBeenCalled();
    expect(apiMocks.client.updateGitHubAuthMethod).toHaveBeenCalled();
    expect(apiMocks.client.deleteGitHubAuthMethod).toHaveBeenCalled();
    expect(apiMocks.client.reorderGitHubAuthMethods).toHaveBeenCalled();
    expect(apiMocks.client.validateConfig).toHaveBeenCalled();
    expect(apiMocks.client.updateGitHubSyncConfig).toHaveBeenCalledWith(
      expect.objectContaining({ intervalSeconds: 600n }),
    );
  });

  it("returns the debug report with its rendered text", async () => {
    const report = { problems: [] };
    apiMocks.client.getDebugReport.mockResolvedValueOnce({
      report,
      text: "PRX diagnostic report\n",
    });
    await expect(getDebugReport()).resolves.toEqual({
      report,
      text: "PRX diagnostic report\n",
    });

    apiMocks.client.getDebugReport.mockResolvedValueOnce({ report: undefined });
    await expect(getDebugReport()).rejects.toThrow(
      "The server returned an empty debug report.",
    );
  });

  it("reads synchronization status and requests a due refresh", async () => {
    const status = { intervalSeconds: 3600n, succeeded: 2, failed: 0 };
    apiMocks.client.getGitHubSyncStatus.mockResolvedValueOnce({ status });
    await expect(getSyncStatus()).resolves.toBe(status);
    apiMocks.client.getGitHubSyncStatus.mockResolvedValueOnce({});
    await expect(getSyncStatus()).rejects.toThrow("empty sync status");
    apiMocks.client.syncGitHubIfDue.mockResolvedValueOnce({
      ran: false,
      status,
    });
    await expect(syncIfDue()).resolves.toEqual({ ran: false, status });
  });

  it("requests a native local file selection", async () => {
    apiMocks.client.selectLocalFile.mockResolvedValueOnce({
      path: "/tmp/plan.md",
      canceled: false,
    });
    await expect(selectLocalFile()).resolves.toEqual({
      path: "/tmp/plan.md",
      canceled: false,
    });
    expect(apiMocks.client.selectLocalFile).toHaveBeenCalledOnce();
  });

  it("serializes every mutation request and propagates markdown content", async () => {
    await mutations.createFeature({
      slug: "release",
      title: "Release",
      description: "Ship it",
    });
    expect(apiMocks.client.createFeature).toHaveBeenCalledWith(
      expect.objectContaining({
        slug: "release",
        title: "Release",
        description: "Ship it",
      }),
    );

    await mutations.updateFeature({
      id: "feature-1",
      title: "Updated",
      status: 2,
      archived: true,
    });
    expect(apiMocks.client.updateFeature).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "feature-1",
        title: "Updated",
        status: 2,
        archived: true,
      }),
    );

    await mutations.deleteFeature("feature-1");
    expect(apiMocks.client.deleteFeature).toHaveBeenCalledWith(
      expect.objectContaining({ id: "feature-1", cascade: true }),
    );

    await mutations.createTask({
      featureId: "feature-1",
      title: "Implement",
      scope: "WebUI",
      kind: 1,
      assignee: "Mika",
    });
    expect(apiMocks.client.createTask).toHaveBeenCalledWith(
      expect.objectContaining({
        featureId: "feature-1",
        title: "Implement",
        scope: "WebUI",
        kind: 1,
        assignee: "Mika",
      }),
    );

    await mutations.updateTask({
      id: "task-1",
      scope: "API",
      status: 2,
      assignee: "Ren",
    });
    expect(apiMocks.client.updateTask).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "task-1",
        scope: "API",
        status: 2,
        assignee: "Ren",
      }),
    );

    await mutations.deleteTask("task-1");
    expect(apiMocks.client.deleteTask).toHaveBeenCalledWith(
      expect.objectContaining({ id: "task-1", cascade: true }),
    );

    await mutations.addDependency("task-1", "task-2");
    expect(apiMocks.client.addDependency).toHaveBeenCalledWith(
      expect.objectContaining({
        blockerTaskId: "task-1",
        blockedTaskId: "task-2",
      }),
    );
    await mutations.removeDependency("task-1", "task-2");
    expect(apiMocks.client.removeDependency).toHaveBeenCalledWith(
      expect.objectContaining({
        blockerTaskId: "task-1",
        blockedTaskId: "task-2",
      }),
    );

    await mutations.attachPR("task-1", "https://github.com/acme/prx/pull/42");
    expect(apiMocks.client.attachPullRequest).toHaveBeenCalledWith(
      expect.objectContaining({
        taskId: "task-1",
        url: "https://github.com/acme/prx/pull/42",
      }),
    );
    await mutations.detachPR("task-1");
    expect(apiMocks.client.detachPullRequest).toHaveBeenCalledWith(
      expect.objectContaining({ taskId: "task-1" }),
    );

    await mutations.addDocument({
      taskId: "task-1",
      kind: 1,
      title: "Runbook",
      value: "https://example.com/runbook",
    });
    expect(apiMocks.client.addDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        taskId: "task-1",
        title: "Runbook",
        source: {
          case: "url",
          value: "https://example.com/runbook",
        },
      }),
    );
    await mutations.getDocument("document-1");
    expect(apiMocks.client.getDocument).toHaveBeenCalledWith(
      expect.objectContaining({ id: "document-1" }),
    );
    await mutations.updateDocument({
      id: "document-1",
      source: { case: "markdown", value: "# Plan" },
      isImplementationPlan: true,
    });
    expect(apiMocks.client.updateDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "document-1",
        source: { case: "markdown", value: "# Plan" },
        isImplementationPlan: true,
      }),
    );
    await mutations.deleteDocument("document-1");
    expect(apiMocks.client.deleteDocument).toHaveBeenCalledWith(
      expect.objectContaining({ id: "document-1" }),
    );

    await mutations.sync();
    await mutations.sync("feature-1", "task-1");
    expect(apiMocks.client.sync).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({}),
    );
    expect(apiMocks.client.sync).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ featureId: "feature-1", taskId: "task-1" }),
    );

    apiMocks.client.readDocumentContent.mockResolvedValueOnce({
      content: "# Notes",
    });
    await expect(readDocumentContent("document-1")).resolves.toBe("# Notes");
    expect(apiMocks.client.readDocumentContent).toHaveBeenCalledWith(
      expect.objectContaining({ id: "document-1" }),
    );
  });
});
