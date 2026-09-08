import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { configMutations } from "../api";
import { useConfig, useConfigMutation } from "../hooks";
import { AuthSettingsSection } from "./ServerAuthSettings";
import { HostSettingsSection } from "./ServerHostSettings";
import {
  minimumSyncSeconds,
  serverDraftDirty,
  serverDraftFromConfig,
  serverDraftInvalid,
  serverSavePlan,
  syncSecondsInvalid,
  type AuthDraft,
  type HostDraft,
  type ServerDraft,
} from "./serverSettingsDraft";
import { useRegisterSettingsSection } from "./settingsSections";

export function ServerSettingsPanel() {
  const { t } = useTranslation();
  const config = useConfig();
  const addHost = useConfigMutation(configMutations.addHost);
  const updateHost = useConfigMutation(configMutations.updateHost);
  const deleteHost = useConfigMutation(configMutations.deleteHost);
  const addAuth = useConfigMutation(configMutations.addAuth);
  const updateAuth = useConfigMutation(configMutations.updateAuth);
  const deleteAuth = useConfigMutation(configMutations.deleteAuth);
  const reorderAuth = useConfigMutation(configMutations.reorderAuth);
  const updateSync = useConfigMutation(configMutations.updateSync);
  const [edited, setEdited] = useState<ServerDraft>();

  const base = useMemo(
    () => (config.data ? serverDraftFromConfig(config.data) : undefined),
    [config.data],
  );
  // 編集を始めるまではサーバーの設定をそのまま見せる。編集後は再取得で上書き
  // せず下書きを残す。保存前の編集が黙って消えてはならないため。
  const draft = edited ?? base;

  const dirty = Boolean(draft && base && serverDraftDirty(draft, base));
  const invalid = Boolean(draft && serverDraftInvalid(draft));

  async function save() {
    if (!draft || !base) return;
    const plan = serverSavePlan(draft, base);
    if (plan.syncSeconds !== undefined)
      await updateSync.mutateAsync(BigInt(plan.syncSeconds));
    for (const host of plan.addHosts)
      await addHost.mutateAsync({ host: host.host, ...urls(host) });
    for (const { host, draft: next } of plan.updateHosts)
      await updateHost.mutateAsync({ host, newHost: next.host, ...urls(next) });
    for (const auth of plan.addAuths)
      await addAuth.mutateAsync(authInput(auth));
    for (const { id, draft: next } of plan.updateAuths)
      await updateAuth.mutateAsync({ ...authInput(next), id, newId: next.id });
    for (const id of plan.deleteAuths) await deleteAuth.mutateAsync(id);
    for (const host of plan.deleteHosts) await deleteHost.mutateAsync(host);
    if (plan.reorderAuths) await reorderAuth.mutateAsync(plan.reorderAuths);
    // 保存した後は、サーバーが返す設定をそのまま見せる状態に戻す。
    setEdited(undefined);
  }

  useRegisterSettingsSection("server", { dirty, invalid, save });

  const error =
    config.error ??
    addHost.error ??
    updateHost.error ??
    deleteHost.error ??
    addAuth.error ??
    updateAuth.error ??
    deleteAuth.error ??
    reorderAuth.error ??
    updateSync.error ??
    null;

  if (config.isPending || !draft) {
    return (
      <p className="settings-panel-state">
        {config.error ? config.error.message : t("serverSettings.loading")}
      </p>
    );
  }

  return (
    <>
      <AutoSyncSettings draft={draft} onChange={setEdited} />
      <HostSettingsSection draft={draft} onChange={setEdited} />
      <AuthSettingsSection draft={draft} onChange={setEdited} />
      {error && <p className="form-error">{error.message}</p>}
    </>
  );
}

function urls(draft: HostDraft) {
  return {
    webUrl: draft.webUrl,
    apiUrl: draft.apiUrl,
    uploadUrl: draft.uploadUrl,
    graphqlUrl: draft.graphqlUrl,
  };
}

// token は入力があったときだけ送る。空のまま送ると保存済みの値を消してしまう。
function authInput(draft: AuthDraft) {
  return {
    id: draft.id,
    host: draft.host,
    type: draft.type,
    account: draft.account,
    service: draft.service,
    variable: draft.variable,
    user: draft.user,
    ...(draft.token ? { token: draft.token } : {}),
  };
}

function AutoSyncSettings({
  draft,
  onChange,
}: {
  draft: ServerDraft;
  onChange: (next: ServerDraft) => void;
}) {
  const { t } = useTranslation();
  const invalid = syncSecondsInvalid(draft.syncSeconds);
  return (
    <section className="settings-section" aria-labelledby="settings-sync">
      <header>
        <h3 id="settings-sync">{t("serverSettings.syncTitle")}</h3>
      </header>
      <label className="settings-sync-field">
        {t("serverSettings.syncInterval")}
        <input
          type="number"
          min={minimumSyncSeconds}
          step={1}
          value={Number.isNaN(draft.syncSeconds) ? "" : draft.syncSeconds}
          aria-invalid={invalid}
          onChange={(event) => {
            onChange({
              ...draft,
              syncSeconds: event.target.valueAsNumber,
            });
          }}
        />
      </label>
      {invalid && (
        <p className="form-error">{t("serverSettings.syncInvalid")}</p>
      )}
      <small>{t("serverSettings.syncHint")}</small>
    </section>
  );
}
