import { render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it } from "vitest";
import { TaskNode, type TaskFlowNode } from "../src/views/TaskNode";

describe("TaskNode", () => {
  it("exposes ready and stale state without relying on color", () => {
    const props = {
      id: "task",
      data: {
        title: "Merge billing schema",
        repository: "acme/api #42",
        assignee: "Ren",
        state: "review_waiting",
        ready: true,
        stale: true,
      },
      selected: false,
      isConnectable: false,
      zIndex: 0,
      dragging: false,
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
  });
});
