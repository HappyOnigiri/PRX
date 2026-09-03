import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Snapshot } from "../src/gen/prx/v1/prx_pb";
import { ProjectListPage } from "../src/views/ProjectListPage";
import { makeProject, makeSnapshot } from "./factories";

const listMocks = vi.hoisted(() => ({
  navigate: vi.fn().mockResolvedValue(undefined),
  createProject: vi.fn().mockResolvedValue({}),
  search: { archived: false },
  state: {
    data: undefined as Snapshot | undefined,
    isPending: true,
    error: null as Error | null,
    refetch: vi.fn(),
  },
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => (
    <a href="#test">{children}</a>
  ),
  useNavigate: () => listMocks.navigate,
  useSearch: () => listMocks.search,
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => listMocks.state,
  useDomainMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: listMocks.createProject,
    isPending: false,
    error: null,
  }),
}));
vi.mock("../src/api", () => ({ mutations: { createProject: vi.fn() } }));

describe("ProjectListPage", () => {
  afterEach(cleanup);
  beforeEach(() => {
    listMocks.state.data = undefined;
    listMocks.state.isPending = true;
    listMocks.state.error = null;
    listMocks.state.refetch.mockReset();
    listMocks.navigate.mockClear();
    listMocks.createProject.mockReset();
    listMocks.createProject.mockResolvedValue({});
    listMocks.search.archived = false;
  });

  it("renders loading, error, and retry states", () => {
    const { rerender } = render(<ProjectListPage />);
    expect(
      screen.getByRole("heading", { name: "Loading projects…" }),
    ).toBeInTheDocument();

    listMocks.state.isPending = false;
    listMocks.state.error = new Error("database unavailable");
    rerender(<ProjectListPage />);
    expect(screen.getByText("database unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(listMocks.state.refetch).toHaveBeenCalledOnce();
  });

  it("explains an empty list differently for the archived view", () => {
    listMocks.state.isPending = false;
    listMocks.state.data = makeSnapshot({ projects: [], features: [] });
    const { rerender } = render(<ProjectListPage />);
    expect(
      screen.getByRole("heading", { name: "No projects yet" }),
    ).toBeInTheDocument();

    listMocks.search.archived = true;
    rerender(<ProjectListPage />);
    expect(
      screen.getByRole("heading", { name: "No archived projects" }),
    ).toBeInTheDocument();
  });

  it("creates a project and opens its workspace", async () => {
    listMocks.state.isPending = false;
    listMocks.state.data = makeSnapshot({ projects: [], features: [] });
    listMocks.createProject.mockResolvedValue({
      project: makeProject({ id: "P-7" }),
    });
    render(<ProjectListPage />);

    fireEvent.click(screen.getByRole("button", { name: "New project" }));
    fireEvent.change(screen.getByLabelText("Slug"), {
      target: { value: "delivery" },
    });
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Delivery platform" },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Create project" }));
    await waitFor(() => {
      expect(listMocks.navigate).toHaveBeenCalledWith({
        to: "/projects/$projectId",
        params: { projectId: "P-7" },
      });
    });
    expect(listMocks.createProject).toHaveBeenCalledWith({
      slug: "delivery",
      title: "Delivery platform",
      description: "",
    });
  });

  it("opens the create dialog and closes it on cancel", () => {
    listMocks.state.isPending = false;
    listMocks.state.data = makeSnapshot({
      projects: [makeProject()],
      features: [],
    });
    render(<ProjectListPage />);

    fireEvent.click(screen.getByRole("button", { name: "New project" }));
    expect(
      screen.getByRole("form", { name: "Create project" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("form", { name: "Create project" }),
    ).not.toBeInTheDocument();
  });
});
