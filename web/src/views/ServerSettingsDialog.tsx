import { useState, type ReactNode, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { configMutations } from "../api";
import {
  GithubAuthMethodType,
  type GitHubAuthMethod,
  type GitHubHost,
} from "../gen/prx/v1/prx_pb";
import { useConfig, useConfigMutation } from "../hooks";

interface AuthDraft {
  id: string;
  host: string;
  type: GithubAuthMethodType;
  account: string;
  service: string;
  variable: string;
  user: string;
  token: string;
}

interface HostDraft {
  host: string;
  webUrl: string;
  apiUrl: string;
  uploadUrl: string;
}

interface HostController {
  draft: HostDraft;
  editingHost: string | undefined;
  pending: boolean;
  error: Error | null;
  beginEdit(host: GitHubHost): void;
  reset(): void;
  setDraft(draft: HostDraft): void;
  submit(event: SyntheticEvent<HTMLFormElement>): Promise<void>;
  remove(host: string): Promise<void>;
}

interface AuthController {
  draft: AuthDraft | undefined;
  editingAuth: string | undefined;
  pending: boolean;
  error: Error | null;
  beginEdit(method: GitHubAuthMethod): void;
  reset(): void;
  setDraft(draft: AuthDraft): void;
  submit(event: SyntheticEvent<HTMLFormElement>): Promise<void>;
  remove(id: string): Promise<void>;
  move(index: number, direction: -1 | 1): Promise<void>;
}

interface HostFormProps {
  controller: HostController;
}

interface AuthFormProps {
  hosts: GitHubHost[];
  controller: AuthController;
  defaultHost: string;
}

const emptyHostDraft: HostDraft = {
  host: "",
  webUrl: "",
  apiUrl: "",
  uploadUrl: "",
};

function emptyAuthDraft(host = ""): AuthDraft {
  return {
    id: "",
    host,
    type: GithubAuthMethodType.GH_CLI,
    account: "",
    service: "",
    variable: "",
    user: "",
    token: "",
  };
}

export function ServerSettingsDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const config = useConfig();
  const hosts = config.data?.hosts ?? [];
  const authMethods = config.data?.authMethods ?? [];
  const defaultHost = hosts[0]?.host ?? "github.com";
  const hostController = useHostController();
  const authController = useAuthController(authMethods);
  const error =
    config.error ?? hostController.error ?? authController.error ?? null;

  if (config.isPending) {
    return (
      <SettingsFrame onClose={onClose} title={t("serverSettings.title")}>
        <p>{t("serverSettings.loading")}</p>
      </SettingsFrame>
    );
  }

  return (
    <SettingsFrame onClose={onClose} title={t("serverSettings.title")}>
      <p className="dialog-lead">{t("serverSettings.description")}</p>
      <HostSettingsSection hosts={hosts} controller={hostController} />
      <AuthSettingsSection
        hosts={hosts}
        authMethods={authMethods}
        controller={authController}
        defaultHost={defaultHost}
      />
      {error && <p className="form-error">{error.message}</p>}
    </SettingsFrame>
  );
}

function useHostController(): HostController {
  const { t } = useTranslation();
  const addHost = useConfigMutation(configMutations.addHost);
  const updateHost = useConfigMutation(configMutations.updateHost);
  const deleteHost = useConfigMutation(configMutations.deleteHost);
  const [draft, setDraft] = useState(emptyHostDraft);
  const [editingHost, setEditingHost] = useState<string>();

  function beginEdit(host: GitHubHost) {
    setEditingHost(host.host);
    setDraft({
      host: host.host,
      webUrl: host.webUrl,
      apiUrl: host.apiUrl,
      uploadUrl: host.uploadUrl,
    });
  }

  function reset() {
    setEditingHost(undefined);
    setDraft(emptyHostDraft);
  }

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (editingHost) {
      await updateHost.mutateAsync({
        host: editingHost,
        newHost: draft.host,
        webUrl: draft.webUrl,
        apiUrl: draft.apiUrl,
        uploadUrl: draft.uploadUrl,
      });
    } else {
      await addHost.mutateAsync({
        host: draft.host,
        webUrl: draft.webUrl,
        apiUrl: draft.apiUrl,
        uploadUrl: draft.uploadUrl,
      });
    }
    reset();
  }

  async function remove(host: string) {
    if (
      host === "github.com" ||
      !window.confirm(t("serverSettings.removeHostConfirm", { host }))
    ) {
      return;
    }
    await deleteHost.mutateAsync(host);
  }

  return {
    draft,
    editingHost,
    pending: addHost.isPending || updateHost.isPending || deleteHost.isPending,
    error: addHost.error ?? updateHost.error ?? deleteHost.error ?? null,
    beginEdit,
    reset,
    setDraft,
    submit,
    remove,
  };
}

