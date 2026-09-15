import { create } from "@bufbuild/protobuf";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TaskLabelOverridesSchema, TaskStatus } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import {
  TaskLabelOverridesTabPanel,
  TaskLabelSettingsPanel,
} from "../src/views/TaskLabelOverridesPanel";
import { useSettingsSections } from "../src/views/settingsSections";
import { taskStatusLabelKey } from "../src/views/taskStatusLabelKey";

const mocks = vi.hoisted(() => ({
  getTaskLabelConfig: vi.fn(),
  updateTaskLabels: vi.fn(),
}));

vi.mock("../src/api", () => ({
  configMutations: { updateTaskLabels: mocks.updateTaskLabels },
  getTaskLabelConfig: mocks.getTaskLabelConfig,
}));

const config = {
  keys: [
    "status.not_started",
    "status.designing",
    "status.designed",
    "status.in_progress",
    "status.implemented",
    "status.in_review",
    "status.approved",
    "status.completed",
    "status.merged",
    "status.closed",
    "status.unknown",
    "block.dependency_unresolved",
    "block.conflict",
    "block.changes_requested",
    "block.ci_failed",
    "block.unknown",
  ],
  maxTextCodepoints: 32,
  overrides: { values: {} },
  builtIn: {
    values: [
      {
        key: "status.in_progress",
        text: "in_progress",
        color: "#ffffff",
      },
      {
        key: "block.dependency_unresolved",
        text: "dependency_unresolved",
        color: "#123456",
      },
    ],
  },
};

function Wrapper({ children }: { children: React.ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

function SettingsWrapper({ children }: { children: React.ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const sections = useSettingsSections();
  return (
    <QueryClientProvider client={queryClient}>
      {sections.provider(
        <>
          {children}
          <button
            type="button"
            disabled={!sections.dirty}
            onClick={() => {
              void sections.saveAll();
            }}
          >
            Save labels
          </button>
        </>,
      )}
    </QueryClientProvider>
  );
}

describe("task label settings", () => {
  afterEach(cleanup);

  beforeEach(async () => {
    await setDisplayLanguage("en");
    mocks.getTaskLabelConfig.mockResolvedValue(config);
    mocks.updateTaskLabels.mockResolvedValue({});
  });

  it("renders effective values, previews both themes, validates drafts, and saves", async () => {
    render(<TaskLabelSettingsPanel />, { wrapper: SettingsWrapper });
    const textInput = await screen.findByRole("textbox", {
      name: "in progress Text",
    });
    const colorInput = screen.getByRole("textbox", {
      name: "in progress Color",
    });
    fireEvent.change(textInput, { target: { value: "Coding" } });
    fireEvent.change(colorInput, { target: { value: "#ffffff" } });
    const saveButton = screen.getByRole("button", { name: "Save labels" });
    await waitFor(() => {
      expect(saveButton).toBeEnabled();
    });
    fireEvent.click(saveButton);
    await waitFor(() => {
      expect(mocks.updateTaskLabels).toHaveBeenCalled();
    });
    expect(screen.getAllByText(/low contrast/i).length).toBeGreaterThan(0);
    fireEvent.change(textInput, { target: { value: "bad\u0001text" } });
    expect(screen.getAllByText(/control characters/i).length).toBeGreaterThan(
      0,
    );
    fireEvent.change(textInput, { target: { value: "a".repeat(33) } });
    expect(
      screen.getAllByText(/32 characters or fewer/i).length,
    ).toBeGreaterThan(0);
    fireEvent.change(textInput, { target: { value: "Coding" } });
    fireEvent.change(colorInput, { target: { value: "red" } });
    expect(screen.getAllByText(/#RRGGBB/i).length).toBeGreaterThan(0);
    fireEvent.change(colorInput, { target: { value: "#ffffff" } });
    const inheritText = screen.getAllByRole("button", {
      name: "Inherit text",
    })[3];
    const inheritColor = screen.getAllByRole("button", {
      name: "Inherit color",
    })[3];
    if (!inheritText || !inheritColor)
      throw new Error("inherit buttons not found");
    fireEvent.click(inheritText);
    fireEvent.click(inheritColor);
    expect(mocks.updateTaskLabels).toHaveBeenCalled();
  });

  it("mounts the scoped editor only after its labels tab is opened", async () => {
    const onStateChange = vi.fn();
    const { rerender } = render(
      <TaskLabelOverridesTabPanel
        active={false}
        idPrefix="project-edit"
        labelsMounted={false}
        scope="project"
        editable
        onStateChange={onStateChange}
      />,
      { wrapper: Wrapper },
    );
    expect(screen.queryByText("Statuses")).not.toBeInTheDocument();
    rerender(
      <TaskLabelOverridesTabPanel
        active
        idPrefix="project-edit"
        labelsMounted
        scope="project"
        overrides={create(TaskLabelOverridesSchema)}
        parentOverrides={create(TaskLabelOverridesSchema)}
        editable
        onStateChange={onStateChange}
      />,
    );
    expect(await screen.findByText("Statuses")).toBeInTheDocument();
    expect(onStateChange).toHaveBeenCalled();
  });

  it("maps editable task statuses to label keys", () => {
    expect(taskStatusLabelKey(TaskStatus.UNSPECIFIED)).toBe("status.unknown");
    expect(taskStatusLabelKey(TaskStatus.NOT_STARTED)).toBe(
      "status.not_started",
    );
    expect(taskStatusLabelKey(TaskStatus.DESIGNING)).toBe("status.designing");
    expect(taskStatusLabelKey(TaskStatus.IN_PROGRESS)).toBe(
      "status.in_progress",
    );
    expect(taskStatusLabelKey(TaskStatus.COMPLETED)).toBe("status.completed");
    expect(taskStatusLabelKey(TaskStatus.CLOSED)).toBe("status.closed");
  });
});
