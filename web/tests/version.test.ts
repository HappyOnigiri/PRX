import { afterEach, describe, expect, it } from "vitest";
import { appVersion } from "../src/version";

describe("appVersion", () => {
  afterEach(() => {
    document.querySelector('meta[name="prx-version"]')?.remove();
  });

  it("uses the version injected by the Go server", () => {
    const meta = document.createElement("meta");
    meta.name = "prx-version";
    meta.content = "1.2.3";
    document.head.append(meta);

    expect(appVersion()).toBe("1.2.3");
  });

  it("uses the base release with a dev suffix without an injected version", () => {
    expect(appVersion()).toBe(import.meta.env.APP_VERSION);
    expect(appVersion()).toMatch(/-dev$/);
  });
});
