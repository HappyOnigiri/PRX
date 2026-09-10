import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TaskDisplayState } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { BatchPromptDialog } from "../src/views/BatchPromptDialog";
import { makeTask } from "./factories";

const batchMocks = vi.hoisted(() => ({ getBatchPrompt: vi.fn() }));

vi.mock("../src/api", () => ({ getBatchPrompt: batchMocks.getBatchPrompt }));

const designed = {
  displayState: TaskDisplayState.DESIGNED,
  hasImplementationPlan: true,
  ready: true,
};

function renderDialog(onClose = vi.fn()) {
  return render(
    <BatchPromptDialog
      featureId="feature-1"
      tasks={[
        makeTask({ id: "task-1", title: "Build API", ...designed }),
        makeTask({ id: "task-2", title: "Bill the order", ...designed }),
        // 未着手。まだ何も設計されていない。
        makeTask({ id: "task-3", title: "Draft the schema" }),
        // 設計済みで、batch に含められる task を待っているため、その task の
        // 後ろに続けて引き渡せる。
        makeTask({
          id: "task-4",
          title: "Ship the change",
          ...designed,
          ready: false,
          pendingBlockerTaskIds: ["task-1"],
        }),
        // 計画のない task を待っている。どの batch も引き渡せないので、これを
        // 選べるようになることはない。
        makeTask({
          id: "task-5",
          title: "Archive the ledger",
          ...designed,
          ready: false,
          pendingBlockerTaskIds: ["task-3"],
        }),
        // batch に含められる task を 2 つ待っているため、pull request を積む
        // 起点が 1 つに定まらない。
        makeTask({
          id: "task-6",
          title: "Close the books",
          ...designed,
          ready: false,
          pendingBlockerTaskIds: ["task-1", "task-2"],
        }),
      ]}
      onClose={onClose}
    />,
  );
}

// 行そのものがコントロールなので、task はチェックボックスのラベルではなく
// タイトルと識別子を持つボタンで指定する。
function taskRow(title: string) {
  return screen.getByRole("button", { name: new RegExp(title) });
}

function includeBlocked() {
  return screen.getByLabelText("Include dependent tasks");
}

function stubClipboard(writeText: ReturnType<typeof vi.fn>) {
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
}

