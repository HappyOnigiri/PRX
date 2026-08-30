import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
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
    mutations: {
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

vi.mock("../src/api", () => ({ configMutations: settingsMocks.api }));
vi.mock("../src/hooks", () => ({
  useConfig: () => settingsMocks.config,
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

  it("submits host changes, reorders credentials, and keeps an omitted inline token", async () => {
    const onClose = vi.fn();
    render(<SettingsDialog onClose={onClose} />);

    expect(
      screen.getByRole("dialog", { name: "Settings" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/gith…cret/)).toBeInTheDocument();
    expect(
      screen.queryByDisplayValue("github_pat_rpc_secret"),
    ).not.toBeInTheDocument();

    const syncForm = screen
      .getByLabelText("Interval in seconds")
      .closest("form");
    if (!(syncForm instanceof HTMLFormElement))
      throw new Error("sync form missing");
    fireEvent.change(within(syncForm).getByLabelText("Interval in seconds"), {
      target: { value: "600" },
    });
    fireEvent.submit(syncForm);
    await waitFor(() => {
      expect(settingsMocks.mutations.updateSync.mutate).toHaveBeenCalledWith(
        600n,
      );
    });

    const hostForm = screen
      .getByRole("heading", { name: "Register a host" })
      .closest("form");
    if (!(hostForm instanceof HTMLFormElement))
      throw new Error("host form missing");
    fireEvent.change(within(hostForm).getByLabelText("Host"), {
      target: { value: "ghe-two.example.com" },
    });
    fireEvent.submit(hostForm);
    await waitFor(() => {
      expect(settingsMocks.mutations.addHost.mutateAsync).toHaveBeenCalledWith({
        host: "ghe-two.example.com",
        webUrl: "",
        apiUrl: "",
        uploadUrl: "",
        graphqlUrl: "",
      });
    });

    const hostRow = screen
      .getByText("ghe.example.com", { selector: "strong" })
      .closest(".settings-row");
    if (!(hostRow instanceof HTMLElement)) throw new Error("host row missing");
    fireEvent.click(
      within(hostRow).getByRole("button", { name: "Edit ghe.example.com" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(
      within(hostRow).getByRole("button", { name: "Edit ghe.example.com" }),
    );
    const editHostForm = screen
      .getByRole("heading", { name: "Edit host" })
      .closest("form");
    if (!(editHostForm instanceof HTMLFormElement)) {
      throw new Error("host edit form missing");
    }
    fireEvent.change(within(editHostForm).getByLabelText("Host"), {
      target: { value: "ghe-renamed.example.com" },
    });
    fireEvent.change(within(editHostForm).getByLabelText("Web URL"), {
      target: { value: "https://ghe-renamed.example.com" },
    });
    fireEvent.change(within(editHostForm).getByLabelText("API URL"), {
      target: { value: "https://ghe-renamed.example.com/api/v3/" },
    });
    fireEvent.change(within(editHostForm).getByLabelText("Upload URL"), {
      target: { value: "https://ghe-renamed.example.com/api/uploads/" },
    });
    fireEvent.change(within(editHostForm).getByLabelText("GraphQL URL"), {
      target: { value: "https://ghe-renamed.example.com/api/graphql" },
    });
    fireEvent.submit(editHostForm);
    await waitFor(() => {
      expect(
        settingsMocks.mutations.updateHost.mutateAsync,
      ).toHaveBeenCalledWith({
        host: "ghe.example.com",
        newHost: "ghe-renamed.example.com",
        webUrl: "https://ghe-renamed.example.com",
        apiUrl: "https://ghe-renamed.example.com/api/v3/",
        uploadUrl: "https://ghe-renamed.example.com/api/uploads/",
        graphqlUrl: "https://ghe-renamed.example.com/api/graphql",
      });
    });
    fireEvent.click(
      within(hostRow).getByRole("button", {
        name: "Remove ghe.example.com",
      }),
    );
    await waitFor(() => {
      expect(
        settingsMocks.mutations.deleteHost.mutateAsync,
      ).toHaveBeenCalledWith("ghe.example.com");
    });

    const editButtons = screen.getAllByRole("button", { name: /^Edit / });
    const workEditButton = editButtons[2];
    if (!workEditButton) throw new Error("work edit button missing");
    fireEvent.click(workEditButton);
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    const refreshedEditButtons = screen.getAllByRole("button", {
      name: /^Edit /,
    });
    const refreshedWorkEditButton = refreshedEditButtons[2];
    if (!refreshedWorkEditButton) throw new Error("work edit button missing");
    fireEvent.click(refreshedWorkEditButton);
    const authForm = screen
      .getByRole("heading", { name: "Edit credential" })
      .closest("form");
    if (!(authForm instanceof HTMLFormElement))
      throw new Error("auth form missing");
    const token = within(authForm).getByLabelText("Inline token");
    expect(token).toHaveAttribute("type", "password");
    expect(token).toHaveValue("");
    fireEvent.submit(authForm);
    await waitFor(() => {
      expect(
        settingsMocks.mutations.updateAuth.mutateAsync,
      ).toHaveBeenCalledWith({
        id: "work",
        host: "github.com",
        type: GithubAuthMethodType.INLINE,
        account: "",
        service: "",
        variable: "",
        user: "",
      });
    });

    const firstMoveDown = screen.getByRole("button", {
      name: "Move work down",
    });
    fireEvent.click(firstMoveDown);
    await waitFor(() => {
      expect(
        settingsMocks.mutations.reorderAuth.mutateAsync,
      ).toHaveBeenCalledWith(["ghe", "work"]);
    });
    const secondMoveUp = screen.getByRole("button", {
      name: "Move ghe up",
    });
    fireEvent.click(secondMoveUp);
    await waitFor(() => {
      expect(
        settingsMocks.mutations.reorderAuth.mutateAsync,
      ).toHaveBeenLastCalledWith(["ghe", "work"]);
    });
    const removeButtons = screen.getAllByRole("button", { name: /^Remove / });
    const lastRemove = removeButtons.at(-1);
    if (!lastRemove) throw new Error("remove button missing");
    fireEvent.click(lastRemove);
    fireEvent.click(screen.getByRole("button", { name: "Done" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("switches the credential form by source and sends a new inline token", async () => {
    render(<SettingsDialog onClose={vi.fn()} />);
    const authForm = screen
      .getByRole("heading", { name: "Register a credential" })
      .closest("form");
    if (!(authForm instanceof HTMLFormElement))
      throw new Error("auth form missing");
    const type = within(authForm).getByLabelText("Credential source");
    fireEvent.change(within(authForm).getByLabelText("Host"), {
      target: { value: "ghe.example.com" },
    });
    fireEvent.change(within(authForm).getByLabelText("gh CLI user"), {
      target: { value: "work-account" },
    });
    fireEvent.change(type, {
      target: { value: String(GithubAuthMethodType.KEYCHAIN) },
    });
    fireEvent.change(within(authForm).getByLabelText("Keychain account"), {
      target: { value: "prx" },
    });
    fireEvent.change(within(authForm).getByLabelText("Keychain service"), {
      target: { value: "github" },
    });
    fireEvent.change(type, {
      target: { value: String(GithubAuthMethodType.ENVIRONMENT) },
    });
    fireEvent.change(within(authForm).getByLabelText("Environment variable"), {
      target: { value: "GITHUB_TOKEN" },
    });
    fireEvent.change(type, {
      target: { value: String(GithubAuthMethodType.INLINE) },
    });
    const token = within(authForm).getByLabelText("Inline token");
    fireEvent.change(within(authForm).getByLabelText("Method ID"), {
      target: { value: "new-inline" },
    });
    fireEvent.change(token, { target: { value: "github_pat_new" } });
    fireEvent.submit(authForm);
    await waitFor(() => {
      expect(settingsMocks.mutations.addAuth.mutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          id: "new-inline",
          type: GithubAuthMethodType.INLINE,
          token: "github_pat_new",
        }),
      );
    });
  });

  it("keeps server drafts mounted while navigating tabs by keyboard", () => {
    render(<SettingsDialog onClose={vi.fn()} />);
    const serverTab = screen.getByRole("tab", { name: "Server" });
    const displayTab = screen.getByRole("tab", { name: "Display" });
    const licensesTab = screen.getByRole("tab", { name: "Licenses" });
    const hostForm = screen
      .getByRole("heading", { name: "Register a host" })
      .closest("form");
    if (!(hostForm instanceof HTMLFormElement))
      throw new Error("host form missing");
    fireEvent.change(within(hostForm).getByLabelText("Host"), {
      target: { value: "draft.example.com" },
    });

    fireEvent.keyDown(serverTab, { key: "ArrowRight" });
    expect(displayTab).toHaveFocus();
    expect(displayTab).toHaveAttribute("aria-selected", "true");
    expect(
      screen.getByRole("combobox", { name: "Display language" }),
    ).toBeInTheDocument();

    fireEvent.keyDown(displayTab, { key: "Home" });
    expect(serverTab).toHaveFocus();
    expect(within(hostForm).getByLabelText("Host")).toHaveValue(
      "draft.example.com",
    );

    fireEvent.keyDown(serverTab, { key: "End" });
    expect(licensesTab).toHaveFocus();
    fireEvent.keyDown(licensesTab, { key: "ArrowRight" });
    expect(serverTab).toHaveFocus();
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

  // The interval form owns its own mutation, so the dialog's shared error area
  // never sees its failures.
  it("shows why saving the synchronization interval failed", () => {
    settingsMocks.mutations.updateSync.error = new Error(
      "config file is read-only",
    );
    render(<SettingsDialog onClose={vi.fn()} />);
    expect(screen.getByText("config file is read-only")).toBeInTheDocument();
  });
});
