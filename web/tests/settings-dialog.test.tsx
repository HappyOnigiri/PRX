import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GithubAuthMethodType } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { SettingsDialog } from "../src/views/SettingsDialog";

const settingsMocks = vi.hoisted(() => {
  const api = {
    addHost: vi.fn(),
    updateHost: vi.fn(),
    deleteHost: vi.fn(),
    addAuth: vi.fn(),
    updateAuth: vi.fn(),
    deleteAuth: vi.fn(),
    reorderAuth: vi.fn(),
    updateSync: vi.fn(),
  };
  const mutation = () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
    error: null as Error | null,
  });
  return {
    api,
    config: {
      data: {
        autoSyncIntervalSeconds: 3600n,
        hosts: [
          {
            host: "github.com",
            webUrl: "https://github.com",
            apiUrl: "https://api.github.com/",
            uploadUrl: "https://uploads.github.com/",
            graphqlUrl: "https://api.github.com/graphql",
          },
          {
            host: "ghe.example.com",
            webUrl: "https://ghe.example.com",
            apiUrl: "https://ghe.example.com/api/v3/",
            uploadUrl: "https://ghe.example.com/api/uploads/",
            graphqlUrl: "https://ghe.example.com/api/graphql",
          },
        ],
        authMethods: [
          {
            id: "work",
            host: "github.com",
            type: 3,
            account: "",
            service: "",
            variable: "",
            user: "",
            secretHint: "gith…cret",
          },
          {
            id: "ghe",
            host: "ghe.example.com",
            type: 2,
            account: "",
            service: "",
            variable: "GH_ENTERPRISE_TOKEN",
            user: "",
          },
        ],
      },
      isPending: false,
      error: null as Error | null,
    },
    debugReport: {
      data: {
        report: {
          problems: [],
          runtime: { generatedAt: "2026-09-03T04:05:06Z" },
        },
        text: "PRX diagnostic report\n",
      },
      isError: false,
      error: null as Error | null,
      refetch: vi.fn(),
    },
    promptTemplates: {
      data: {
        design: "Design {{task_id}}",
        implementation: "Build {{task_id}}",
        batch: "Batch {{task_list}}",
        supportedPlaceholders: ["task_id", "feature_id"],
        requiredPlaceholder: "task_id",
        batchSupportedPlaceholders: ["task_list", "feature_id"],
        batchRequiredPlaceholder: "task_list",
        builtIn: {
          design: "Built-in design {{task_id}}",
          implementation: "Built-in build {{task_id}}",
          batch: "Built-in batch {{task_list}}",
        },
      },
      isPending: false,
      error: null as Error | null,
    },
    mutations: {
      updatePrompts: mutation(),
      addHost: mutation(),
      updateHost: mutation(),
      deleteHost: mutation(),
      addAuth: mutation(),
      updateAuth: mutation(),
      deleteAuth: mutation(),
      reorderAuth: mutation(),
      updateSync: mutation(),
    },
  };
});

vi.mock("../src/api", () => ({
  configMutations: settingsMocks.api,
  promptMutations: { updateTemplates: vi.fn() },
}));
vi.mock("../src/hooks", () => ({
  useConfig: () => settingsMocks.config,
  useDebugReport: () => settingsMocks.debugReport,
  useQueryDiagnostics: () => [{ name: "snapshot", state: "success, idle" }],
  usePromptTemplates: () => settingsMocks.promptTemplates,
  usePromptTemplatesMutation: () => settingsMocks.mutations.updatePrompts,
  useConfigMutation: (mutation: unknown) => {
    const entries: [
      unknown,
      (typeof settingsMocks.mutations)[keyof typeof settingsMocks.mutations],
    ][] = [
      [settingsMocks.api.addHost, settingsMocks.mutations.addHost],
      [settingsMocks.api.updateHost, settingsMocks.mutations.updateHost],
      [settingsMocks.api.deleteHost, settingsMocks.mutations.deleteHost],
      [settingsMocks.api.addAuth, settingsMocks.mutations.addAuth],
      [settingsMocks.api.updateAuth, settingsMocks.mutations.updateAuth],
      [settingsMocks.api.deleteAuth, settingsMocks.mutations.deleteAuth],
      [settingsMocks.api.reorderAuth, settingsMocks.mutations.reorderAuth],
      [settingsMocks.api.updateSync, settingsMocks.mutations.updateSync],
    ];
    return entries.find(([key]) => key === mutation)?.[1];
  },
}));

