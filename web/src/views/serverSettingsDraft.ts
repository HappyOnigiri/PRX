import { GithubAuthMethodType, type GitHubConfig } from "../gen/prx/v1/prx_pb";

// 設定ダイアログの保存は 1 つなので、ホストと認証の一覧はまず下書きとして持ち、
// フッタの保存で下書きとサーバーの差分だけを送る。
// docs/design/webui.md を参照。

export const minimumSyncSeconds = 600;

export interface HostDraft {
  key: string;
  // original はサーバー上の現在のホスト名。未保存の行は持たない。
  original?: string;
  host: string;
  webUrl: string;
  apiUrl: string;
  uploadUrl: string;
  graphqlUrl: string;
}

export interface AuthDraft {
  key: string;
  original?: string;
  id: string;
  host: string;
  type: GithubAuthMethodType;
  account: string;
  service: string;
  variable: string;
  user: string;
  // token は保存済みの値をサーバーが返さないので、常に空から始まる。
  token: string;
  // secretHint はサーバーが出す読み取り専用の手がかり。編集の対象ではない。
  secretHint: string;
}

export interface ServerDraft {
  syncSeconds: number;
  hosts: HostDraft[];
  auths: AuthDraft[];
}

export interface ServerSavePlan {
  syncSeconds?: number;
  addHosts: HostDraft[];
  updateHosts: { host: string; draft: HostDraft }[];
  deleteHosts: string[];
  addAuths: AuthDraft[];
  updateAuths: { id: string; draft: AuthDraft }[];
  deleteAuths: string[];
  reorderAuths?: string[];
}

let keyCounter = 0;

function nextDraftKey(): string {
  keyCounter += 1;
  return `draft-${String(keyCounter)}`;
}

export function serverDraftFromConfig(config: GitHubConfig): ServerDraft {
  return {
    syncSeconds: Number(config.autoSyncIntervalSeconds),
    hosts: config.hosts.map((host) => ({
      key: `host:${host.host}`,
      original: host.host,
      host: host.host,
      webUrl: host.webUrl,
      apiUrl: host.apiUrl,
      uploadUrl: host.uploadUrl,
      graphqlUrl: host.graphqlUrl,
    })),
    auths: config.authMethods.map((method) => ({
      key: `auth:${method.id}`,
      original: method.id,
      id: method.id,
      host: method.host,
      type: method.type,
      account: method.account,
      service: method.service,
      variable: method.variable,
      user: method.user,
      token: "",
      secretHint: method.secretHint,
    })),
  };
}

export function emptyHostDraft(): HostDraft {
  return {
    key: nextDraftKey(),
    host: "",
    webUrl: "",
    apiUrl: "",
    uploadUrl: "",
    graphqlUrl: "",
  };
}

export function emptyAuthDraft(host: string): AuthDraft {
  return {
    key: nextDraftKey(),
    id: "",
    host,
    type: GithubAuthMethodType.GH_CLI,
    account: "",
    service: "",
    variable: "",
    user: "",
    token: "",
    secretHint: "",
  };
}

// key は React の一覧のためだけの値なので、比較からは外す。
function comparable(draft: ServerDraft): string {
  return JSON.stringify({
    syncSeconds: draft.syncSeconds,
    hosts: draft.hosts.map((host) => ({ ...host, key: "" })),
    auths: draft.auths.map((auth) => ({ ...auth, key: "" })),
  });
}

export function serverDraftDirty(draft: ServerDraft, base: ServerDraft) {
  return comparable(draft) !== comparable(base);
}

export function invalidHostKeys(draft: ServerDraft): string[] {
  const duplicated = new Set(
    draft.hosts
      .map((host) => host.host)
      .filter((host, index, all) => all.indexOf(host) !== index),
  );
  return draft.hosts
    .filter((host) => !host.host || duplicated.has(host.host))
    .map((host) => host.key);
}

export function invalidAuthKeys(draft: ServerDraft): string[] {
  const hosts = new Set(draft.hosts.map((host) => host.host));
  const duplicated = new Set(
    draft.auths
      .map((auth) => auth.id)
      .filter((id, index, all) => all.indexOf(id) !== index),
  );
  return draft.auths
    .filter(
      (auth) =>
        !auth.id ||
        duplicated.has(auth.id) ||
        !hosts.has(auth.host) ||
        !authSourceComplete(auth),
    )
    .map((auth) => auth.key);
}

