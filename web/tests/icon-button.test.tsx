import { render, screen } from "@testing-library/react";
import { Plus, RefreshCw, Trash2 } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { IconButton } from "../src/views/IconButton";

describe("IconButton", () => {
  it("provides a labeled standard button with a decorative icon", () => {
    render(
      <IconButton
        icon={Plus}
        label="Add task"
        variant="primary"
        onClick={vi.fn()}
      />,
    );

    const button = screen.getByRole("button", { name: "Add task" });
    expect(button).toHaveAttribute("type", "button");
    expect(button).toHaveAttribute("aria-label", "Add task");
    expect(button).toHaveAttribute("title", "Add task");
    expect(button).toHaveClass(
      "icon-button-standard",
      "icon-button-primary",
      "icon-button-with-label",
    );
    expect(screen.getByText("Add task")).toBeInTheDocument();

    const icon = button.querySelector("svg");
    expect(icon).toHaveAttribute("aria-hidden", "true");
    expect(icon).toHaveAttribute("focusable", "false");
    expect(icon).toHaveAttribute("width", "16");
    expect(icon).toHaveAttribute("height", "16");
  });

  it("supports compact icon-only disabled submit buttons", () => {
    render(
      <IconButton
        icon={Trash2}
        label="Delete task"
        variant="danger"
        size="compact"
        iconOnly
        type="submit"
        disabled
      />,
    );

    const button = screen.getByRole("button", { name: "Delete task" });
    expect(button).toHaveAttribute("type", "submit");
    expect(button).toHaveAttribute("aria-label", "Delete task");
    expect(button).toHaveAttribute("title", "Delete task");
    expect(button).toHaveClass(
      "icon-button-compact",
      "icon-button-danger",
      "icon-button-only",
    );
    expect(screen.queryByText("Delete task")).not.toBeInTheDocument();
    expect(button).toBeDisabled();

    const icon = button.querySelector("svg");
    expect(icon).toHaveAttribute("aria-hidden", "true");
    expect(icon).toHaveAttribute("focusable", "false");
    expect(icon).toHaveAttribute("width", "14");
    expect(icon).toHaveAttribute("height", "14");
  });

  it("marks a busy button so the icon animates and assistive tech hears it", () => {
    const { rerender } = render(
      <IconButton icon={RefreshCw} label="Sync GitHub" busy />,
    );

    const button = screen.getByRole("button", { name: "Sync GitHub" });
    expect(button).toHaveClass("icon-button-busy");
    expect(button).toHaveAttribute("aria-busy", "true");

    rerender(<IconButton icon={RefreshCw} label="Sync GitHub" />);
    expect(button).not.toHaveClass("icon-button-busy");
    expect(button).not.toHaveAttribute("aria-busy");
  });
});