describe("SettingsDialog", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  beforeEach(async () => {
    localStorage.clear();
    await setDisplayLanguage("en");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => [
          {
            license: "MIT",
            name: "@xyflow/react",
            repository: "https://github.com/xyflow/xyflow",
            source:
              "https://registry.npmjs.org/@xyflow/react/-/react-12.8.5.tgz",
            version: "12.8.5",
          },
          {
            license: "MIT",
            name: "react",
            repository: "https://github.com/facebook/react",
            source: "https://registry.npmjs.org/react/-/react-19.1.1.tgz",
            version: "19.1.1",
          },
        ],
      }),
    );
    settingsMocks.config.isPending = false;
    settingsMocks.config.error = null;
    vi.spyOn(window, "confirm").mockReturnValue(true);
    for (const value of Object.values(settingsMocks.mutations)) {
      value.mutate.mockReset();
      value.mutateAsync.mockReset();
      value.mutateAsync.mockResolvedValue({});
      value.isPending = false;
      value.error = null;
    }
  });

  function hostFieldset(name: string) {
    const legend = screen.getByText(`Settings for ${name}`);
    const fieldset = legend.closest("fieldset");
    if (!(fieldset instanceof HTMLFieldSetElement))
      throw new Error(`fieldset missing for ${name}`);
    return fieldset;
  }

  function authFieldset(name: string) {
    const legend = screen.getByText(`Settings for ${name}`);
    const fieldset = legend.closest("fieldset");
    if (!(fieldset instanceof HTMLFieldSetElement))
      throw new Error(`fieldset missing for ${name}`);
    return fieldset;
  }

  function saveSettings() {
    return act(() => {
      fireEvent.click(screen.getByRole("button", { name: "Save" }));
      return Promise.resolve();
    });
  }

  it("writes every settings draft in one save", async () => {
    const onClose = vi.fn();
    render(<SettingsDialog onClose={onClose} />);

    expect(
      screen.getByRole("dialog", { name: "Settings" }),
    ).toBeInTheDocument();
    // 保存済みの token はサーバーが返さないので、行は空欄と手がかりだけを出す。
    expect(screen.getByText(/gith…cret/)).toBeInTheDocument();
    expect(
      screen.queryByDisplayValue("github_pat_rpc_secret"),
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();

    fireEvent.change(screen.getByLabelText("Interval in seconds"), {
      target: { value: "600" },
    });

    const ghe = hostFieldset("ghe.example.com");
    fireEvent.change(within(ghe).getByLabelText("Host"), {
      target: { value: "ghe-renamed.example.com" },
    });
    fireEvent.change(within(ghe).getByLabelText("Web URL"), {
      target: { value: "https://ghe-renamed.example.com" },
    });
    fireEvent.change(within(ghe).getByLabelText("API URL"), {
      target: { value: "https://ghe-renamed.example.com/api/v3/" },
    });
    fireEvent.change(within(ghe).getByLabelText("Upload URL"), {
      target: { value: "https://ghe-renamed.example.com/api/uploads/" },
    });
    fireEvent.change(within(ghe).getByLabelText("GraphQL URL"), {
      target: { value: "https://ghe-renamed.example.com/api/graphql" },
    });

    fireEvent.click(screen.getByRole("button", { name: "Register a host" }));
    const added = hostFieldset("New host");
    fireEvent.change(within(added).getByLabelText("Host"), {
      target: { value: "ghe-two.example.com" },
    });

    fireEvent.change(
      within(authFieldset("ghe")).getByLabelText("Environment variable"),
      { target: { value: "GHE_TOKEN" } },
    );

    // 認証の並びは上から試す順序なので、移動もフッタの保存でまとめて書く。
    fireEvent.click(screen.getByRole("button", { name: "Move work down" }));
    fireEvent.click(screen.getByRole("button", { name: "Move work up" }));
    fireEvent.click(screen.getByRole("button", { name: "Move ghe up" }));

    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    await saveSettings();

    expect(settingsMocks.mutations.updateSync.mutateAsync).toHaveBeenCalledWith(
      600n,
    );
    expect(settingsMocks.mutations.addHost.mutateAsync).toHaveBeenCalledWith({
      host: "ghe-two.example.com",
      webUrl: "",
      apiUrl: "",
      uploadUrl: "",
      graphqlUrl: "",
    });
    expect(settingsMocks.mutations.updateHost.mutateAsync).toHaveBeenCalledWith(
      {
        host: "ghe.example.com",
        newHost: "ghe-renamed.example.com",
        webUrl: "https://ghe-renamed.example.com",
        apiUrl: "https://ghe-renamed.example.com/api/v3/",
        uploadUrl: "https://ghe-renamed.example.com/api/uploads/",
        graphqlUrl: "https://ghe-renamed.example.com/api/graphql",
      },
    );
    expect(
      settingsMocks.mutations.reorderAuth.mutateAsync,
    ).toHaveBeenCalledWith(["ghe", "work"]);
    // ホスト名を変えたので、そのホストを指していた認証も付け替えて送る。
    expect(settingsMocks.mutations.updateAuth.mutateAsync).toHaveBeenCalledWith(
      {
        id: "ghe",
        newId: "ghe",
        host: "ghe-renamed.example.com",
        type: GithubAuthMethodType.ENVIRONMENT,
        account: "",
        service: "",
        variable: "GHE_TOKEN",
        user: "",
      },
    );
    // token を触っていない inline の認証は変更がないので送らない。空の token を
    // 送ると保存済みの値を消してしまう。
    expect(
      settingsMocks.mutations.updateAuth.mutateAsync,
    ).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Saved")).toBeInTheDocument();
  });

  it("removes a host and its credential in the same save", async () => {
    render(<SettingsDialog onClose={vi.fn()} />);

    const ghe = hostFieldset("ghe.example.com");
    fireEvent.click(
      within(ghe).getByRole("button", { name: "Remove ghe.example.com" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Remove ghe" }));
    await saveSettings();

    // 認証を先に消してから host を消す。逆に送ると、まだ参照されている host の
    // 削除をサーバーが拒む。
    expect(settingsMocks.mutations.deleteAuth.mutateAsync).toHaveBeenCalledWith(
      "ghe",
    );
    expect(settingsMocks.mutations.deleteHost.mutateAsync).toHaveBeenCalledWith(
      "ghe.example.com",
    );
    const deleteAuthOrder =
      settingsMocks.mutations.deleteAuth.mutateAsync.mock
        .invocationCallOrder[0];
    const deleteHostOrder =
      settingsMocks.mutations.deleteHost.mutateAsync.mock
        .invocationCallOrder[0];
    expect(deleteAuthOrder).toBeLessThan(deleteHostOrder ?? 0);
  });

  // github.com はサーバーが常に持つので、行ごと消せてはならない。
  it("keeps the github.com row undeletable", () => {
    render(<SettingsDialog onClose={vi.fn()} />);
    const primary = hostFieldset("github.com");
    expect(
      within(primary).getByRole("button", { name: "Remove github.com" }),
    ).toBeDisabled();
  });

  it("blocks the save while a draft is incomplete", () => {
    render(<SettingsDialog onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "Register a host" }));
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(
      screen.getByText(
        "Every host needs a name, and the names must not repeat.",
      ),
    ).toBeInTheDocument();

    fireEvent.change(within(hostFieldset("New host")).getByLabelText("Host"), {
      target: { value: "ghe-two.example.com" },
    });
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("adds a credential with a new inline token", async () => {
    render(<SettingsDialog onClose={vi.fn()} />);
    fireEvent.click(
      screen.getByRole("button", { name: "Register a credential" }),
    );
    const added = screen
      .getByText("Settings for New credential")
      .closest("fieldset");
    if (!(added instanceof HTMLFieldSetElement))
      throw new Error("credential fieldset missing");

    fireEvent.change(within(added).getByLabelText("Method ID"), {
      target: { value: "new-inline" },
    });
    fireEvent.change(within(added).getByLabelText("Host"), {
      target: { value: "ghe.example.com" },
    });
    fireEvent.change(within(added).getByLabelText("gh CLI user"), {
      target: { value: "work-account" },
    });
    fireEvent.change(within(added).getByLabelText("Credential source"), {
      target: { value: String(GithubAuthMethodType.KEYCHAIN) },
    });
    fireEvent.change(within(added).getByLabelText("Keychain account"), {
      target: { value: "prx" },
    });
    fireEvent.change(within(added).getByLabelText("Keychain service"), {
      target: { value: "github" },
    });
    fireEvent.change(within(added).getByLabelText("Credential source"), {
      target: { value: String(GithubAuthMethodType.INLINE) },
    });
    const token = within(added).getByLabelText("Inline token");
    expect(token).toHaveAttribute("type", "password");
    fireEvent.change(token, { target: { value: "github_pat_new" } });
    await saveSettings();

    expect(settingsMocks.mutations.addAuth.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "new-inline",
        host: "ghe.example.com",
        type: GithubAuthMethodType.INLINE,
        token: "github_pat_new",
      }),
    );
  });

  it("warns before closing with unsaved settings", () => {
    const onClose = vi.fn();
    render(<SettingsDialog onClose={onClose} />);

    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledOnce();

    onClose.mockClear();
    fireEvent.change(screen.getByLabelText("Interval in seconds"), {
      target: { value: "1200" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).not.toHaveBeenCalled();
    expect(
      screen.getByRole("dialog", { name: "Discard unsaved changes?" }),
    ).toBeInTheDocument();
    // 確認をやめたら編集はそのまま残り、ダイアログも開いたままになる。
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("dialog", { name: "Discard unsaved changes?" }),
    ).not.toBeInTheDocument();
    expect(screen.getByLabelText("Interval in seconds")).toHaveValue(1200);

    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    fireEvent.click(screen.getByRole("button", { name: "Discard" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("keeps server drafts mounted while navigating tabs by keyboard", () => {
    render(<SettingsDialog onClose={vi.fn()} />);
    const serverTab = screen.getByRole("tab", { name: "Server" });
    const promptsTab = screen.getByRole("tab", { name: "Prompts" });
    const displayTab = screen.getByRole("tab", { name: "Display" });
    const licensesTab = screen.getByRole("tab", { name: "Licenses" });
    fireEvent.change(
      within(hostFieldset("github.com")).getByLabelText("Host"),
      {
        target: { value: "draft.example.com" },
      },
    );

    fireEvent.keyDown(serverTab, { key: "ArrowRight" });
    expect(promptsTab).toHaveFocus();
    fireEvent.keyDown(promptsTab, { key: "ArrowRight" });
    expect(displayTab).toHaveFocus();
    expect(displayTab).toHaveAttribute("aria-selected", "true");
    expect(
      screen.getByRole("combobox", { name: "Display language" }),
    ).toBeInTheDocument();

    fireEvent.keyDown(displayTab, { key: "Home" });
    expect(serverTab).toHaveFocus();
    expect(
      within(hostFieldset("draft.example.com")).getByLabelText("Host"),
    ).toHaveValue("draft.example.com");

    fireEvent.keyDown(serverTab, { key: "End" });
    expect(licensesTab).toHaveFocus();
    fireEvent.keyDown(licensesTab, { key: "ArrowRight" });
    expect(serverTab).toHaveFocus();
    fireEvent.keyDown(serverTab, { key: "ArrowLeft" });
    expect(licensesTab).toHaveFocus();
  });

  it("keeps unsaved prompt edits while navigating away and back", () => {
    render(<SettingsDialog onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole("tab", { name: "Prompts" }));
    const design = screen.getByLabelText(/Design prompt/);
    fireEvent.change(design, { target: { value: "Draft {{task_id}}" } });

    fireEvent.click(screen.getByRole("tab", { name: "Display" }));
    fireEvent.click(screen.getByRole("tab", { name: "Prompts" }));

    expect(screen.getByLabelText(/Design prompt/)).toHaveValue(
      "Draft {{task_id}}",
    );
  });

  it("lists bundled OSS packages and their licenses", async () => {
    render(<SettingsDialog onClose={vi.fn()} />);

    fireEvent.click(screen.getByRole("tab", { name: "Licenses" }));

    const panel = screen.getByRole("tabpanel", { name: "Licenses" });
    expect(panel).toBeVisible();
    expect(
      await within(panel).findByRole("link", {
        name: "@xyflow/react@12.8.5",
      }),
    ).toHaveAttribute("href", "https://github.com/xyflow/xyflow");
    expect(within(panel).getAllByText("MIT")).toHaveLength(2);
    expect(within(panel).getAllByRole("link")).toHaveLength(2);
  });

  it("keeps display settings available while server settings load", () => {
    settingsMocks.config.isPending = true;
    render(<SettingsDialog onClose={vi.fn()} />);
    expect(screen.getByText("Loading server settings…")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Display" }));
    expect(
      screen.getByRole("combobox", { name: "Display language" }),
    ).toBeInTheDocument();
  });

  // 間隔のフォームは自前の mutation を持つので、その失敗はダイアログ共通の
  // エラー表示には出ない。
  it("shows why saving the synchronization interval failed", () => {
    settingsMocks.mutations.updateSync.error = new Error(
      "config file is read-only",
    );
    render(<SettingsDialog onClose={vi.fn()} />);
    expect(screen.getByText("config file is read-only")).toBeInTheDocument();
  });
});
