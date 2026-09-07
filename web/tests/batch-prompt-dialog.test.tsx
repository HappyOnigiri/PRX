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
        // Designed, and waiting on a task the batch can carry, so it can be
        // handed over behind that task.
        makeTask({
          id: "task-4",
          title: "Ship the change",
          ...designed,
          ready: false,
          pendingBlockerTaskIds: ["task-1"],
        }),
        // Waiting on a task without a plan, which no batch can hand over, so
        // selecting this one could never become possible.
        makeTask({
          id: "task-5",
          title: "Archive the ledger",
          ...designed,
          ready: false,
          pendingBlockerTaskIds: ["task-3"],
        }),
        // Waiting on two tasks the batch could carry, which leaves its pull
        // request no single base to stack on.
        makeTask({
          id: "task-6",
          title: "Close the books",
          ...designed,
          ready: false,
          pendingBlockerTaskIds: ["task-1", "task-2"],
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

function includeBlocked() {
  return screen.getByLabelText("Include dependent tasks");
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
    expect(includeBlocked()).not.toBeChecked();
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

  // A blocked task can be handed over as long as the work it waits for travels
  // with it, so it is offered only once the reader asks for the dependent tasks.
  it("offers a blocked task behind the work it waits for", () => {
    renderDialog();

    fireEvent.click(includeBlocked());

    const blocked = taskRow("Ship the change");
    expect(blocked).toBeDisabled();
    expect(blocked).toHaveTextContent("after task-1");
    expect(screen.getByText("0 of 3 selected")).toBeInTheDocument();
    // Its blocker has no plan, so no batch could ever carry this one.
    expect(
      screen.queryByRole("button", { name: /Archive the ledger/ }),
    ).not.toBeInTheDocument();
    // Two blockers leave no single pull request to stack on, so the task is
    // not offered even though the batch could carry both of them.
    expect(
      screen.queryByRole("button", { name: /Close the books/ }),
    ).not.toBeInTheDocument();

    fireEvent.click(taskRow("Build API"));
    expect(taskRow("Ship the change")).toBeEnabled();
    fireEvent.click(taskRow("Ship the change"));
    expect(taskRow("Ship the change")).toHaveAttribute("aria-pressed", "true");
  });

  // Dropping a blocker strands whatever was stacked on it, so the selection
  // cannot be left describing work with no base.
  it("clears a dependent task when its blocker leaves the selection", () => {
    renderDialog();

    fireEvent.click(includeBlocked());
    fireEvent.click(taskRow("Build API"));
    fireEvent.click(taskRow("Ship the change"));
    expect(screen.getByText("2 of 3 selected")).toBeInTheDocument();

    fireEvent.click(taskRow("Build API"));
    expect(screen.getByText("0 of 3 selected")).toBeInTheDocument();
    expect(taskRow("Ship the change")).toBeDisabled();
  });

  it("drops the blocked tasks again when the option is turned off", () => {
    renderDialog();

    fireEvent.click(includeBlocked());
    fireEvent.click(screen.getByRole("button", { name: "Select all" }));
    expect(screen.getByText("3 of 3 selected")).toBeInTheDocument();

    fireEvent.click(includeBlocked());
    expect(screen.getByText("2 of 2 selected")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Ship the change/ }),
    ).not.toBeInTheDocument();
  });

  // The batch names a blocker before the work stacked on it, so the receiving
  // agent reads the list in an order it can act on.
  it("copies the tasks with each blocker ahead of what waits for it", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    batchMocks.getBatchPrompt.mockResolvedValue({
      featureId: "feature-1",
      taskIds: ["task-1", "task-2", "task-4"],
      prompt: "Batch feature-1",
    });
    renderDialog();

    fireEvent.click(includeBlocked());
    fireEvent.click(screen.getByRole("button", { name: "Select all" }));
    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText("Copied a prompt for 3 tasks."),
      ).toBeInTheDocument();
    });
    expect(batchMocks.getBatchPrompt).toHaveBeenCalledWith("feature-1", [
      "task-1",
      "task-2",
      "task-4",
    ]);
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
