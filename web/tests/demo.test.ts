import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
  isDemoMode,
  readDemoNoticeDismissed,
  writeDemoNoticeDismissed,
} from "../src/demo";

function injectMeta(name: string, content: string) {
  const meta = document.createElement("meta");
  meta.name = name;
  meta.content = content;
  document.head.append(meta);
}

describe("isDemoMode", () => {
  afterEach(() => {
    document.querySelector('meta[name="prx-demo"]')?.remove();
  });

  it("reads demo mode injected by the Go server", () => {
    injectMeta("prx-demo", "true");
    expect(isDemoMode()).toBe(true);
  });

  it("defaults to normal mode without injected metadata", () => {
    expect(isDemoMode()).toBe(false);
  });

  it("ignores the placeholder left in the built HTML", () => {
    injectMeta("prx-demo", "__PRX_DEMO__");
    expect(isDemoMode()).toBe(false);
  });
});

describe("demo notice dismissal", () => {
  beforeEach(() => {
    localStorage.clear();
  });
  afterEach(() => {
    document.querySelector('meta[name="prx-demo-session"]')?.remove();
  });

  it("remembers the dismissal for the session that is being served", () => {
    injectMeta("prx-demo-session", "session-1");
    expect(readDemoNoticeDismissed()).toBe(false);
    writeDemoNoticeDismissed();
    expect(readDemoNoticeDismissed()).toBe(true);
  });

  it("forgets the dismissal once another session is served", () => {
    injectMeta("prx-demo-session", "session-1");
    writeDemoNoticeDismissed();

    document.querySelector('meta[name="prx-demo-session"]')?.remove();
    injectMeta("prx-demo-session", "session-2");
    expect(readDemoNoticeDismissed()).toBe(false);
  });

  it("keeps nothing when no session is injected", () => {
    writeDemoNoticeDismissed();
    expect(localStorage.length).toBe(0);
    expect(readDemoNoticeDismissed()).toBe(false);
  });
});
