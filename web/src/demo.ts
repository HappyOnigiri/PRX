const demoPlaceholder = "__PRX_DEMO__";
const demoSessionPlaceholder = "__PRX_DEMO_SESSION__";
const dismissedSessionKey = "prx.webui.demoNoticeDismissedSession";

function readInjected(name: string, placeholder: string): string {
  const injected = document
    .querySelector<HTMLMetaElement>(`meta[name="${name}"]`)
    ?.getAttribute("content");
  if (!injected || injected === placeholder) return "";
  return injected;
}

export function isDemoMode(): boolean {
  return readInjected("prx-demo", demoPlaceholder) === "true";
}

// 閉じた警告は demo を配信しているプロセスの ID に結び付ける。読み込み直しても
// 同じ ID なので戻らず、サーバを起動し直すと ID が変わって戻る。
export function readDemoNoticeDismissed(): boolean {
  const session = readInjected("prx-demo-session", demoSessionPlaceholder);
  if (!session) return false;
  try {
    return localStorage.getItem(dismissedSessionKey) === session;
  } catch {
    return false;
  }
}

export function writeDemoNoticeDismissed() {
  const session = readInjected("prx-demo-session", demoSessionPlaceholder);
  if (!session) return;
  try {
    localStorage.setItem(dismissedSessionKey, session);
  } catch {
    // ストレージが使えなくても、この描画の間は警告が消える。
  }
}