function useAuthController(authMethods: GitHubAuthMethod[]): AuthController {
  const { t } = useTranslation();
  const addAuth = useConfigMutation(configMutations.addAuth);
  const updateAuth = useConfigMutation(configMutations.updateAuth);
  const deleteAuth = useConfigMutation(configMutations.deleteAuth);
  const reorderAuth = useConfigMutation(configMutations.reorderAuth);
  const [draft, setDraft] = useState<AuthDraft>();
  const [editingAuth, setEditingAuth] = useState<string>();

  function beginEdit(method: GitHubAuthMethod) {
    setEditingAuth(method.id);
    setDraft({
      id: method.id,
      host: method.host,
      type: method.type,
      account: method.account,
      service: method.service,
      variable: method.variable,
      user: method.user,
      token: "",
    });
  }

  function reset() {
    setEditingAuth(undefined);
    setDraft(undefined);
  }

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!draft) return;
    const input = {
      id: draft.id,
      host: draft.host,
      type: draft.type,
      account: draft.account,
      service: draft.service,
      variable: draft.variable,
      user: draft.user,
      ...(draft.type === GithubAuthMethodType.INLINE && draft.token
        ? { token: draft.token }
        : {}),
    };
    if (editingAuth) {
      await updateAuth.mutateAsync(input);
    } else {
      await addAuth.mutateAsync(input);
    }
    reset();
  }

  async function remove(id: string) {
    if (!window.confirm(t("serverSettings.removeAuthConfirm", { id }))) {
      return;
    }
    await deleteAuth.mutateAsync(id);
  }

  async function move(index: number, direction: -1 | 1) {
    const next = [...authMethods];
    const target = index + direction;
    if (target < 0 || target >= next.length) return;
    const current = next[index];
    const replacement = next[target];
    if (!current || !replacement) return;
    next[index] = replacement;
    next[target] = current;
    await reorderAuth.mutateAsync(next.map((method) => method.id));
  }

  return {
    draft,
    editingAuth,
    pending:
      addAuth.isPending ||
      updateAuth.isPending ||
      deleteAuth.isPending ||
      reorderAuth.isPending,
    error:
      addAuth.error ??
      updateAuth.error ??
      deleteAuth.error ??
      reorderAuth.error ??
      null,
    beginEdit,
    reset,
    setDraft,
    submit,
    remove,
    move,
  };
}

function HostSettingsSection({
  hosts,
  controller,
}: {
  hosts: GitHubHost[];
  controller: HostController;
}) {
  const { t } = useTranslation();
  return (
    <section className="settings-section" aria-labelledby="settings-hosts">
      <header>
        <p className="section-label">{t("serverSettings.hostsLabel")}</p>
        <h3 id="settings-hosts">{t("serverSettings.hostsTitle")}</h3>
      </header>
      <div className="settings-list">
        {hosts.map((host) => (
          <div className="settings-row" key={host.host}>
            <div>
              <strong>{host.host}</strong>
              <small>{host.apiUrl}</small>
            </div>
            <div className="settings-row-actions">
              <button
                className="secondary"
                onClick={() => {
                  controller.beginEdit(host);
                }}
              >
                {t("common.edit")}
              </button>
              <button
                className="secondary"
                disabled={host.host === "github.com" || controller.pending}
                onClick={() => {
                  void controller.remove(host.host);
                }}
              >
                {t("common.remove")}
              </button>
            </div>
          </div>
        ))}
      </div>
      <HostForm controller={controller} />
    </section>
  );
}

