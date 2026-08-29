import { fireEvent, render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";
import { DocumentKind, TaskDisplayState } from "../src/gen/prx/v1/prx_pb";
import { TaskNode, type TaskFlowNode } from "../src/views/TaskNode";

describe("TaskNode", () => {
  it("exposes ready and stale state without relying on color", () => {
    const onEdit = vi.fn();
    const onPreview = vi.fn();
    const markdown = {
      id: "doc-md",
      kind: DocumentKind.MARKDOWN_PATH,
      title: "Rollout plan",
      value: "docs/rollout.md",
    };
    const props = {
      id: "task",
      data: {
        title: "Merge billing schema",
        assignee: "Ren",
        state: TaskDisplayState.REVIEW_WAITING,
        ready: true,
        stale: true,
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
            value: "https://example.com/runbook",
          },
        ],
        onEdit,
        onPreview,
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
    expect(screen.getByText("Merge billing schema")).toBeInTheDocument();
    expect(screen.getByText("READY")).toBeInTheDocument();
    expect(container.querySelector(".is-stale")).toBeInTheDocument();
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
});
