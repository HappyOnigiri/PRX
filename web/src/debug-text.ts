import { webUISettingsKey } from "./i18n/settings";

export interface QueryDiagnostic {
  name: string;
  state: string;
}

const versionMetaSelector = 'meta[name="prx-version"]';

// formatBrowserDebugSection は、サーバーが出力したレポートにブラウザしか知らな
// い情報を追記する。ここの値はすべてこのタブの事実。全体が 1 つのレポートとして
// 読めるよう、体裁はサーバーの流儀に合わせる。
export function formatBrowserDebugSection(queries: QueryDiagnostic[]): string {
  const injected = serverVersion();
  // bundle のバージョンは appVersion を介さず直接読む。appVersion は注入値を
  // 優先するため、同じ値どうしを比較してしまう。
  const bundle = import.meta.env.APP_VERSION;
  const lines = [
    "",
    "browser:",
    `  route: ${window.location.pathname}${window.location.search}`,
    `  server_version: ${injected || "unset"}`,
    `  bundle_version: ${bundle}`,
    // 不一致なら、このタブは古いビルドのキャッシュ済み bundle で動いている。
    `  version_match: ${injected === bundle ? "yes" : "no"}`,
    `  user_agent: ${navigator.userAgent}`,
    `  viewport: ${window.innerWidth}x${window.innerHeight}`,
    `  local_storage: ${readStoredSettings()}`,
    "  queries:",
  ];
  for (const query of queries) lines.push(`    ${query.name}: ${query.state}`);
  return `${lines.join("\n")}\n`;
}

function serverVersion(): string {
  return (
    document
      .querySelector<HTMLMetaElement>(versionMetaSelector)
      ?.content.trim() ?? ""
  );
}

function readStoredSettings(): string {
  try {
    return localStorage.getItem(webUISettingsKey) ?? "unset";
  } catch {
    return "unavailable";
  }
}