function HostForm({ controller }: HostFormProps) {
  const { t } = useTranslation();
  const { draft } = controller;
  return (
    <form
      className="settings-form"
      onSubmit={(event) => {
        void controller.submit(event);
      }}
    >
      <h4>
        {controller.editingHost
          ? t("serverSettings.editHost")
          : t("serverSettings.addHost")}
      </h4>
      <div className="form-row">
        <label>
          {t("serverSettings.host")}
          <input
            required
            value={draft.host}
            onChange={(event) => {
              controller.setDraft({ ...draft, host: event.target.value });
            }}
          />
        </label>
        <label>
          {t("serverSettings.webUrl")}
          <input
            value={draft.webUrl}
            onChange={(event) => {
              controller.setDraft({ ...draft, webUrl: event.target.value });
            }}
            placeholder="https://ghe.example.com"
          />
        </label>
      </div>
      <div className="form-row">
        <label>
          {t("serverSettings.apiUrl")}
          <input
            value={draft.apiUrl}
            onChange={(event) => {
              controller.setDraft({ ...draft, apiUrl: event.target.value });
            }}
            placeholder="https://ghe.example.com/api/v3/"
          />
        </label>
        <label>
          {t("serverSettings.uploadUrl")}
          <input
            value={draft.uploadUrl}
            onChange={(event) => {
              controller.setDraft({ ...draft, uploadUrl: event.target.value });
            }}
            placeholder="https://ghe.example.com/api/uploads/"
          />
        </label>
      </div>
      <div className="settings-form-actions">
        {controller.editingHost && (
          <button
            type="button"
            className="secondary"
            onClick={() => {
              controller.reset();
            }}
          >
            {t("common.cancel")}
          </button>
        )}
        <button disabled={controller.pending}>
          {controller.editingHost ? t("common.save") : t("common.add")}
        </button>
      </div>
    </form>
  );
}

function AuthSettingsSection({
  hosts,
  authMethods,
  controller,
  defaultHost,
}: {
  hosts: GitHubHost[];
  authMethods: GitHubAuthMethod[];
  controller: AuthController;
  defaultHost: string;
}) {
  const { t } = useTranslation();
  return (
    <section className="settings-section" aria-labelledby="settings-auth">
      <header>
        <p className="section-label">{t("serverSettings.authLabel")}</p>
        <h3 id="settings-auth">{t("serverSettings.authTitle")}</h3>
      </header>
      <div className="settings-list">
        {authMethods.map((method, index) => (
          <div className="settings-row" key={method.id}>
            <div className="settings-auth-name">
              <span className="settings-order">
                {String(index + 1).padStart(2, "0")}
              </span>
              <div>
                <strong>{method.id}</strong>
                <small>
                  {method.host} · {authTypeLabel(method.type)}
                  {method.secretHint ? ` · ${method.secretHint}` : ""}
                </small>
              </div>
            </div>
            <div className="settings-row-actions">
              <button
                className="secondary"
                disabled={index === 0 || controller.pending}
                onClick={() => {
                  void controller.move(index, -1);
                }}
                aria-label={t("serverSettings.moveUp")}
              >
                ↑
              </button>
              <button
                className="secondary"
                disabled={
                  index === authMethods.length - 1 || controller.pending
                }
                onClick={() => {
                  void controller.move(index, 1);
                }}
                aria-label={t("serverSettings.moveDown")}
              >
                ↓
              </button>
              <button
                className="secondary"
                disabled={controller.pending}
                onClick={() => {
                  controller.beginEdit(method);
                }}
              >
                {t("common.edit")}
              </button>
              <button
                className="secondary"
                disabled={controller.pending}
                onClick={() => {
                  void controller.remove(method.id);
                }}
              >
                {t("common.remove")}
              </button>
            </div>
          </div>
        ))}
        {authMethods.length === 0 && (
          <p className="settings-empty">{t("serverSettings.noAuth")}</p>
        )}
      </div>
      <AuthForm
        hosts={hosts}
        controller={controller}
        defaultHost={defaultHost}
      />
    </section>
  );
}

