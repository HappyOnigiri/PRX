import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { FeatureTabId } from "../src/feature-tabs";
import { DocumentKind, type Snapshot } from "../src/gen/prx/v1/prx_pb";
import { ProjectWorkspace } from "../src/views/ProjectWorkspace";
import {
  makeDocument,
  makeFeature,
  makeProject,
  makeSnapshot,
} from "./factories";

const workspaceMocks = vi.hoisted(() => {
  const search: { features: FeatureTabId } = { features: "active" };
  return {
    navigate: vi.fn().mockResolvedValue(undefined),
    projectId: "P-1",
    search,
    snapshot: { data: undefined as Snapshot | undefined, isPending: false },
  };
});

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => (
    <a href="#test">{children}</a>
  ),
  useNavigate: () => workspaceMocks.navigate,
  useParams: () => ({ projectId: workspaceMocks.projectId }),
  useSearch: () => workspaceMocks.search,
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => workspaceMocks.snapshot,
  useDomainMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
    error: null,
  }),
}));
vi.mock("../src/api", () => ({
  mutations: {
    createFeature: vi.fn(),
    updateProject: vi.fn(),
    deleteProject: vi.fn(),
    deleteDocument: vi.fn(),
    updateDocument: vi.fn(),
    getDocument: vi.fn(),
    addDocument: vi.fn(),
  },
  selectLocalFile: vi.fn(),
}));
vi.mock("../src/views/MarkdownPreview", () => ({
  MarkdownPreview: ({
    document,
    onClose,
  }: {
    document: { title: string };
    onClose: () => void;
  }) => (
    <div role="dialog" aria-label="Mock Markdown preview">
      <p>{document.title}</p>
      <button onClick={onClose}>Mock close preview</button>
    </div>
  ),
}));

function populatedSnapshot(archived = false) {
  return makeSnapshot({
    projects: [
      makeProject({
        id: "P-1",
        title: "Delivery platform",
        description: "Shared work",
        archived,
      }),
    ],
    features: [
      makeFeature({
        id: "F-1",
        title: "Checkout rollout",
        projectId: "P-1",
        taskCount: 4,
        mergedCount: 2,
        finishedCount: 3,
      }),
      makeFeature({ id: "F-2", title: "Search revamp" }),
      makeFeature({
        id: "F-3",
        title: "Legacy checkout",
        projectId: "P-1",
        archived: true,
      }),
    ],
    documents: [
      makeDocument({
        id: "charter",
        taskId: "",
        projectId: "P-1",
        kind: DocumentKind.LOCAL_FILE,
        title: "Platform charter",
        locator: "docs/charter.md",
      }),
      makeDocument({ id: "task-doc", title: "Task runbook" }),
    ],
    tasks: [],
  });
}

