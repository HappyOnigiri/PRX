import { ChevronDown, ChevronUp, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { GithubAuthMethodType } from "../gen/prx/v1/prx_pb";
import { IconButton } from "./IconButton";
import {
  emptyAuthDraft,
  invalidAuthKeys,
  type AuthDraft,
  type ServerDraft,
} from "./serverSettingsDraft";

export function AuthSettingsSection({
  draft,
  onChange,
}: {
  draft: ServerDraft;
  onChange: (next: ServerDraft) => void;
}) {
  const { t } = useTranslation();
  const invalid = new Set(invalidAuthKeys(draft));
  const defaultHost = draft.hosts[0]?.host ?? "github.com";

  function replace(auths: AuthDraft[]) {
    onChange({ ...draft, auths });
  }

  function move(index: number, direction: -1 | 1) {
    const target = index + direction;
    const auths = [...draft.auths];
    const current = auths[index];
    const swapped = auths[target];
    if (!current || !swapped) return;
    auths[index] = swapped;
    auths[target] = current;
    replace(auths);
  }

  return (
    <section className="settings-section" aria-labelledby="settings-auth">
      <header>
        <h3 id="settings-auth">{t("serverSettings.authTitle")}</h3>
      </header>
      <div className="settings-draft-list">
        {draft.auths.map((auth, index) => (
          <AuthFields
            key={auth.key}
            auth={auth}
            hosts={draft.hosts.map((host) => host.host)}
            invalid={invalid.has(auth.key)}
            order={index + 1}
            first={index === 0}
            last={index === draft.auths.length - 1}
            onChange={(next) => {
              replace(
                draft.auths.map((item) =>
                  item.key === auth.key ? next : item,
                ),
              );
            }}
            onMove={(direction) => {
              move(index, direction);
            }}
            onRemove={() => {
              replace(draft.auths.filter((item) => item.key !== auth.key));
            }}
          />
        ))}
      </div>
      {draft.auths.length === 0 && (
        <p className="settings-empty">{t("serverSettings.noAuth")}</p>
      )}
      {invalid.size > 0 && (
        <p className="form-error">{t("serverSettings.authInvalid")}</p>
      )}
      <div className="settings-form-actions">
        <IconButton
          icon={Plus}
          label={t("serverSettings.addAuth")}
          variant="secondary"
          onClick={() => {
            replace([...draft.auths, emptyAuthDraft(defaultHost)]);
          }}
        />
      </div>
    </section>
  );
}

function AuthFields({
  auth,
  hosts,
  invalid,
  order,
  first,
  last,
  onChange,
  onMove,
  onRemove,
}: {
  auth: AuthDraft;
  hosts: string[];
  invalid: boolean;
  order: number;
  first: boolean;
  last: boolean;
  onChange: (next: AuthDraft) => void;
  onMove: (direction: -1 | 1) => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const name = auth.id || t("serverSettings.newAuth");
  return (
    <fieldset className="settings-draft-item">
      {/* 認証は上から順に試すので、この番号は読み手が使える情報である。 */}
      <legend>
        <span className="settings-order">{String(order).padStart(2, "0")}</span>
        {t("serverSettings.authFields", { id: name })}
      </legend>
      <div className="form-row">
        <label>
          {t("serverSettings.authId")}
          <input
            value={auth.id}
            aria-invalid={invalid}
            onChange={(event) => {
              onChange({ ...auth, id: event.target.value });
            }}
          />
        </label>
        <label>
          {t("serverSettings.host")}
          <select
            value={auth.host}
            onChange={(event) => {
              onChange({ ...auth, host: event.target.value });
            }}
          >
            {/* 下書きから消したホストを指したままの認証も、値を見せて直せる
                ようにする。 */}
            {!hosts.includes(auth.host) && (
              <option value={auth.host}>{auth.host}</option>
            )}
            {hosts.map((host) => (
              <option value={host} key={host}>
                {host}
              </option>
            ))}
          </select>
        </label>
      </div>
      <label>
        {t("serverSettings.authType")}
        <select
          value={auth.type}
          onChange={(event) => {
            onChange({ ...auth, type: Number(event.target.value) });
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
      <AuthSourceFields auth={auth} onChange={onChange} />
      <div className="settings-draft-actions">
        <IconButton
          icon={ChevronUp}
          label={t("serverSettings.moveUp", { id: name })}
          variant="secondary"
          size="compact"
          iconOnly
          disabled={first}
          onClick={() => {
            onMove(-1);
          }}
        />
        <IconButton
          icon={ChevronDown}
          label={t("serverSettings.moveDown", { id: name })}
          variant="secondary"
          size="compact"
          iconOnly
          disabled={last}
          onClick={() => {
            onMove(1);
          }}
        />
        <IconButton
          icon={Trash2}
          label={t("serverSettings.removeAuthAction", { id: name })}
          variant="danger"
          size="compact"
          iconOnly
          onClick={onRemove}
        />
      </div>
    </fieldset>
  );
}

function AuthSourceFields({
  auth,
  onChange,
}: {
  auth: AuthDraft;
  onChange: (next: AuthDraft) => void;
}) {
  const { t } = useTranslation();
  if (auth.type === GithubAuthMethodType.KEYCHAIN) {
    return (
      <div className="form-row">
        <label>
          {t("serverSettings.account")}
          <input
            value={auth.account}
            onChange={(event) => {
              onChange({ ...auth, account: event.target.value });
            }}
          />
        </label>
        <label>
          {t("serverSettings.service")}
          <input
            value={auth.service}
            onChange={(event) => {
              onChange({ ...auth, service: event.target.value });
            }}
          />
        </label>
      </div>
    );
  }
  if (auth.type === GithubAuthMethodType.ENVIRONMENT) {
    return (
      <label>
        {t("serverSettings.variable")}
        <input
          value={auth.variable}
          placeholder="GH_ENTERPRISE_TOKEN"
          onChange={(event) => {
            onChange({ ...auth, variable: event.target.value });
          }}
        />
      </label>
    );
  }
  if (auth.type === GithubAuthMethodType.INLINE) {
    return (
      <label>
        {t("serverSettings.token")}
        <input
          type="password"
          autoComplete="new-password"
          value={auth.token}
          placeholder={
            auth.original ? t("serverSettings.tokenKeep") : "github_pat_…"
          }
          onChange={(event) => {
            onChange({ ...auth, token: event.target.value });
          }}
        />
        {auth.secretHint && <small>{auth.secretHint}</small>}
      </label>
    );
  }
  return (
    <label>
      {t("serverSettings.user")}
      <input
        value={auth.user}
        placeholder={t("serverSettings.userOptional")}
        onChange={(event) => {
          onChange({ ...auth, user: event.target.value });
        }}
      />
    </label>
  );
}
