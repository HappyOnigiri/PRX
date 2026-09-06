import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setDisplayLanguage } from "../src/i18n";
import { TaskPromptCopyButton } from "../src/views/TaskPromptCopyButton";

const promptMocks = vi.hoisted(() => ({ getTaskPrompt: vi.fn() }));

vi.mock("../src/api", () => ({ getTaskPrompt: promptMocks.getTaskPrompt }));

function renderButton(hasImplementationPlan: boolean) {
  return render(
    <TaskPromptCopyButton
      taskId="T-1"
      hasImplementationPlan={hasImplementationPlan}
    />,
  );
}

function stubClipboard(writeText: ReturnType<typeof vi.fn>) {
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
}

describe("TaskPromptCopyButton", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    await setDisplayLanguage("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("copies the design prompt of a task that has no plan", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);

    fireEvent.click(screen.getByRole("button", { name: "Copy design prompt" }));

    await waitFor(() => {
      expect(screen.getByText("Prompt copied.")).toBeInTheDocument();
    });
    expect(promptMocks.getTaskPrompt).toHaveBeenCalledWith("T-1");
    expect(writeText).toHaveBeenCalledWith("Design T-1");
  });

  // The label follows the snapshot, but the copied text is whatever the server
  // renders, so the two are asserted independently.
  it("offers the implementation prompt once the task has a plan", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Build T-1" });
    renderButton(true);

    fireEvent.click(
      screen.getByRole("button", { name: "Copy implementation prompt" }),
    );

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith("Build T-1");
    });
  });

  it("reports a server failure without writing to the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    promptMocks.getTaskPrompt.mockRejectedValue(
      new Error('task "T-1" was not found'),
    );
    renderButton(false);

    fireEvent.click(screen.getByRole("button", { name: "Copy design prompt" }));

    await waitFor(() => {
      expect(screen.getByText('task "T-1" was not found')).toBeInTheDocument();
    });
    expect(writeText).not.toHaveBeenCalled();
  });

  // A clipboard rejection carries a browser-internal message, so the reader is
  // told what happened in their own language instead.
  it("reports a clipboard failure in the display language", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("clipboard blocked")));
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);

    fireEvent.click(screen.getByRole("button", { name: "Copy design prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText("The prompt could not be copied."),
      ).toBeInTheDocument();
    });
    expect(screen.queryByText("clipboard blocked")).not.toBeInTheDocument();
  });

  // Outside a secure context the whole clipboard API is missing, which would
  // otherwise surface as a raw TypeError.
  it("reports a missing clipboard API", async () => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: undefined,
    });
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);

    fireEvent.click(screen.getByRole("button", { name: "Copy design prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText("The prompt could not be copied."),
      ).toBeInTheDocument();
    });
  });

  it("clears the outcome so it does not describe an earlier click", async () => {
    vi.useFakeTimers();
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);

    fireEvent.click(screen.getByRole("button", { name: "Copy design prompt" }));
    await vi.waitFor(() => {
      expect(screen.getByText("Prompt copied.")).toBeInTheDocument();
    });

    act(() => {
      vi.advanceTimersByTime(1600);
    });
    expect(screen.queryByText("Prompt copied.")).not.toBeInTheDocument();
    vi.useRealTimers();
  });
});
