const demoPlaceholder = "__PRX_DEMO__";

export function isDemoMode(): boolean {
  const injected = document
    .querySelector<HTMLMetaElement>('meta[name="prx-demo"]')
    ?.getAttribute("content");
  return injected !== demoPlaceholder && injected === "true";
}
