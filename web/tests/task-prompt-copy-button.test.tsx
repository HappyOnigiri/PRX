import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type Task } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { TaskPromptCopyButton } from "../src/views/TaskPromptCopyButton";

const promptMocks = vi.hoisted(() => ({ getTaskPrompt: vi.fn() }));

vi.mock("../src/api", () => ({ getTaskPrompt: promptMocks.getTaskPrompt }));

function makeTask(hasImplementationPlan: boolean): Task {
  return { id: "T-1", hasImplementationPlan } as Task;
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
    render(<TaskPromptCopyButton task={makeTask(false)} />);

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
    render(<TaskPromptCopyButton task={makeTask(true)} />);

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
    render(<TaskPromptCopyButton task={makeTask(false)} />);

    fireEvent.click(screen.getByRole("button", { name: "Copy design prompt" }));

    await waitFor(() => {
      expect(screen.getByText('task "T-1" was not found')).toBeInTheDocument();
    });
    expect(writeText).not.toHaveBeenCalled();
  });

  it("reports a clipboard failure", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("clipboard blocked")));
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    render(<TaskPromptCopyButton task={makeTask(false)} />);

    fireEvent.click(screen.getByRole("button", { name: "Copy design prompt" }));

    await waitFor(() => {
      expect(screen.getByText("clipboard blocked")).toBeInTheDocument();
    });
  });
});
