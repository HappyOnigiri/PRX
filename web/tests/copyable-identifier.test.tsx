import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CopyableIdentifier } from "../src/views/CopyableIdentifier";

describe("CopyableIdentifier", () => {
  const writeText = vi.fn().mockResolvedValue(undefined);

  beforeEach(() => {
    writeText.mockReset();
    writeText.mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("displays and copies the identifier", async () => {
    vi.useFakeTimers();
    render(<CopyableIdentifier label="Task ID" value="task-123" />);

    expect(screen.getByText("task-123")).toBeInTheDocument();
    const copyButton = screen.getByRole("button", { name: "Copy Task ID" });
    expect(copyButton.querySelector('[data-icon="copy"]')).toBeInTheDocument();
    fireEvent.click(copyButton);
    await act(async () => {
      await Promise.resolve();
    });

    expect(writeText).toHaveBeenCalledWith("task-123");
    expect(copyButton).toHaveClass("is-copied");
    expect(copyButton.querySelector('[data-icon="check"]')).toBeInTheDocument();
    expect(copyButton).toHaveAccessibleName("Copied");
    expect(screen.queryByText("Copied")).not.toBeInTheDocument();
    act(() => {
      vi.advanceTimersByTime(1600);
    });
    expect(copyButton).not.toHaveClass("is-copied");
    expect(copyButton.querySelector('[data-icon="copy"]')).toBeInTheDocument();
    expect(copyButton).toHaveAccessibleName("Copy Task ID");
  });

  it("reports clipboard failures and clears the status", async () => {
    vi.useFakeTimers();
    writeText.mockRejectedValueOnce(new Error("clipboard denied"));
    render(<CopyableIdentifier label="Feature ID" value="feature-123" />);

    fireEvent.click(screen.getByRole("button", { name: "Copy Feature ID" }));
    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByText("Copy failed")).toBeInTheDocument();
    act(() => {
      vi.advanceTimersByTime(1600);
    });
    expect(screen.queryByText("Copy failed")).not.toBeInTheDocument();
  });

  it("uses the identifier itself as the copy control", async () => {
    render(
      <CopyableIdentifier label="Task ID" value="T-42" valueOnly={true} />,
    );

    expect(screen.queryByText("Task ID")).not.toBeInTheDocument();
    const copyButton = screen.getByRole("button", { name: "Copy Task ID" });
    expect(copyButton).toHaveTextContent("T-42");
    expect(copyButton.querySelector("svg")).not.toBeInTheDocument();

    fireEvent.click(copyButton);
    await act(async () => {
      await Promise.resolve();
    });

    expect(writeText).toHaveBeenCalledWith("T-42");
    expect(copyButton).toHaveClass("is-copied");
    expect(copyButton).toHaveAccessibleName("Copied");
  });
});
