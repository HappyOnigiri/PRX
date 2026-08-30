import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readDocumentContent } from "../src/api";
import { MarkdownPreview } from "../src/views/MarkdownPreview";

vi.mock("../src/api", () => ({
  readDocumentContent: vi.fn(),
}));

describe("MarkdownPreview", () => {
  const writeText = vi.fn().mockResolvedValue(undefined);

  beforeEach(() => {
    vi.mocked(readDocumentContent).mockResolvedValue(
      "# Delivery plan\n\nShip the **API** safely.",
    );
    writeText.mockClear();
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("renders registered Markdown and copies its content or path", async () => {
    const onClose = vi.fn();
    render(
      <MarkdownPreview
        document={{
          id: "document-1",
          title: "Delivery plan",
          locator: "docs/delivery.md",
        }}
        onClose={onClose}
      />,
    );

    expect(
      await screen.findByRole("heading", { name: "Delivery plan" }),
    ).toBeInTheDocument();
    expect(screen.getByText("API")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Copy full text" }));
    expect(writeText).toHaveBeenCalledWith(
      "# Delivery plan\n\nShip the **API** safely.",
    );
    fireEvent.click(screen.getByRole("button", { name: "Copy file path" }));
    expect(writeText).toHaveBeenCalledWith("docs/delivery.md");
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("shows a read error, retries, and opens Markdown links externally", async () => {
    vi.mocked(readDocumentContent)
      .mockRejectedValueOnce(new Error("file disappeared"))
      .mockResolvedValueOnce("[Runbook](https://example.com/runbook)");
    const { unmount } = render(
      <MarkdownPreview
        document={{
          id: "document-2",
          title: "Runbook",
          locator: "docs/runbook.md",
        }}
        onClose={vi.fn()}
      />,
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "file disappeared",
    );
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    const link = await screen.findByRole("link", { name: "Runbook" });
    expect(link).toHaveAttribute("target", "_blank");
    expect(readDocumentContent).toHaveBeenCalledWith("document-2");
    unmount();
  });

  it("reports clipboard failures and clears the status after its timer", async () => {
    writeText.mockRejectedValueOnce(new Error("clipboard denied"));
    render(
      <MarkdownPreview
        document={{
          id: "document-3",
          title: "Notes",
          locator: "docs/notes.md",
        }}
        onClose={vi.fn()}
      />,
    );
    await screen.findByRole("heading", { name: "Notes" });

    vi.useFakeTimers();
    fireEvent.click(screen.getByRole("button", { name: "Copy file path" }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(
      screen.getByText(
        "Could not copy. Check the browser's clipboard permission.",
      ),
    ).toBeInTheDocument();
    act(() => {
      vi.advanceTimersByTime(1600);
    });
    expect(
      screen.queryByText(
        "Could not copy. Check the browser's clipboard permission.",
      ),
    ).not.toBeInTheDocument();
  });
});
