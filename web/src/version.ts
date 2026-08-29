const versionPlaceholder = "__PRX_VERSION__";

export function appVersion() {
  const injected = document
    .querySelector<HTMLMetaElement>('meta[name="prx-version"]')
    ?.content.trim();
  return injected && injected !== versionPlaceholder
    ? injected
    : import.meta.env.APP_VERSION;
}
