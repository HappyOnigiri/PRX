import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setDisplayLanguage } from "../src/i18n";
import { PromptSettingsPanel } from "../src/views/PromptSettingsPanel";

interface PromptDraft {
  design: string;
  implementation: string;
}

const panelMocks = vi.hoisted(() => ({
  updateTemplates: vi.fn(),
  templates: {
    data: undefined as
      | {
          design: string;
          implementation: string;
          supportedPlaceholders: string[];
          requiredPlaceholder: string;
          builtIn: { design: string; implementation: string };
        }
      | undefined,
    isPending: false,
    error: null as Error | null,
  },
  mutation: {
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
}));

vi.mock("../src/api", () => ({
  promptMutations: { updateTemplates: panelMocks.updateTemplates },
}));
vi.mock("../src/hooks", () => ({
  usePromptTemplates: () => panelMocks.templates,
  usePromptTemplatesMutation: () => panelMocks.mutation,
}));

function submitTemplateForm() {
  const form = screen.getByRole("button", { name: "Save" }).closest("form");
  if (!(form instanceof HTMLFormElement))
    throw new Error("template form missing");
  fireEvent.submit(form);
}

describe("PromptSettingsPanel", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    await setDisplayLanguage("en");
    panelMocks.templates.data = {
      design: "Design {{task_id}}",
      implementation: "Build {{task_id}}",
      supportedPlaceholders: ["task_id", "feature_id"],
      requiredPlaceholder: "task_id",
      builtIn: {
        design: "Built-in design {{task_id}}",
        implementation: "Built-in build {{task_id}}",
      },
    };
    panelMocks.templates.isPending = false;
    panelMocks.templates.error = null;
    panelMocks.mutation.isPending = false;
    panelMocks.mutation.error = null;
    panelMocks.mutation.mutateAsync.mockResolvedValue({});
  });

  afterEach(() => {
    cleanup();
  });

  it("reports that the templates are still loading", () => {
    panelMocks.templates.isPending = true;
    render(<PromptSettingsPanel />);
    expect(screen.getByText("Loading prompt templates…")).toBeInTheDocument();
  });

  it("reports why the templates could not be read", () => {
    panelMocks.templates.data = undefined;
    panelMocks.templates.error = new Error("config file is unreadable");
    render(<PromptSettingsPanel />);
    expect(screen.getByText("config file is unreadable")).toBeInTheDocument();
  });

  // The vocabulary is whatever the server reports, so adding or removing a
  // placeholder there never leaves this hint advertising a rejected name.
  it("lists the placeholders the server reported", () => {
    panelMocks.templates.data = {
      design: "Design {{task_id}}",
      implementation: "Build {{task_id}}",
      supportedPlaceholders: ["task_ref", "milestone_id"],
      requiredPlaceholder: "task_ref",
      builtIn: {
        design: "Built-in design {{task_ref}}",
        implementation: "Built-in build {{task_ref}}",
      },
    };
    render(<PromptSettingsPanel />);
    expect(
      screen.getByText(/\{\{task_ref\}\}, \{\{milestone_id\}\}/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/\{\{task_ref\}\} is required/),
    ).toBeInTheDocument();
  });

  it("saves both templates in one write and confirms the result", async () => {
    render(<PromptSettingsPanel />);
    fireEvent.change(screen.getByLabelText(/Design prompt/), {
      target: { value: "Plan {{task_id}}" },
    });
    fireEvent.change(screen.getByLabelText(/Implementation prompt/), {
      target: { value: "Ship {{task_id}}" },
    });
    submitTemplateForm();

    await waitFor(() => {
      expect(panelMocks.mutation.mutateAsync).toHaveBeenCalledWith({
        design: "Plan {{task_id}}",
        implementation: "Ship {{task_id}}",
      });
    });
    expect(
      await screen.findByText("Prompt templates saved."),
    ).toBeInTheDocument();
  });

  // The built-in text comes from the server, so restoring shows what the save
  // will write instead of emptying the fields until the write comes back.
  it("puts the built-in templates in the fields before saving them", async () => {
    render(<PromptSettingsPanel />);
    fireEvent.click(
      screen.getByRole("button", { name: "Restore built-in templates" }),
    );

    expect(screen.getByLabelText(/Design prompt/)).toHaveValue(
      "Built-in design {{task_id}}",
    );
    expect(screen.getByLabelText(/Implementation prompt/)).toHaveValue(
      "Built-in build {{task_id}}",
    );
    submitTemplateForm();

    await waitFor(() => {
      expect(panelMocks.mutation.mutateAsync).toHaveBeenCalledWith({
        design: "Built-in design {{task_id}}",
        implementation: "Built-in build {{task_id}}",
      });
    });
  });

  // The response carries the text that was sent, so it is older than anything
  // typed while the request was in flight.
  it("keeps text typed while a save is in flight", async () => {
    let settle: (result: { templates: PromptDraft }) => void = () => undefined;
    panelMocks.mutation.mutateAsync.mockImplementation(
      () =>
        new Promise<{ templates: PromptDraft }>((resolve) => {
          settle = resolve;
        }),
    );
    render(<PromptSettingsPanel />);
    fireEvent.change(screen.getByLabelText(/Design prompt/), {
      target: { value: "Plan {{task_id}}" },
    });
    submitTemplateForm();
    await waitFor(() => {
      expect(panelMocks.mutation.mutateAsync).toHaveBeenCalled();
    });

    fireEvent.change(screen.getByLabelText(/Design prompt/), {
      target: { value: "Plan {{task_id}} carefully" },
    });
    settle({
      templates: {
        design: "Plan {{task_id}}",
        implementation: "Build {{task_id}}",
      },
    });

    await waitFor(() => {
      expect(screen.getByLabelText(/Design prompt/)).toHaveValue(
        "Plan {{task_id}} carefully",
      );
    });
    expect(
      screen.queryByText("Prompt templates saved."),
    ).not.toBeInTheDocument();
  });

  it("shows a rejected template and keeps the edited text", async () => {
    panelMocks.mutation.mutateAsync.mockRejectedValue(
      new Error("prompts.design: template must use {{task_id}}"),
    );
    panelMocks.mutation.error = new Error(
      "prompts.design: template must use {{task_id}}",
    );
    render(<PromptSettingsPanel />);
    fireEvent.change(screen.getByLabelText(/Design prompt/), {
      target: { value: "no placeholder" },
    });
    submitTemplateForm();

    await waitFor(() => {
      expect(
        screen.getByText("prompts.design: template must use {{task_id}}"),
      ).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/Design prompt/)).toHaveValue(
      "no placeholder",
    );
    expect(
      screen.queryByText("Prompt templates saved."),
    ).not.toBeInTheDocument();
  });
});
