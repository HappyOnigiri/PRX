import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  configMutations,
  getConfig,
  getSnapshot,
  mutations,
  readMarkdownDocument,
} from "../src/api";
import { makeSnapshot } from "./factories";

const apiMocks = vi.hoisted(() => {
  const client = {
    getSnapshot: vi.fn(),
    getConfig: vi.fn(),
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
    deleteDocument: vi.fn(),
    sync: vi.fn(),
    readMarkdownDocument: vi.fn(),
    addGitHubHost: vi.fn(),
    updateGitHubHost: vi.fn(),
    deleteGitHubHost: vi.fn(),
    addGitHubAuthMethod: vi.fn(),
    updateGitHubAuthMethod: vi.fn(),
    deleteGitHubAuthMethod: vi.fn(),
    reorderGitHubAuthMethods: vi.fn(),
    validateConfig: vi.fn(),
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
    expect(apiMocks.client.addGitHubHost).toHaveBeenCalled();
    expect(apiMocks.client.updateGitHubHost).toHaveBeenCalled();
    expect(apiMocks.client.deleteGitHubHost).toHaveBeenCalled();
    expect(apiMocks.client.addGitHubAuthMethod).toHaveBeenCalled();
    expect(apiMocks.client.updateGitHubAuthMethod).toHaveBeenCalled();
    expect(apiMocks.client.deleteGitHubAuthMethod).toHaveBeenCalled();
    expect(apiMocks.client.reorderGitHubAuthMethods).toHaveBeenCalled();
    expect(apiMocks.client.validateConfig).toHaveBeenCalled();
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
        kind: 1,
        title: "Runbook",
        value: "https://example.com/runbook",
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

    apiMocks.client.readMarkdownDocument.mockResolvedValueOnce({
      content: "# Notes",
    });
    await expect(readMarkdownDocument("document-1")).resolves.toBe("# Notes");
    expect(apiMocks.client.readMarkdownDocument).toHaveBeenCalledWith(
      expect.objectContaining({ id: "document-1" }),
    );
  });
});
