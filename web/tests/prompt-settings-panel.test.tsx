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
  batch: string;
}

const panelMocks = vi.hoisted(() => ({
  updateTemplates: vi.fn(),
  templates: {
    data: undefined as
      | {
          design: string;
          implementation: string;
          batch: string;
          supportedPlaceholders: string[];
          requiredPlaceholder: string;
          batchSupportedPlaceholders: string[];
          batchRequiredPlaceholder: string;
          builtIn: { design: string; implementation: string; batch: string };
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
      batch: "Batch {{task_list}}",
      supportedPlaceholders: ["task_id", "feature_id"],
      requiredPlaceholder: "task_id",
      batchSupportedPlaceholders: ["task_list", "feature_id"],
      batchRequiredPlaceholder: "task_list",
      builtIn: {
        design: "Built-in design {{task_id}}",
        implementation: "Built-in build {{task_id}}",
        batch: "Built-in batch {{task_list}}",
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

  // 語彙はサーバーが返すものがすべてなので、そこで placeholder を増減しても
  // このヒントが拒否される名前を案内し続けることはない。
  it("lists the placeholders the server reported", () => {
    panelMocks.templates.data = {
      design: "Design {{task_id}}",
      implementation: "Build {{task_id}}",
      batch: "Batch {{task_group}}",
      supportedPlaceholders: ["task_ref", "milestone_id"],
      requiredPlaceholder: "task_ref",
      batchSupportedPlaceholders: ["task_group"],
      batchRequiredPlaceholder: "task_group",
      builtIn: {
        design: "Built-in design {{task_ref}}",
        implementation: "Built-in build {{task_ref}}",
        batch: "Built-in batch {{task_group}}",
      },
    };
    render(<PromptSettingsPanel />);
    expect(
      screen.getByText(/\{\{task_ref\}\}, \{\{milestone_id\}\}/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/\{\{task_ref\}\} is required/),
    ).toBeInTheDocument();
    // batch テンプレートは独自の語彙を受け付けるので、batch 欄は上の
    // ヒントを共有せず自分のヒントを持つ。
    expect(
      screen.getByText(/\{\{task_group\}\} is required/),
    ).toBeInTheDocument();
  });

  it("saves every template in one write and confirms the result", async () => {
    render(<PromptSettingsPanel />);
    fireEvent.change(screen.getByLabelText(/Design prompt/), {
      target: { value: "Plan {{task_id}}" },
    });
    fireEvent.change(screen.getByLabelText(/Implementation prompt/), {
      target: { value: "Ship {{task_id}}" },
    });
    fireEvent.change(screen.getByLabelText(/Batch implementation prompt/), {
      target: { value: "Group {{task_list}}" },
    });
    submitTemplateForm();

    await waitFor(() => {
      expect(panelMocks.mutation.mutateAsync).toHaveBeenCalledWith({
        design: "Plan {{task_id}}",
        implementation: "Ship {{task_id}}",
        batch: "Group {{task_list}}",
      });
    });
    expect(
      await screen.findByText("Prompt templates saved."),
    ).toBeInTheDocument();
  });

  // 組み込みの文面はサーバー由来なので、復元時は書き込みの応答を待って欄を
  // 空にするのではなく、保存される内容をそのまま表示する。
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
    expect(screen.getByLabelText(/Batch implementation prompt/)).toHaveValue(
      "Built-in batch {{task_list}}",
    );
    submitTemplateForm();

    await waitFor(() => {
      expect(panelMocks.mutation.mutateAsync).toHaveBeenCalledWith({
        design: "Built-in design {{task_id}}",
        implementation: "Built-in build {{task_id}}",
        batch: "Built-in batch {{task_list}}",
      });
    });
  });

  // 応答は送信した文面を返すだけなので、リクエスト中に入力された内容より
  // 古い。
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
        batch: "Batch {{task_list}}",
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
