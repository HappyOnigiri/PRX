import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ComponentProps, PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DocumentKind } from "../src/gen/prx/v1/prx_pb";
import { FeatureReferences } from "../src/views/FeatureReferences";
import { makeDocument } from "./factories";

const apiMocks = vi.hoisted(() => ({
  addDocument: vi.fn(),
  deleteDocument: vi.fn(),
  getDocument: vi.fn(),
  updateDocument: vi.fn(),
}));

vi.mock("../src/api", () => ({
  mutations: apiMocks,
}));

function renderReferences(
  props: Partial<ComponentProps<typeof FeatureReferences>> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  function Wrapper({ children }: PropsWithChildren) {
    return (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
  }
  const onPreview = vi.fn();
  const result = render(
    <FeatureReferences
      featureId="feature-1"
      documents={[
        makeDocument({
          id: "url-doc",
          featureId: "feature-1",
          taskId: "",
          title: "Runbook",
          locator: "https://example.com/runbook",
        }),
        makeDocument({
          id: "local-doc",
          featureId: "feature-1",
          taskId: "",
          kind: DocumentKind.LOCAL_FILE,
          title: "Architecture notes",
          locator: "docs/architecture.md",
        }),
        makeDocument({
          id: "markdown-doc",
          featureId: "feature-1",
          taskId: "",
          kind: DocumentKind.MARKDOWN,
          title: "Decision log",
          locator: "",
        }),
      ]}
      onPreview={onPreview}
      {...props}
    />,
    { wrapper: Wrapper },
  );
  return { ...result, onPreview };
}

describe("FeatureReferences", () => {
  beforeEach(() => {
    apiMocks.addDocument.mockReset().mockResolvedValue({});
    apiMocks.deleteDocument.mockReset().mockResolvedValue({});
    apiMocks.getDocument
      .mockReset()
      .mockResolvedValue({ content: "# Decision" });
    apiMocks.updateDocument.mockReset().mockResolvedValue({});
  });

  afterEach(() => {
    cleanup();
  });

  it("opens a labelled panel and closes it after preview, outside click, and Escape", () => {
    const { onPreview } = renderReferences();
    const trigger = screen.getByRole("button", { name: "References" });

    expect(trigger).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("region", { name: "References" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Add reference" })).toBeNull();

    fireEvent.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("region", { name: "References" })).toBeVisible();
    expect(screen.getByRole("link", { name: /Runbook/ })).toHaveAttribute(
      "target",
      "_blank",
    );

    fireEvent.click(
      screen.getByRole("button", {
        name: "Architecture notesdocs/architecture.md",
      }),
    );
    expect(onPreview).toHaveBeenCalledWith(
      expect.objectContaining({ id: "local-doc" }),
    );
    expect(trigger).toHaveAttribute("aria-expanded", "false");

    fireEvent.click(trigger);
    fireEvent.pointerDown(document.body);
    expect(trigger).toHaveAttribute("aria-expanded", "false");

    fireEvent.click(trigger);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    expect(trigger).toHaveFocus();
  });

  it("adds a reference in a modal, blocks duplicate close, and restores focus", async () => {
    let resolveAdd: (value: object) => void = () => undefined;
    apiMocks.addDocument.mockReturnValue(
      new Promise((resolve) => {
        resolveAdd = resolve;
      }),
    );
    renderReferences();
    const trigger = screen.getByRole("button", { name: "References" });
    fireEvent.click(trigger);
    fireEvent.click(screen.getByRole("button", { name: "Add reference" }));

    const dialog = screen.getByRole("dialog", {
      name: "Add feature reference",
    });
    expect(screen.queryByRole("region", { name: "References" })).toBeNull();
    const kind = screen.getByLabelText("Type");
    await waitFor(() => {
      expect(kind).toHaveFocus();
    });
    fireEvent.change(kind, {
      target: { value: String(DocumentKind.LOCAL_FILE) },
    });
    fireEvent.change(screen.getByLabelText("Title (optional)"), {
      target: { value: "Feature brief" },
    });
    fireEvent.change(screen.getByLabelText("URL or file path"), {
      target: { value: "README.md" },
    });

    const submit = screen.getByRole("button", { name: "Add reference" });
    fireEvent.click(submit);
    await waitFor(() => {
      expect(apiMocks.addDocument).toHaveBeenCalledWith(
        {
          featureId: "feature-1",
          kind: DocumentKind.LOCAL_FILE,
          title: "Feature brief",
          value: "README.md",
        },
        expect.anything(),
      );
    });
    await waitFor(() => {
      expect(submit).toBeDisabled();
    });
    fireEvent.keyDown(dialog, { key: "Escape" });
    expect(dialog).toBeInTheDocument();

    resolveAdd({});
    await waitFor(() => {
      expect(
        screen.queryByRole("dialog", { name: "Add feature reference" }),
      ).toBeNull();
    });
    await waitFor(() => {
      expect(trigger).toHaveFocus();
    });
  });

  it("keeps failed input for retry and resets the next dialog", async () => {
    apiMocks.addDocument.mockRejectedValueOnce(new Error("Reference failed"));
    renderReferences({ documents: [] });
    const trigger = screen.getByRole("button", { name: "References" });
    fireEvent.click(trigger);
    expect(screen.getByText("No references.")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Add reference" }));
    fireEvent.change(screen.getByLabelText("Title (optional)"), {
      target: { value: "Retry me" },
    });
    fireEvent.change(screen.getByLabelText("URL or file path"), {
      target: { value: "https://example.com/retry" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add reference" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Reference failed",
    );
    expect(screen.getByLabelText("Title (optional)")).toHaveValue("Retry me");
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(trigger);
    fireEvent.click(screen.getByRole("button", { name: "Add reference" }));
    expect(screen.getByLabelText("Type")).toHaveValue(String(DocumentKind.URL));
    expect(screen.getByLabelText("Title (optional)")).toHaveValue("");
  });

  it("traps focus inside the add dialog", async () => {
    renderReferences();
    fireEvent.click(screen.getByRole("button", { name: "References" }));
    fireEvent.click(screen.getByRole("button", { name: "Add reference" }));
    const kind = screen.getByLabelText("Type");
    const submit = screen.getByRole("button", { name: "Add reference" });
    await waitFor(() => {
      expect(kind).toHaveFocus();
    });

    fireEvent.keyDown(kind, { key: "Tab", shiftKey: true });
    expect(submit).toHaveFocus();
    fireEvent.keyDown(submit, { key: "Tab" });
    expect(kind).toHaveFocus();
  });

  it("supports Markdown editing and deletion without changing the task UI", async () => {
    renderReferences();
    fireEvent.click(screen.getByRole("button", { name: "References" }));
    fireEvent.click(screen.getByRole("button", { name: "Edit Decision log" }));

    await waitFor(() => {
      expect(apiMocks.getDocument).toHaveBeenCalledWith(
        "markdown-doc",
        expect.anything(),
      );
    });
    await screen.findByRole("textbox", {
      name: "Edit Decision log",
    });
    fireEvent.click(screen.getByRole("button", { name: "Cancel edit" }));
    expect(
      screen.queryByRole("textbox", { name: "Edit Decision log" }),
    ).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Edit Decision log" }));
    const reopenedEditor = await screen.findByRole("textbox", {
      name: "Edit Decision log",
    });
    fireEvent.change(reopenedEditor, { target: { value: "# Revised" } });
    fireEvent.click(screen.getByRole("button", { name: "Save reference" }));
    await waitFor(() => {
      expect(apiMocks.updateDocument).toHaveBeenCalledWith(
        {
          id: "markdown-doc",
          source: { case: "markdown", value: "# Revised" },
        },
        expect.anything(),
      );
    });

    fireEvent.click(
      screen.getByRole("button", { name: "Delete Architecture notes" }),
    );
    await waitFor(() => {
      expect(apiMocks.deleteDocument).toHaveBeenCalledWith(
        "local-doc",
        expect.anything(),
      );
    });
  });

  it("keeps archived references viewable while hiding every mutation", () => {
    const { onPreview } = renderReferences({ readOnly: true });
    fireEvent.click(screen.getByRole("button", { name: "References" }));

    expect(screen.queryByRole("button", { name: "Add reference" })).toBeNull();
    expect(
      screen.queryByRole("button", { name: "Edit Decision log" }),
    ).toBeNull();
    expect(
      screen.queryByRole("button", { name: /Delete Architecture notes/ }),
    ).toBeNull();
    fireEvent.click(
      screen.getByRole("button", {
        name: "Architecture notesdocs/architecture.md",
      }),
    );
    expect(onPreview).toHaveBeenCalledTimes(1);
  });
});
