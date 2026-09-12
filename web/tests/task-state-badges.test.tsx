import { cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { TaskBlockLabel, TaskDisplayState } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { taskDisplayStateToken } from "../src/i18n/domain";
import { StatusBadge } from "../src/views/StatusBadge";
import { TaskBlockLabels, TaskStatusBadge } from "../src/views/TaskStateBadges";

const presentations: {
  state: TaskDisplayState;
  label: string;
  iconClass: string;
  visual?: string;
}[] = [
  {
    state: TaskDisplayState.UNSPECIFIED,
    label: "unknown",
    iconClass: "lucide-circle-question-mark",
  },
  {
    state: TaskDisplayState.NOT_STARTED,
    label: "not started",
    iconClass: "lucide-circle-dashed",
  },
  {
    state: TaskDisplayState.DESIGNING,
    label: "designing",
    iconClass: "lucide-pencil-ruler",
    visual: "Design｜working",
  },
  {
    state: TaskDisplayState.DESIGNED,
    label: "designed",
    iconClass: "lucide-pencil-ruler",
    visual: "Design｜done",
  },
  {
    state: TaskDisplayState.IN_PROGRESS,
    label: "in progress",
    iconClass: "lucide-code-xml",
    visual: "Implementation｜working",
  },
  {
    state: TaskDisplayState.IMPLEMENTED,
    label: "implemented",
    iconClass: "lucide-code-xml",
    visual: "Implementation｜done",
  },
  {
    state: TaskDisplayState.IN_REVIEW,
    label: "in review",
    iconClass: "lucide-messages-square",
    visual: "Review｜waiting",
  },
  {
    state: TaskDisplayState.APPROVED,
    label: "approved",
    iconClass: "lucide-messages-square",
    visual: "Review｜approved",
  },
  {
    state: TaskDisplayState.COMPLETED,
    label: "completed",
    iconClass: "lucide-circle-check",
  },
  {
    state: TaskDisplayState.CLOSED,
    label: "closed",
    iconClass: "lucide-circle-x",
  },
  {
    state: TaskDisplayState.MERGED,
    label: "merged",
    iconClass: "lucide-circle-check",
  },
  {
    state: TaskDisplayState.UNKNOWN,
    label: "unknown",
    iconClass: "lucide-circle-question-mark",
  },
] as const;

describe("TaskStatusBadge", () => {
  afterEach(cleanup);

  beforeEach(async () => {
    await setDisplayLanguage("en");
  });

  it.each(presentations)(
    "renders the $state state with an icon, state class, and accessible name",
    ({ state, label, iconClass, visual }) => {
      const { container } = render(<TaskStatusBadge state={state} />);
      const badge = container.querySelector(".status-badge");
      if (!badge) throw new Error("status badge missing");
      const icon = badge.querySelector("svg");
      if (!icon) throw new Error("status badge icon missing");

      expect(badge).toHaveClass(`state-${taskDisplayStateToken(state)}`);
      expect(badge).toHaveAccessibleName(label);
      expect(icon).toHaveClass(iconClass);
      expect(icon).toHaveAttribute("aria-hidden", "true");
      if (visual) {
        expect(badge.querySelector(".status-badge-visual")).toHaveTextContent(
          visual,
        );
        expect(
          badge.querySelector(".status-badge-accessible"),
        ).toHaveTextContent(label);
      } else {
        expect(badge.querySelector(".status-badge-visual")).toBeNull();
        expect(badge).toHaveTextContent(label);
      }
    },
  );

  it("localizes the phase and in-phase state independently", async () => {
    const { container } = render(
      <TaskStatusBadge state={TaskDisplayState.IN_PROGRESS} />,
    );
    expect(container.querySelector(".status-badge-visual")).toHaveTextContent(
      "Implementation｜working",
    );

    await setDisplayLanguage("ja");
    expect(container.querySelector(".status-badge-visual")).toHaveTextContent(
      "実装｜作業中",
    );
    expect(
      container.querySelector(".status-badge-accessible"),
    ).toHaveTextContent("実装中");
  });

  it("keeps feature and block badges as plain labels", () => {
    const { container } = render(
      <>
        <StatusBadge className="status-active" label="Active" />
        <TaskBlockLabels labels={[TaskBlockLabel.CONFLICT]} />
      </>,
    );

    expect(container.querySelectorAll(".status-badge")).toHaveLength(2);
    expect(container.querySelectorAll(".status-badge-icon")).toHaveLength(0);
    expect(container.querySelectorAll(".status-badge-visual")).toHaveLength(0);
    expect(container.querySelector(".status-active")).toHaveTextContent(
      "Active",
    );
    expect(container.querySelector(".block-conflict")).toHaveTextContent(
      "conflict",
    );
  });
});