describe("BatchPromptDialog", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    await setDisplayLanguage("en");
  });

  afterEach(() => {
    cleanup();
  });

  // 着手可否と計画の有無はどちらもサーバー側の導出なので、ダイアログは今すぐ
  // 着手できるとサーバーが示した task だけを出す。
  it("offers the designed tasks whose blockers are clear", () => {
    renderDialog();

    expect(taskRow("Build API")).toBeInTheDocument();
    expect(taskRow("Bill the order")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Draft the schema/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Ship the change/ }),
    ).not.toBeInTheDocument();
    expect(includeBlocked()).not.toBeChecked();
    expect(screen.getByText("0 of 2 selected")).toBeInTheDocument();
    // まだ何も選択していないので、コピーするプロンプトがない。
    expect(screen.getByRole("button", { name: "Copy prompt" })).toBeDisabled();
  });

  it("copies one prompt for the tasks selected in the listed order", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    batchMocks.getBatchPrompt.mockResolvedValue({
      featureId: "feature-1",
      taskIds: ["task-1", "task-2"],
      prompt: "Batch feature-1",
    });
    renderDialog();

    // 後ろの task を先にクリックしてもリクエストの順序は変わらない。読み手は
    // 見えていた順に batch を引き渡す。
    fireEvent.click(taskRow("Bill the order"));
    fireEvent.click(taskRow("Build API"));
    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText("Copied a prompt for 2 tasks."),
      ).toBeInTheDocument();
    });
    expect(batchMocks.getBatchPrompt).toHaveBeenCalledWith("feature-1", [
      "task-1",
      "task-2",
    ]);
    expect(writeText).toHaveBeenCalledWith("Batch feature-1");
  });

  it("selects and clears every offered task with one control", () => {
    renderDialog();

    fireEvent.click(screen.getByRole("button", { name: "Select all" }));
    expect(screen.getByText("2 of 2 selected")).toBeInTheDocument();
    expect(taskRow("Build API")).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(screen.getByRole("button", { name: "Clear selection" }));
    expect(screen.getByText("0 of 2 selected")).toBeInTheDocument();
    expect(taskRow("Build API")).toHaveAttribute("aria-pressed", "false");
  });

  // 行の選択は背後にチェックボックスのないポインタ操作なので、行自体が
  // キーボードで到達して操作できるコントロールである必要がある。
  it("offers each row as a focusable toggle", () => {
    renderDialog();

    const row = taskRow("Build API");
    expect(row).toHaveAttribute("type", "button");
    expect(row).toHaveAttribute("aria-pressed", "false");
    row.focus();
    expect(row).toHaveFocus();
  });

  // ブロックされた task も待ち先の作業が一緒に渡るなら引き渡せるため、依存
  // task を含める指示があったときだけ選択肢に出す。
  it("offers a blocked task behind the work it waits for", () => {
    renderDialog();

    fireEvent.click(includeBlocked());

    const blocked = taskRow("Ship the change");
    expect(blocked).toBeDisabled();
    expect(blocked).toHaveTextContent("after task-1");
    expect(screen.getByText("0 of 3 selected")).toBeInTheDocument();
    // ブロッカーに計画がないので、どの batch もこれを運べない。
    expect(
      screen.queryByRole("button", { name: /Archive the ledger/ }),
    ).not.toBeInTheDocument();
    // ブロッカーが 2 つあると積む先の pull request が定まらないため、batch が
    // 両方を運べても選択肢には出さない。
    expect(
      screen.queryByRole("button", { name: /Close the books/ }),
    ).not.toBeInTheDocument();

    fireEvent.click(taskRow("Build API"));
    expect(taskRow("Ship the change")).toBeEnabled();
    fireEvent.click(taskRow("Ship the change"));
    expect(taskRow("Ship the change")).toHaveAttribute("aria-pressed", "true");
  });

  // ブロッカーを外すとその上に積んだ作業が宙に浮くため、土台のない作業を
  // 選択に残してはいけない。
  it("clears a dependent task when its blocker leaves the selection", () => {
    renderDialog();

    fireEvent.click(includeBlocked());
    fireEvent.click(taskRow("Build API"));
    fireEvent.click(taskRow("Ship the change"));
    expect(screen.getByText("2 of 3 selected")).toBeInTheDocument();

    fireEvent.click(taskRow("Build API"));
    expect(screen.getByText("0 of 3 selected")).toBeInTheDocument();
    expect(taskRow("Ship the change")).toBeDisabled();
  });

  it("drops the blocked tasks again when the option is turned off", () => {
    renderDialog();

    fireEvent.click(includeBlocked());
    fireEvent.click(screen.getByRole("button", { name: "Select all" }));
    expect(screen.getByText("3 of 3 selected")).toBeInTheDocument();

    fireEvent.click(includeBlocked());
    expect(screen.getByText("2 of 2 selected")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Ship the change/ }),
    ).not.toBeInTheDocument();
  });

  // batch はブロッカーをその上に積んだ作業より先に並べるため、受け取る agent
  // は実行できる順序でリストを読める。
  it("copies the tasks with each blocker ahead of what waits for it", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    batchMocks.getBatchPrompt.mockResolvedValue({
      featureId: "feature-1",
      taskIds: ["task-1", "task-2", "task-4"],
      prompt: "Batch feature-1",
    });
    renderDialog();

    fireEvent.click(includeBlocked());
    fireEvent.click(screen.getByRole("button", { name: "Select all" }));
    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText("Copied a prompt for 3 tasks."),
      ).toBeInTheDocument();
    });
    expect(batchMocks.getBatchPrompt).toHaveBeenCalledWith("feature-1", [
      "task-1",
      "task-2",
      "task-4",
    ]);
  });

  // サーバーは原因の task やテンプレートを名指しするので、そのメッセージは
  // そのまま読み手に届ける。
  it("reports a server failure without writing to the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    batchMocks.getBatchPrompt.mockRejectedValue(
      new Error('task "task-2" was not found'),
    );
    renderDialog();

    fireEvent.click(screen.getByRole("button", { name: "Select all" }));
    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText('task "task-2" was not found'),
      ).toBeInTheDocument();
    });
    expect(writeText).not.toHaveBeenCalled();
  });

  // クリップボードの拒否はブラウザ内部のメッセージを伴い、セキュアコンテキスト
  // 外では API 自体が存在しない。
  it("reports a clipboard failure in the display language", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("clipboard blocked")));
    batchMocks.getBatchPrompt.mockResolvedValue({
      featureId: "feature-1",
      taskIds: ["task-1"],
      prompt: "Batch feature-1",
    });
    renderDialog();

    fireEvent.click(taskRow("Build API"));
    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));

    await waitFor(() => {
      expect(
        screen.getByText("The prompt could not be copied."),
      ).toBeInTheDocument();
    });
    expect(screen.queryByText("clipboard blocked")).not.toBeInTheDocument();
  });

  it("says when the feature has nothing ready to implement", () => {
    const onClose = vi.fn();
    render(
      <BatchPromptDialog
        featureId="feature-1"
        tasks={[makeTask({ id: "task-3", title: "Draft the schema" })]}
        onClose={onClose}
      />,
    );

    expect(
      screen.getByText("No task in this feature is ready to implement."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy prompt" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalled();
  });

  it("closes on Escape", () => {
    const onClose = vi.fn();
    renderDialog(onClose);

    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });
});
