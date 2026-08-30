import { afterEach, describe, expect, it } from "vitest";
import { isDemoMode } from "../src/demo";

describe("isDemoMode", () => {
  afterEach(() => {
    document.querySelector('meta[name="prx-demo"]')?.remove();
  });

  it("reads demo mode injected by the Go server", () => {
    const meta = document.createElement("meta");
    meta.name = "prx-demo";
    meta.content = "true";
    document.head.append(meta);
    expect(isDemoMode()).toBe(true);
  });

  it("defaults to normal mode without injected metadata", () => {
    expect(isDemoMode()).toBe(false);
  });
});
