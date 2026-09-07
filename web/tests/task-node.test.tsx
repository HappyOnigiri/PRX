import { act, fireEvent, render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";
import {
  DocumentKind,
  TaskBlockLabel,
  TaskDisplayState,
} from "../src/gen/prx/v1/prx_pb";
import { TaskNode, type TaskFlowNode } from "../src/views/TaskNode";

describe("TaskNode", () => {
  it("shows and copies the task ID and supports adding references", async () => {
    const onEdit = vi.fn();
    const onPreview = vi.fn();
    const onAddReference = vi.fn();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    const markdown = {
      id: "doc-md",
      kind: DocumentKind.LOCAL_FILE,
      title: "Rollout plan",
      locator: "docs/rollout.md",
      isImplementationPlan: true,
    };
    const props = {
      id: "T-42",
      data: {
        title: "Merge billing schema",
        assignee: "Carol",
        state: TaskDisplayState.IN_REVIEW,
        dormant: false,
        blocked: false,
        blockLabels: [TaskBlockLabel.CONFLICT],
        hasImplementationPlan: true,
        stale: true,
        syncError: "",
        pullRequest: {
          label: "acme/api #42",
          url: "https://github.com/acme/api/pull/42",
        },
        documents: [
          markdown,
          {
            id: "doc-url",
            kind: DocumentKind.URL,
            title: "Runbook",
            locator: "https://example.com/runbook",
            isImplementationPlan: false,
          },
        ],
        incomingPorts: [{ id: "edge-target", top: 44 }],
        outgoingPorts: [{ id: "edge-source", top: 92 }],
        readOnly: false,
        onEdit,
        onPreview,
        onAddReference,
      },
      selected: false,
      isConnectable: true,
      zIndex: 0,
      dragging: false,
      draggable: false,
      selectable: true,
      deletable: false,
      type: "task",
      positionAbsoluteX: 0,
      positionAbsoluteY: 0,
    } as NodeProps<TaskFlowNode>;
    const { container } = render(
      <ReactFlowProvider>
        <TaskNode {...props} />
      </ReactFlowProvider>,
    );
    const taskIdButton = screen.getByRole("button", { name: "Copy Task ID" });
    expect(taskIdButton).toHaveTextContent("T-42");
    expect(taskIdButton.querySelector("svg")).not.toBeInTheDocument();
    const taskActions = container.querySelector(".task-node-actions");
    expect(taskActions).toHaveClass("nodrag", "nopan");
    expect(taskActions?.firstElementChild).toContainElement(taskIdButton);
    await act(async () => {
      fireEvent.click(taskIdButton);
      await Promise.resolve();
    });
    expect(writeText).toHaveBeenCalledWith("T-42");
    expect(taskIdButton).toHaveClass("is-copied");
    expect(taskIdButton).toHaveAccessibleName("Copied");
    expect(screen.getByText("Merge billing schema")).toBeInTheDocument();
    expect(screen.getByText("Carol")).toBeInTheDocument();
    expect(screen.queryByText("READY")).not.toBeInTheDocument();
    expect(container.querySelector(".state-in-review")).toBeInTheDocument();
    // ステータスとブロックラベルは別の系統なので、ノードには両方が出る。
    expect(container.querySelector(".block-conflict")).toBeInTheDocument();
    // pull request 自身の異常は、その行の印として理由つきで出る。
    expect(
      screen.getByRole("button", {
        name: "Stale: this pull request may no longer match its state on GitHub.",
      }),
    ).toHaveClass("pr-flag", "is-stale");
    expect(container.querySelector(".pr-flag.is-sync-error")).toBeNull();
    const edgePorts = container.querySelectorAll(".task-edge-port");
    expect(edgePorts).toHaveLength(2);
    expect(edgePorts[0]).toHaveStyle({ top: "44px" });
    expect(edgePorts[1]).toHaveStyle({ top: "92px" });
    for (const port of edgePorts) {
      expect(port).toHaveClass("connectable");
      expect(port).toHaveClass("connectableend");
      expect(port).not.toHaveClass("connectablestart");
    }
    expect(
      screen.getByLabelText("Blocked task input (drop here)"),
    ).toHaveAttribute("title", "Blocked task input (drop here)");
    expect(
      screen.getByLabelText("Blocker output (drag from here)"),
    ).toHaveAttribute("title", "Blocker output (drag from here)");
    expect(screen.getByRole("link", { name: /acme\/api #42/ })).toHaveAttribute(
      "target",
      "_blank",
    );
    expect(screen.getByRole("link", { name: /Runbook/ })).toHaveAttribute(
      "target",
      "_blank",
    );
    fireEvent.click(screen.getByRole("button", { name: /Rollout plan/ }));
    expect(onPreview).toHaveBeenCalledWith(markdown);
    fireEvent.click(
      screen.getByRole("button", { name: "Edit Merge billing schema" }),
    );
    expect(onEdit).toHaveBeenCalledOnce();
    const addReference = screen.getByRole("button", {
      name: "Add reference to Merge billing schema",
    });
    expect(addReference).toHaveAttribute(
      "title",
      "Add reference to Merge billing schema",
    );
    fireEvent.click(addReference);
    expect(onAddReference).toHaveBeenCalledWith(addReference);
  });

  it("labels the inspector action as view-only for archived tasks", () => {
    const onEdit = vi.fn();
    const props = {
      id: "task",
      data: {
        title: "Archived task",
        assignee: "",
        state: TaskDisplayState.NOT_STARTED,
        dormant: false,
        blocked: false,
        blockLabels: [],
        hasImplementationPlan: false,
        stale: false,
        syncError: "",
        pullRequest: undefined,
        documents: [],
        readOnly: true,
        onEdit,
        onPreview: vi.fn(),
      },
      selected: false,
      isConnectable: false,
      zIndex: 0,
      dragging: false,
      draggable: false,
      selectable: true,
      deletable: false,
      type: "task",
      positionAbsoluteX: 0,
      positionAbsoluteY: 0,
    } as NodeProps<TaskFlowNode>;
    const { container } = render(
      <ReactFlowProvider>
        <TaskNode {...props} />
      </ReactFlowProvider>,
    );
    expect(screen.queryByText("Unassigned")).not.toBeInTheDocument();
    expect(container.querySelector(".node-assignee")).not.toBeInTheDocument();
    expect(
      container.querySelector(".node-add-reference"),
    ).not.toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "View Archived task details" }),
    );
    expect(onEdit).toHaveBeenCalledOnce();
  });

  it("stands in for hidden completed dependencies on each side", () => {
    const props = {
      id: "task",
      data: {
        title: "Ship API",
        assignee: "",
        state: TaskDisplayState.NOT_STARTED,
        dormant: false,
        blocked: false,
        blockLabels: [],
        hasImplementationPlan: false,
        ready: false,
        stale: false,
        syncError: "",
        pullRequest: undefined,
        documents: [],
        hiddenDependencies: {
          blockers: ["Migrate schema", "Design schema"],
          blocked: ["Announce"],
        },
        readOnly: true,
        onEdit: vi.fn(),
        onPreview: vi.fn(),
      },
      selected: false,
      isConnectable: false,
      zIndex: 0,
      dragging: false,
      draggable: false,
      selectable: true,
      deletable: false,
      type: "task",
      positionAbsoluteX: 0,
      positionAbsoluteY: 0,
    } as NodeProps<TaskFlowNode>;
    const { container } = render(
      <ReactFlowProvider>
        <TaskNode {...props} />
      </ReactFlowProvider>,
    );
    const blockers = screen.getByRole("img", {
      name: "Hidden completed blockers: Migrate schema, Design schema",
    });
    expect(blockers).toHaveClass("node-hidden-dependency-in");
    expect(blockers).toHaveAttribute(
      "title",
      "Hidden completed blockers: Migrate schema, Design schema",
    );
    expect(
      screen.getByRole("img", {
        name: "Hidden completed dependents: Announce",
      }),
    ).toHaveClass("node-hidden-dependency-out");
    expect(container.querySelectorAll(".node-hidden-dependency")).toHaveLength(
      2,
    );
  });

  it("leaves out the stub when nothing is hidden behind the task", () => {
    const props = {
      id: "task",
      data: {
        title: "Ship API",
        assignee: "",
        state: TaskDisplayState.NOT_STARTED,
        dormant: false,
        blocked: false,
        blockLabels: [],
        hasImplementationPlan: false,
        ready: false,
        stale: false,
        syncError: "",
        pullRequest: undefined,
        documents: [],
        hiddenDependencies: { blockers: [], blocked: [] },
        readOnly: true,
        onEdit: vi.fn(),
        onPreview: vi.fn(),
      },
      selected: false,
      isConnectable: false,
      zIndex: 0,
      dragging: false,
      draggable: false,
      selectable: true,
      deletable: false,
      type: "task",
      positionAbsoluteX: 0,
      positionAbsoluteY: 0,
    } as NodeProps<TaskFlowNode>;
    const { container } = render(
      <ReactFlowProvider>
        <TaskNode {...props} />
      </ReactFlowProvider>,
    );
    expect(
      container.querySelector(".node-hidden-dependency"),
    ).not.toBeInTheDocument();
  });

  it("sinks the surface of a task that waits on a blocker", () => {
    const props = {
      id: "task",
      data: {
        title: "Ship API",
        assignee: "",
        state: TaskDisplayState.IMPLEMENTED,
        dormant: true,
        blocked: true,
        blockLabels: [TaskBlockLabel.DEPENDENCY_UNRESOLVED],
        hasImplementationPlan: false,
        stale: false,
        syncError: "",
        pullRequest: undefined,
        documents: [],
        readOnly: true,
        onEdit: vi.fn(),
        onPreview: vi.fn(),
      },
      selected: false,
      isConnectable: false,
      zIndex: 0,
      dragging: false,
      draggable: false,
      selectable: true,
      deletable: false,
      type: "task",
      positionAbsoluteX: 0,
      positionAbsoluteY: 0,
    } as NodeProps<TaskFlowNode>;
    const { container } = render(
      <ReactFlowProvider>
        <TaskNode {...props} />
      </ReactFlowProvider>,
    );
    // 決着済みと違って作業は残っているので、沈んだ面に枠を残す印も付く。
    expect(container.querySelector(".task-node")).toHaveClass(
      "is-dormant",
      "is-dependency-blocked",
    );
  });
});
