import { Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { IconButton } from "./IconButton";
import {
  emptyHostDraft,
  invalidHostKeys,
  type HostDraft,
  type ServerDraft,
} from "./serverSettingsDraft";

// github.com はサーバーが常に持つので、行そのものは消せない。
const permanentHost = "github.com";

export function HostSettingsSection({
  draft,
  onChange,
}: {
  draft: ServerDraft;
  onChange: (next: ServerDraft) => void;
}) {
  const { t } = useTranslation();
  const invalid = new Set(invalidHostKeys(draft));
  return (
    <section className="settings-section" aria-labelledby="settings-hosts">
      <header>
        <h3 id="settings-hosts">{t("serverSettings.hostsTitle")}</h3>
      </header>
      <div className="settings-draft-list">
        {draft.hosts.map((host) => (
          <HostFields
            key={host.key}
            host={host}
            invalid={invalid.has(host.key)}
            onChange={(next) => {
              onChange({
                ...draft,
                hosts: draft.hosts.map((item) =>
                  item.key === host.key ? next : item,
                ),
                // ホスト名を変えたら、そのホストを指していた認証も付け替える。
                // 名前だけの変更で認証が宛先を失ってはならない。
                auths:
                  host.host === next.host
                    ? draft.auths
                    : draft.auths.map((auth) =>
                        auth.host === host.host
                          ? { ...auth, host: next.host }
                          : auth,
                      ),
              });
            }}
            onRemove={() => {
              onChange({
                ...draft,
                hosts: draft.hosts.filter((item) => item.key !== host.key),
              });
            }}
          />
        ))}
      </div>
      {invalid.size > 0 && (
        <p className="form-error">{t("serverSettings.hostsInvalid")}</p>
      )}
      <div className="settings-form-actions">
        <IconButton
          icon={Plus}
          label={t("serverSettings.addHost")}
          variant="secondary"
          onClick={() => {
            onChange({ ...draft, hosts: [...draft.hosts, emptyHostDraft()] });
          }}
        />
      </div>
    </section>
  );
}

function HostFields({
  host,
  invalid,
  onChange,
  onRemove,
}: {
  host: HostDraft;
  invalid: boolean;
  onChange: (next: HostDraft) => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const name = host.host || t("serverSettings.newHost");
  return (
    <fieldset className="settings-draft-item">
      <legend>{t("serverSettings.hostFields", { host: name })}</legend>
      <div className="form-row">
        <label>
          {t("serverSettings.host")}
          <input
            value={host.host}
            aria-invalid={invalid}
            onChange={(event) => {
              onChange({ ...host, host: event.target.value });
            }}
          />
        </label>
        <label>
          {t("serverSettings.webUrl")}
          <input
            value={host.webUrl}
            placeholder="https://ghe.example.com"
            onChange={(event) => {
              onChange({ ...host, webUrl: event.target.value });
            }}
          />
        </label>
      </div>
      <label>
        {t("serverSettings.graphqlUrl")}
        <input
          value={host.graphqlUrl}
          placeholder="https://ghe.example.com/api/graphql"
          onChange={(event) => {
            onChange({ ...host, graphqlUrl: event.target.value });
          }}
        />
      </label>
      <div className="form-row">
        <label>
          {t("serverSettings.apiUrl")}
          <input
            value={host.apiUrl}
            placeholder="https://ghe.example.com/api/v3/"
            onChange={(event) => {
              onChange({ ...host, apiUrl: event.target.value });
            }}
          />
        </label>
        <label>
          {t("serverSettings.uploadUrl")}
          <input
            value={host.uploadUrl}
            placeholder="https://ghe.example.com/api/uploads/"
            onChange={(event) => {
              onChange({ ...host, uploadUrl: event.target.value });
            }}
          />
        </label>
      </div>
      <div className="settings-draft-actions">
        <IconButton
          icon={Trash2}
          label={t("serverSettings.removeHostAction", { host: name })}
          variant="danger"
          size="compact"
          iconOnly
          disabled={host.original === permanentHost}
          onClick={onRemove}
        />
      </div>
    </fieldset>
  );
}
