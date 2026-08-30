import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DocumentKind } from "../src/gen/prx/v1/prx_pb";
import { AddDocumentDialog } from "../src/views/AddDocumentDialog";

const dialogMocks = vi.hoisted(() => ({
  addDocument: vi.fn(),
  selectLocalFile: vi.fn(),
}));

vi.mock("../src/api", () => ({
  mutations: { addDocument: dialogMocks.addDocument },
  selectLocalFile: dialogMocks.selectLocalFile,
}));

function renderDialog(props: { featureId?: string; taskId?: string } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  function Wrapper({ children }: PropsWithChildren) {
    return (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
  }
  const trigger = document.createElement("button");
  document.body.append(trigger);
  const onClose = vi.fn();
  const parent = props.taskId
    ? { taskId: props.taskId }
    : { featureId: props.featureId ?? "feature-1" };
  render(
    <AddDocumentDialog {...parent} trigger={trigger} onClose={onClose} />,
    { wrapper: Wrapper },
  );
  return { onClose, trigger };
}

describe("AddDocumentDialog", () => {
  beforeEach(() => {
    dialogMocks.addDocument.mockReset().mockResolvedValue({});
    dialogMocks.selectLocalFile
      .mockReset()
      .mockResolvedValue({ path: "/tmp/plan.md", canceled: false });
  });

  afterEach(() => {
    cleanup();
  });

  it("keeps tab inputs and adds task Markdown as an implementation plan", async () => {
    const { onClose } = renderDialog({ taskId: "task-1" });
    const urlTab = screen.getByRole("tab", { name: "URL" });
    await waitFor(() => {
      expect(urlTab).toHaveFocus();
    });
    fireEvent.change(screen.getByLabelText("Document URL"), {
      target: { value: "https://example.com/spec" },
    });

    fireEvent.keyDown(urlTab, { key: "ArrowRight" });
    const localTab = screen.getByRole("tab", { name: "Local file" });
    expect(localTab).toHaveFocus();
    fireEvent.click(screen.getByRole("button", { name: "Choose file…" }));
    await waitFor(() => {
      expect(screen.getByLabelText("File path")).toHaveValue("/tmp/plan.md");
    });

    fireEvent.keyDown(localTab, { key: "End" });
    const markdownTab = screen.getByRole("tab", { name: "Markdown" });
    expect(markdownTab).toHaveFocus();
    fireEvent.change(screen.getByLabelText("Markdown content"), {
      target: { value: "# Delivery\n\nShip safely." },
    });
    fireEvent.click(
      screen.getByLabelText("Use as this task's implementation plan"),
    );
    fireEvent.change(screen.getByLabelText("Reference title (optional)"), {
      target: { value: "Delivery plan" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add reference" }));

    await waitFor(() => {
      expect(dialogMocks.addDocument).toHaveBeenCalledWith(
        {
          taskId: "task-1",
          kind: DocumentKind.MARKDOWN,
          title: "Delivery plan",
          value: "# Delivery\n\nShip safely.",
          isImplementationPlan: true,
        },
        expect.anything(),
      );
    });
    expect(onClose).toHaveBeenCalledOnce();
    fireEvent.click(urlTab);
    expect(screen.getByLabelText("Document URL")).toHaveValue(
      "https://example.com/spec",
    );
  });

  it("treats picker cancellation as recoverable and displays picker errors", async () => {
    dialogMocks.selectLocalFile
      .mockResolvedValueOnce({ path: "", canceled: true })
      .mockRejectedValueOnce(new Error("picker unavailable"));
    renderDialog();
    fireEvent.click(screen.getByRole("tab", { name: "Local file" }));
    fireEvent.click(screen.getByRole("button", { name: "Choose file…" }));
    expect(await screen.findByText(/No file was selected/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("File path"), {
      target: { value: "/tmp/manual.md" },
    });
    expect(screen.queryByText(/No file was selected/)).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Choose file…" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "picker unavailable",
    );
  });

  it("supports Home, ArrowLeft, Escape, and the feature-only form", async () => {
    const { onClose } = renderDialog({ featureId: "feature-2" });
    const urlTab = screen.getByRole("tab", { name: "URL" });
    await waitFor(() => {
      expect(urlTab).toHaveFocus();
    });
    fireEvent.keyDown(urlTab, { key: "ArrowLeft" });
    const markdownTab = screen.getByRole("tab", { name: "Markdown" });
    expect(markdownTab).toHaveFocus();
    fireEvent.keyDown(markdownTab, { key: "Home" });
    expect(urlTab).toHaveFocus();
    expect(
      screen.queryByLabelText("Use as this task's implementation plan"),
    ).toBeNull();
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });
});