describe("ProjectWorkspace", () => {
  afterEach(cleanup);
  beforeEach(() => {
    workspaceMocks.projectId = "P-1";
    workspaceMocks.search.features = "active";
    workspaceMocks.snapshot.data = undefined;
    workspaceMocks.snapshot.isPending = false;
    workspaceMocks.navigate.mockClear();
  });

  it("reports loading and a missing project", () => {
    workspaceMocks.snapshot.isPending = true;
    const { rerender } = render(<ProjectWorkspace />);
    expect(
      screen.getByRole("heading", { name: "Loading project…" }),
    ).toBeInTheDocument();

    workspaceMocks.snapshot.isPending = false;
    workspaceMocks.projectId = "P-9";
    workspaceMocks.snapshot.data = populatedSnapshot();
    rerender(<ProjectWorkspace />);
    expect(
      screen.getByRole("heading", { name: "Project not found" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Return to projects" }));
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({
      to: "/projects",
      search: { archived: false },
    });
  });

  // このページはプロジェクトに属する feature と document だけを表示し、
  // プロジェクト ID が空の未所属のものは出さない。
  it("lists its own features and references and opens the edit dialog", () => {
    workspaceMocks.snapshot.data = populatedSnapshot();
    render(<ProjectWorkspace />);

    expect(
      screen.getByRole("heading", { name: "Delivery platform", level: 1 }),
    ).toBeInTheDocument();
    // アクセシビリティツリーに出るのは開いているパネルだけで、畳んだ側も
    // 行は DOM に残るため、検証は開いているパネルから読む。
    const features = screen.getByRole("tabpanel");
    expect(features).toHaveTextContent("Checkout rollout");
    expect(features).not.toHaveTextContent("Search revamp");
    expect(features).not.toHaveTextContent("Legacy checkout");
    // 進捗は merge 済みだけでなく完了したタスクをすべて数えるので、
    // pull request なしで閉じた 3 つ目のタスクもここに含まれる。
    expect(features).toHaveTextContent("3/4 finished");

    fireEvent.click(screen.getByRole("button", { name: "References" }));
    expect(screen.getByText("Platform charter")).toBeInTheDocument();
    expect(screen.queryByText("Task runbook")).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Add reference" }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Edit project" }));
    expect(
      screen.getByRole("form", { name: "Edit project" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("form", { name: "Edit project" }),
    ).not.toBeInTheDocument();
  });

  // ステータスの絞り込みは search parameter なので、タブを選ぶと遷移し、
  // ページは遷移後の URL が示すとおりに描画する。
  it("narrows the feature list to the tab the URL names", () => {
    workspaceMocks.snapshot.data = populatedSnapshot();
    const { rerender } = render(<ProjectWorkspace />);

    fireEvent.click(screen.getByRole("tab", { name: "Archived" }));
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "P-1" },
      search: { features: "archived" },
    });

    workspaceMocks.search.features = "archived";
    rerender(<ProjectWorkspace />);
    const features = screen.getByRole("tabpanel");
    expect(features).toHaveTextContent("Legacy checkout");
    expect(features).not.toHaveTextContent("Checkout rollout");

    workspaceMocks.search.features = "completed";
    rerender(<ProjectWorkspace />);
    expect(
      screen.getByRole("heading", { name: "No completed features" }),
    ).toBeInTheDocument();
  });

  // feature は所属先のプロジェクトから作るので、操作はここに置き、
  // プロジェクトを選ぶ項目は要らない。
  it("opens the feature creation dialog for this project", () => {
    workspaceMocks.snapshot.data = populatedSnapshot();
    render(<ProjectWorkspace />);

    fireEvent.click(screen.getByRole("button", { name: "Create feature" }));
    expect(
      screen.getByRole("form", { name: "Create feature" }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Project")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("form", { name: "Create feature" }),
    ).not.toBeInTheDocument();
  });

  it("presents an archived project without reference editing", () => {
    workspaceMocks.snapshot.data = populatedSnapshot(true);
    render(<ProjectWorkspace />);

    expect(screen.getByText("Archived · read-only")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Create feature" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "References" }));
    expect(
      screen.queryByRole("button", { name: "Add reference" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Manage project" }));
    expect(
      screen.getByRole("dialog", { name: "Manage archived project" }),
    ).toBeInTheDocument();
  });

  it("previews a stored reference and returns from the preview", () => {
    workspaceMocks.snapshot.data = populatedSnapshot();
    render(<ProjectWorkspace />);

    fireEvent.click(screen.getByRole("button", { name: "References" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Platform charterdocs/charter.md" }),
    );
    expect(
      screen.getByRole("dialog", { name: "Mock Markdown preview" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Mock close preview" }));
    expect(
      screen.queryByRole("dialog", { name: "Mock Markdown preview" }),
    ).not.toBeInTheDocument();
  });

  it("navigates back to the list after the project is deleted", async () => {
    workspaceMocks.snapshot.data = populatedSnapshot();
    render(<ProjectWorkspace />);

    fireEvent.click(screen.getByRole("button", { name: "Edit project" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete project" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete permanently" }));
    await waitFor(() => {
      expect(workspaceMocks.navigate).toHaveBeenCalledWith({
        to: "/projects",
        search: { archived: false },
      });
    });
  });
});
