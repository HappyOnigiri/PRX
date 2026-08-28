import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MarkdownPreview } from "../src/views/MarkdownPreview";
import { readMarkdownDocument } from "../src/api";

vi.mock("../src/api", () => ({
  readMarkdownDocument: vi.fn(),
}));

describe("MarkdownPreview", () => {
  const writeText = vi.fn().mockResolvedValue(undefined);

  beforeEach(() => {
    vi.mocked(readMarkdownDocument).mockResolvedValue(
      "# Delivery plan\n\nShip the **API** safely.",
    );
    writeText.mockClear();
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
  });

  it("renders registered Markdown and copies its content or path", async () => {
    const onClose = vi.fn();
    render(
      <MarkdownPreview
        document={{
          id: "document-1",
          title: "Delivery plan",
          value: "docs/delivery.md",
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
});
