import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode, PointerEvent as ReactPointerEvent } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DependencyEdge } from "../src/views/DependencyEdge";

vi.mock("@xyflow/react", () => ({
  BaseEdge: ({ path }: { path: string }) => (
    <div data-path={path} data-testid="base-edge" />
  ),
  EdgeToolbar: ({
    children,
    isVisible,
    onPointerDown,
    x,
    y,
  }: {
    children: ReactNode;
    isVisible: boolean;
    onPointerDown?: (event: ReactPointerEvent) => void;
    x: number;
    y: number;
  }) =>
    isVisible ? (
      <div
        data-testid="edge-toolbar"
        data-x={x}
        data-y={y}
        onPointerDown={onPointerDown}
      >
        {children}
      </div>
    ) : null,
}));

function edgeProps(
  overrides: Partial<Parameters<typeof DependencyEdge>[0]> = {},
) {
  return {
    id: "A-B",
    source: "A",
    target: "B",
    sourceX: 284,
    sourceY: 72,
    targetX: 406,
    targetY: 154,
    selected: true,
    data: {
      disabled: false,
      label: "A → B",
      onRemove: vi.fn(),
      readOnly: false,
      removeLabel: "Remove dependency A → B",
      route: {
        points: [
          { x: 284, y: 72 },
          { x: 340, y: 72 },
          { x: 340, y: 154 },
          { x: 406, y: 154 },
        ],
        sourcePortId: "A-B-source",
        sourcePortTop: 72,
        targetPortId: "A-B-target",
        targetPortTop: 74,
      },
    },
    ...overrides,
  } as Parameters<typeof DependencyEdge>[0];
}

function edgeData() {
  const data = edgeProps().data;
  if (!data) throw new Error("dependency edge data missing");
  return data;
}

describe("DependencyEdge", () => {
  afterEach(cleanup);

  it("draws the ELK route and exposes the selected dependency action", () => {
    const onRemove = vi.fn();
    render(
      <DependencyEdge {...edgeProps({ data: { ...edgeData(), onRemove } })} />,
    );

    expect(screen.getByTestId("base-edge")).toHaveAttribute(
      "data-path",
      "M284 72 L340 72 L340 154 L406 154",
    );
    expect(screen.getByTestId("edge-toolbar")).toHaveAttribute("data-x", "340");
    expect(screen.getByTestId("edge-toolbar")).toHaveAttribute("data-y", "118");
    const pointerEvent = new Event("pointerdown", { bubbles: true });
    const stopPropagation = vi.spyOn(pointerEvent, "stopPropagation");
    fireEvent(screen.getByTestId("edge-toolbar"), pointerEvent);
    expect(stopPropagation).toHaveBeenCalledOnce();
    fireEvent.click(
      screen.getByRole("button", { name: "Remove dependency A → B" }),
    );
    expect(onRemove).toHaveBeenCalledOnce();
  });

  it("falls back to endpoint coordinates and keeps read-only edges immutable", () => {
    render(
      <DependencyEdge
        {...edgeProps({
          data: {
            ...edgeData(),
            readOnly: true,
            route: undefined,
          },
        })}
      />,
    );

    expect(screen.getByTestId("base-edge")).toHaveAttribute(
      "data-path",
      "M284 72 L406 154",
    );
    expect(screen.getByText("A → B")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