function authSourceComplete(auth: AuthDraft): boolean {
  switch (auth.type) {
    case GithubAuthMethodType.KEYCHAIN:
      return Boolean(auth.account && auth.service);
    case GithubAuthMethodType.ENVIRONMENT:
      return Boolean(auth.variable);
    case GithubAuthMethodType.INLINE:
      // 保存済みの認証は token を空にすると現在の値を残すので、新規のときだけ
      // 入力を求める。
      return Boolean(auth.original ?? auth.token);
    case GithubAuthMethodType.GH_CLI:
    case GithubAuthMethodType.UNSPECIFIED:
      return true;
  }
}

export function syncSecondsInvalid(seconds: number): boolean {
  return !Number.isSafeInteger(seconds) || seconds < minimumSyncSeconds;
}

export function serverDraftInvalid(draft: ServerDraft): boolean {
  return (
    syncSecondsInvalid(draft.syncSeconds) ||
    invalidHostKeys(draft).length > 0 ||
    invalidAuthKeys(draft).length > 0
  );
}

function hostChanged(draft: HostDraft, base: HostDraft): boolean {
  return (
    draft.host !== base.host ||
    draft.webUrl !== base.webUrl ||
    draft.apiUrl !== base.apiUrl ||
    draft.uploadUrl !== base.uploadUrl ||
    draft.graphqlUrl !== base.graphqlUrl
  );
}

function authChanged(draft: AuthDraft, base: AuthDraft): boolean {
  return (
    draft.id !== base.id ||
    draft.host !== base.host ||
    draft.type !== base.type ||
    draft.account !== base.account ||
    draft.service !== base.service ||
    draft.variable !== base.variable ||
    draft.user !== base.user ||
    draft.token !== ""
  );
}

export function serverSavePlan(
  draft: ServerDraft,
  base: ServerDraft,
): ServerSavePlan {
  const baseHosts = new Map(
    base.hosts.map((host) => [host.original ?? host.host, host]),
  );
  const baseAuths = new Map(
    base.auths.map((auth) => [auth.original ?? auth.id, auth]),
  );
  const plan: ServerSavePlan = {
    addHosts: [],
    updateHosts: [],
    deleteHosts: [],
    addAuths: [],
    updateAuths: [],
    deleteAuths: [],
  };
  if (draft.syncSeconds !== base.syncSeconds)
    plan.syncSeconds = draft.syncSeconds;

  for (const host of draft.hosts) {
    const previous = host.original ? baseHosts.get(host.original) : undefined;
    if (!previous) plan.addHosts.push(host);
    else if (hostChanged(host, previous))
      plan.updateHosts.push({ host: previous.host, draft: host });
  }
  const keptHosts = new Set(
    draft.hosts.flatMap((host) => (host.original ? [host.original] : [])),
  );
  plan.deleteHosts = [...baseHosts.keys()].filter(
    (host) => !keptHosts.has(host),
  );

  for (const auth of draft.auths) {
    const previous = auth.original ? baseAuths.get(auth.original) : undefined;
    if (!previous) plan.addAuths.push(auth);
    else if (authChanged(auth, previous))
      plan.updateAuths.push({ id: previous.id, draft: auth });
  }
  const keptAuths = new Set(
    draft.auths.flatMap((auth) => (auth.original ? [auth.original] : [])),
  );
  plan.deleteAuths = [...baseAuths.keys()].filter((id) => !keptAuths.has(id));

  // 追加と削除を適用した直後のサーバー側の並びと比べる。末尾に足しただけなら
  // 並べ替えは要らない。
  const applied = [
    ...base.auths
      .filter((auth) => keptAuths.has(auth.original ?? auth.id))
      .map(
        (auth) =>
          draft.auths.find((item) => item.original === auth.original)?.id ??
          auth.id,
      ),
    ...plan.addAuths.map((auth) => auth.id),
  ];
  const order = draft.auths.map((auth) => auth.id);
  if (!sameOrder(order, applied)) plan.reorderAuths = order;

  return plan;
}

function sameOrder(left: string[], right: string[]): boolean {
  return (
    left.length === right.length &&
    left.every((value, index) => value === right[index])
  );
}
