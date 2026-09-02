import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { formatBrowserDebugSection } from "../src/debug-text";
import { DebugProblemCode } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { DebugSettingsPanel } from "../src/views/DebugSettingsPanel";

interface DebugReportMock {
  data: { report: { problems: unknown[] }; text: string } | undefined;
  isError: boolean;
  error: Error | null;
  refetch: () => void;
}

const debugMocks = vi.hoisted(() => {
  const report: DebugReportMock = {
    data: undefined,
    isError: false,
    error: null,
    refetch: vi.fn(),
  };
  return {
    report,
    diagnostics: [{ name: "snapshot", state: "success, idle" }],
  };
});

vi.mock("../src/hooks", () => ({
  useDebugReport: () => debugMocks.report,
  useQueryDiagnostics: () => debugMocks.diagnostics,
}));

const reportText = "PRX diagnostic report\n\nproblems:\n  detected: none\n";

function reportBody() {
  return document.querySelector(".settings-debug-text")?.textContent;
}

function makeReport(problems: unknown[] = []) {
  return {
    data: {
      report: { problems, runtime: { generatedAt: "2026-09-03T04:05:06Z" } },
      text: reportText,
    },
    isError: false,
    error: null,
    refetch: vi.fn(),
  };
}

describe("DebugSettingsPanel", () => {
  beforeEach(async () => {
    await setDisplayLanguage("en");
    localStorage.clear();
    vi.useRealTimers();
    debugMocks.report = makeReport();
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("shows the report, its problems, and when it was taken", () => {
    debugMocks.report = makeReport([
      {
        code: DebugProblemCode.DATABASE_INTEGRITY_ERRORS,
        target: "~/prx/prx.db",
        evidence: "1 integrity errors",
        nextCommand: "prx validate",
      },
    ]);
    render(<DebugSettingsPanel />);

    expect(
      screen.getByText("Taken at 2026-09-03T04:05:06Z"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Stored dependency data failed validation"),
    ).toBeInTheDocument();
    expect(screen.getByText("~/prx/prx.db")).toBeInTheDocument();
    expect(screen.getByText("prx validate")).toBeInTheDocument();
    expect(reportBody()).toBe(reportText);
  });

  // A report from a newer server can carry a problem this bundle predates, and
  // the panel has to name it rather than render a translation key.
  it("falls back to a generic label for an unknown problem code", () => {
    debugMocks.report = makeReport([{ code: 9999, evidence: "something" }]);
    render(<DebugSettingsPanel />);
    expect(screen.getByText("Unrecognized problem")).toBeInTheDocument();
  });

  it("reports that nothing was detected", () => {
    render(<DebugSettingsPanel />);
    expect(screen.getByText("No problems were detected.")).toBeInTheDocument();
  });

  it("copies the server text with this browser's own state appended", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", {
      clipboard: { writeText },
      userAgent: "vitest",
    });
    render(<DebugSettingsPanel />);

    fireEvent.click(screen.getByRole("button", { name: "Copy report" }));
    await screen.findByText("Copied");
    const copied = String(writeText.mock.calls[0]?.[0]);
    expect(copied.startsWith(reportText)).toBe(true);
    expect(copied).toContain("browser:");
    expect(copied).toContain("snapshot: success, idle");
  });

  it("returns to the copy hint after confirming a copy", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", {
      clipboard: { writeText },
      userAgent: "vitest",
    });
    render(<DebugSettingsPanel />);

    fireEvent.click(screen.getByRole("button", { name: "Copy report" }));
    await screen.findByText("Copied");
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(
      screen.getByText(
        "The copied text is English and includes this browser's own state.",
      ),
    ).toBeInTheDocument();
    vi.useRealTimers();
  });

  it("reports a failed copy without losing the report", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("denied"));
    vi.stubGlobal("navigator", {
      clipboard: { writeText },
      userAgent: "vitest",
    });
    render(<DebugSettingsPanel />);

    fireEvent.click(screen.getByRole("button", { name: "Copy report" }));
    expect(await screen.findByText("Copy failed")).toBeInTheDocument();
    expect(reportBody()).toBe(reportText);
  });

  it("takes a new report on request", () => {
    render(<DebugSettingsPanel />);
    fireEvent.click(screen.getByRole("button", { name: "Take a new report" }));
    expect(debugMocks.report.refetch).toHaveBeenCalledOnce();
  });

  it("explains a failed request instead of showing an empty report", () => {
    debugMocks.report = {
      data: undefined,
      isError: true,
      error: new Error("the server is unavailable"),
      refetch: vi.fn(),
    };
    render(<DebugSettingsPanel />);
    expect(screen.getByRole("alert")).toHaveTextContent(
      "the server is unavailable",
    );
  });

  it("shows that the report is still being collected", () => {
    debugMocks.report = {
      data: undefined,
      isError: false,
      error: null,
      refetch: vi.fn(),
    };
    render(<DebugSettingsPanel />);
    expect(
      screen.getByText("Collecting the diagnostic report…"),
    ).toBeInTheDocument();
  });
});

describe("browser debug section", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    document.head.innerHTML = "";
    localStorage.clear();
  });

  it("follows the server layout and reports a version mismatch", () => {
    const meta = document.createElement("meta");
    meta.name = "prx-version";
    meta.content = "9.9.9";
    document.head.appendChild(meta);
    localStorage.setItem("prx.webui.settings", '{"language":"ja"}');

    const text = formatBrowserDebugSection([
      { name: "snapshot", state: "error: boom" },
    ]);
    expect(text.startsWith("\nbrowser:\n")).toBe(true);
    expect(text).toContain("  server_version: 9.9.9");
    expect(text).toContain("  version_match: no");
    expect(text).toContain('  local_storage: {"language":"ja"}');
    expect(text).toContain("  queries:\n    snapshot: error: boom\n");
  });

  it("reports unavailable storage and a missing server version", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    const text = formatBrowserDebugSection([]);
    expect(text).toContain("  server_version: unset");
    expect(text).toContain("  local_storage: unavailable");
  });
});
