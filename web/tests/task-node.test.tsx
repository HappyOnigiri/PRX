import { fireEvent, render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";
import { DocumentKind, TaskDisplayState } from "../src/gen/prx/v1/prx_pb";
import { TaskNode, type TaskFlowNode } from "../src/views/TaskNode";

describe("TaskNode", () => {
  it("shows an assignee and reflects ready and stale state without badges", () => {
    const onEdit = vi.fn();
    const onPreview = vi.fn();
    const markdown = {
      id: "doc-md",
      kind: DocumentKind.LOCAL_FILE,
      title: "Rollout plan",
      locator: "docs/rollout.md",
      isImplementationPlan: true,
    };
    const props = {
      id: "task",
      data: {
        title: "Merge billing schema",
        assignee: "Ren",
        state: TaskDisplayState.REVIEW_WAITING,
        ready: true,
        stale: true,
        syncError: false,
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
    expect(screen.getByText("Merge billing schema")).toBeInTheDocument();
    expect(screen.getByText("Ren")).toBeInTheDocument();
    expect(screen.queryByText("READY")).not.toBeInTheDocument();
    expect(container.querySelector(".is-ready")).toBeInTheDocument();
    expect(container.querySelector(".is-stale")).toBeInTheDocument();
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
  });

  it("labels the inspector action as view-only for archived tasks", () => {
    const onEdit = vi.fn();
    const props = {
      id: "task",
      data: {
        title: "Archived task",
        assignee: "",
        state: TaskDisplayState.NOT_STARTED,
        ready: false,
        stale: false,
        syncError: false,
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
    expect(container.querySelector("footer")).not.toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "View Archived task details" }),
    );
    expect(onEdit).toHaveBeenCalledOnce();
  });
});
