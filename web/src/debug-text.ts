import { webUISettingsKey } from "./i18n/settings";

export interface QueryDiagnostic {
  name: string;
  state: string;
}

const versionMetaSelector = 'meta[name="prx-version"]';

// formatBrowserDebugSection appends what only the browser knows to the report
// the server rendered. It is not a second derivation of server state: every
// value here is a fact about this tab that the server cannot observe.
// The layout follows the server's own convention so the whole text reads as one
// report: a section header at the left margin and two-space indented fields.
export function formatBrowserDebugSection(queries: QueryDiagnostic[]): string {
  const injected = serverVersion();
  // The bundle version is read directly rather than through appVersion, which
  // prefers the injected value and would compare a version with itself.
  const bundle = import.meta.env.APP_VERSION;
  const lines = [
    "",
    "browser:",
    `  route: ${window.location.pathname}${window.location.search}`,
    `  server_version: ${injected || "unset"}`,
    `  bundle_version: ${bundle}`,
    // A mismatch means this tab is running a cached bundle from an older build.
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
