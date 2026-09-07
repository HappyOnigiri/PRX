import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TaskDisplayState } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { BatchPromptDialog } from "../src/views/BatchPromptDialog";
import { makeTask } from "./factories";

const batchMocks = vi.hoisted(() => ({ getBatchPrompt: vi.fn() }));

vi.mock("../src/api", () => ({ getBatchPrompt: batchMocks.getBatchPrompt }));

const designed = {
  displayState: TaskDisplayState.DESIGNED,
  hasImplementationPlan: true,
  ready: true,
};

function renderDialog(onClose = vi.fn()) {
  return render(
    <BatchPromptDialog
      featureId="feature-1"
      tasks={[
        makeTask({ id: "task-1", title: "Build API", ...designed }),
        makeTask({ id: "task-2", title: "Bill the order", ...designed }),
        // Not started: nothing has been designed for it yet.
        makeTask({ id: "task-3", title: "Draft the schema" }),
        // Designed but waiting on a blocker, so it cannot be started now.
        makeTask({
          id: "task-4",
          title: "Ship the change",
          ...designed,
          ready: false,
        }),
      ]}
      onClose={onClose}
    />,
  );
}

// The row is the control now, so a task is addressed by the button carrying its
// title and identifier rather than by a checkbox label.
function taskRow(title: string) {
  return screen.getByRole("button", { name: new RegExp(title) });
}

function stubClipboard(writeText: ReturnType<typeof vi.fn>) {
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
}

describe("BatchPromptDialog", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    await setDisplayLanguage("en");
  });

  afterEach(() => {
    cleanup();
  });

  // Readiness and the plan are both server derivations, so the dialog offers a
  // task only when the server says the work can start now.
  it("offers the designed tasks whose blockers are clear", () => {
    renderDialog();

    expect(taskRow("Build API")).toBeInTheDocument();
    expect(taskRow("Bill the order")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Draft the schema/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Ship the change/ }),
    ).not.toBeInTheDocument();
    expect(screen.getByText("0 of 2 selected")).toBeInTheDocument();
    // Nothing is selected yet, so there is no prompt to copy.
    expect(screen.getByRole("button", { name: "Copy prompt" })).toBeDisabled();
  });

  it("copies one prompt for the tasks selected in the listed order", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    batchMocks.getBatchPrompt.mockResolvedValue({
      featureId: "feature-1",
      taskIds: ["task-1", "task-2"],
      prompt: "Batch feature-1",
    });
    renderDialog();

    // Clicking the later task first must not reorder the request: the reader
    // hands over the batch in the order they saw it.
    fireEvent.click(taskRow("Bill the order"));
    fireEvent.click(taskRow("Build API"));
    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText("Copied a prompt for 2 tasks."),
      ).toBeInTheDocument();
    });
    expect(batchMocks.getBatchPrompt).toHaveBeenCalledWith("feature-1", [
      "task-1",
      "task-2",
    ]);
    expect(writeText).toHaveBeenCalledWith("Batch feature-1");
  });

  it("selects and clears every offered task with one control", () => {
    renderDialog();

    fireEvent.click(screen.getByRole("button", { name: "Select all" }));
    expect(screen.getByText("2 of 2 selected")).toBeInTheDocument();
    expect(taskRow("Build API")).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(screen.getByRole("button", { name: "Clear selection" }));
    expect(screen.getByText("0 of 2 selected")).toBeInTheDocument();
    expect(taskRow("Build API")).toHaveAttribute("aria-pressed", "false");
  });

  // Selecting a row is a pointer gesture with no checkbox behind it, so the
  // row has to be a control a keyboard reaches and activates on its own.
  it("offers each row as a focusable toggle", () => {
    renderDialog();

    const row = taskRow("Build API");
    expect(row).toHaveAttribute("type", "button");
    expect(row).toHaveAttribute("aria-pressed", "false");
    row.focus();
    expect(row).toHaveFocus();
  });

  // The server names the task or the template at fault, so its message reaches
  // the reader unchanged.
  it("reports a server failure without writing to the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    batchMocks.getBatchPrompt.mockRejectedValue(
      new Error('task "task-2" was not found'),
    );
    renderDialog();

    fireEvent.click(screen.getByRole("button", { name: "Select all" }));
    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText('task "task-2" was not found'),
      ).toBeInTheDocument();
    });
    expect(writeText).not.toHaveBeenCalled();
  });

  // A clipboard rejection carries a browser-internal message, and outside a
  // secure context the API is missing altogether.
  it("reports a clipboard failure in the display language", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("clipboard blocked")));
    batchMocks.getBatchPrompt.mockResolvedValue({
      featureId: "feature-1",
      taskIds: ["task-1"],
      prompt: "Batch feature-1",
    });
    renderDialog();

    fireEvent.click(taskRow("Build API"));
    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText("The prompt could not be copied."),
      ).toBeInTheDocument();
    });
    expect(screen.queryByText("clipboard blocked")).not.toBeInTheDocument();
  });

  it("says when the feature has nothing ready to implement", () => {
    const onClose = vi.fn();
    render(
      <BatchPromptDialog
        featureId="feature-1"
        tasks={[makeTask({ id: "task-3", title: "Draft the schema" })]}
        onClose={onClose}
      />,
    );

    expect(
      screen.getByText("No task in this feature is ready to implement."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy prompt" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalled();
  });
});