function AuthForm({ hosts, controller, defaultHost }: AuthFormProps) {
  const { t } = useTranslation();
  const draft = controller.draft ?? emptyAuthDraft(defaultHost);
  return (
    <form
      className="settings-form"
      onSubmit={(event) => {
        void controller.submit(event);
      }}
    >
      <h4>
        {controller.editingAuth
          ? t("serverSettings.editAuth")
          : t("serverSettings.addAuth")}
      </h4>
      <div className="form-row">
        <label>
          {t("serverSettings.authId")}
          <input
            required
            value={draft.id}
            disabled={Boolean(controller.editingAuth)}
            onChange={(event) => {
              controller.setDraft({ ...draft, id: event.target.value });
            }}
          />
        </label>
        <label>
          {t("serverSettings.host")}
          <select
            required
            value={draft.host}
            onChange={(event) => {
              controller.setDraft({ ...draft, host: event.target.value });
            }}
          >
            {hosts.map((host) => (
              <option value={host.host} key={host.host}>
                {host.host}
              </option>
            ))}
          </select>
        </label>
      </div>
      <label>
        {t("serverSettings.authType")}
        <select
          value={draft.type}
          onChange={(event) => {
            controller.setDraft({
              ...draft,
              type: Number(event.target.value),
            });
          }}
        >
          <option value={GithubAuthMethodType.KEYCHAIN}>
            {t("serverSettings.keychain")}
          </option>
          <option value={GithubAuthMethodType.ENVIRONMENT}>
            {t("serverSettings.environment")}
          </option>
          <option value={GithubAuthMethodType.INLINE}>
            {t("serverSettings.inline")}
          </option>
          <option value={GithubAuthMethodType.GH_CLI}>
            {t("serverSettings.ghCli")}
          </option>
        </select>
      </label>
      <AuthSourceFields
        draft={draft}
        editing={Boolean(controller.editingAuth)}
        controller={controller}
      />
      <div className="settings-form-actions">
        {controller.editingAuth && (
          <button
            type="button"
            className="secondary"
            onClick={() => {
              controller.reset();
            }}
          >
            {t("common.cancel")}
          </button>
        )}
        <button disabled={controller.pending}>
          {controller.editingAuth ? t("common.save") : t("common.add")}
        </button>
      </div>
    </form>
  );
}

function AuthSourceFields({
  draft,
  editing,
  controller,
}: {
  draft: AuthDraft;
  editing: boolean;
  controller: AuthController;
}) {
  const { t } = useTranslation();
  if (draft.type === GithubAuthMethodType.KEYCHAIN) {
    return (
      <div className="form-row">
        <label>
          {t("serverSettings.account")}
          <input
            required
            value={draft.account}
            onChange={(event) => {
              controller.setDraft({ ...draft, account: event.target.value });
            }}
          />
        </label>
        <label>
          {t("serverSettings.service")}
          <input
            required
            value={draft.service}
            onChange={(event) => {
              controller.setDraft({ ...draft, service: event.target.value });
            }}
          />
        </label>
      </div>
    );
  }
  if (draft.type === GithubAuthMethodType.ENVIRONMENT) {
    return (
      <label>
        {t("serverSettings.variable")}
        <input
          required
          value={draft.variable}
          onChange={(event) => {
            controller.setDraft({ ...draft, variable: event.target.value });
          }}
          placeholder="GH_ENTERPRISE_TOKEN"
        />
      </label>
    );
  }
  if (draft.type === GithubAuthMethodType.INLINE) {
    return (
      <label>
        {t("serverSettings.token")}
        <input
          type="password"
          autoComplete="new-password"
          required={!editing}
          value={draft.token}
          onChange={(event) => {
            controller.setDraft({ ...draft, token: event.target.value });
          }}
          placeholder={editing ? t("serverSettings.tokenKeep") : "github_pat_…"}
        />
      </label>
    );
  }
  if (draft.type === GithubAuthMethodType.GH_CLI) {
    return (
      <label>
        {t("serverSettings.user")}
        <input
          value={draft.user}
          onChange={(event) => {
            controller.setDraft({ ...draft, user: event.target.value });
          }}
          placeholder={t("serverSettings.userOptional")}
        />
      </label>
    );
  }
  return null;
}

function SettingsFrame({
  onClose,
  title,
  children,
}: {
  onClose: () => void;
  title: string;
  children: ReactNode;
}) {
  const { t } = useTranslation();
  return (
    <div className="scrim" role="presentation">
      <section
        className="dialog settings-dialog"
        role="dialog"
        aria-modal="true"
        aria-label={title}
      >
        <header className="settings-dialog-head">
          <div>
            <p className="section-label">{t("serverSettings.eyebrow")}</p>
            <h2>{title}</h2>
          </div>
          <button
            className="icon-button"
            onClick={onClose}
            aria-label={t("common.close")}
          >
            ×
          </button>
        </header>
        {children}
        <footer>
          <button className="secondary" onClick={onClose}>
            {t("common.done")}
          </button>
        </footer>
      </section>
    </div>
  );
}

function authTypeLabel(type: GithubAuthMethodType): string {
  switch (type) {
    case GithubAuthMethodType.UNSPECIFIED:
      return "unspecified";
    case GithubAuthMethodType.KEYCHAIN:
      return "keychain";
    case GithubAuthMethodType.ENVIRONMENT:
      return "environment";
    case GithubAuthMethodType.INLINE:
      return "inline";
    case GithubAuthMethodType.GH_CLI:
      return "gh cli";
  }
}
