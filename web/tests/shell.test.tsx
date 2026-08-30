import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setDisplayLanguage } from "../src/i18n";
import { AppShell } from "../src/shell";
import { appVersion } from "../src/version";
import { makeFeature, makeSnapshot } from "./factories";

const shellMocks = vi.hoisted(() => ({
  navigate: vi.fn().mockResolvedValue(undefined),
  mutation: {
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
}));

const snapshot = makeSnapshot({
  features: [
    makeFeature({ id: "active", title: "Active feature", readyCount: 1 }),
    makeFeature({
      id: "conflict",
      title: "Conflict feature",
      conflictCount: 1,
      readyCount: 0,
    }),
    makeFeature({ id: "archived", title: "Archived feature", archived: true }),
  ],
});

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    className,
  }: {
    children: ReactNode;
    className?: string;
  }) => <span className={className}>{children}</span>,
  useNavigate: () => shellMocks.navigate,
}));
vi.mock("../src/api", () => ({
  mutations: { createFeature: vi.fn() },
  configMutations: {
    addHost: vi.fn(),
    updateHost: vi.fn(),
    deleteHost: vi.fn(),
    addAuth: vi.fn(),
    updateAuth: vi.fn(),
    deleteAuth: vi.fn(),
    reorderAuth: vi.fn(),
  },
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => ({ data: snapshot, isError: false }),
  useDomainMutation: () => shellMocks.mutation,
  useConfig: () => ({ data: { hosts: [], authMethods: [] }, isPending: false }),
  useConfigMutation: () => shellMocks.mutation,
}));

describe("AppShell", () => {
  afterEach(cleanup);
  beforeEach(async () => {
    localStorage.clear();
    await setDisplayLanguage("en");
    shellMocks.navigate.mockClear();
    shellMocks.mutation.mutateAsync.mockReset();
    shellMocks.mutation.mutateAsync.mockResolvedValue({
      feature: makeFeature({ id: "created" }),
    });
    shellMocks.mutation.isPending = false;
    shellMocks.mutation.error = null;
  });

  it("shows active features and persists language and theme selections", async () => {
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );

    expect(screen.getByText("Active feature")).toBeInTheDocument();
    expect(screen.getByText("Conflict feature")).toBeInTheDocument();
    expect(screen.queryByText("Archived feature")).not.toBeInTheDocument();
    expect(screen.getByText("Workspace")).toBeInTheDocument();
    expect(screen.getByText(`v${appVersion()}`)).toBeInTheDocument();

    fireEvent.change(
      screen.getByRole("combobox", { name: "Display language" }),
      {
        target: { value: "ja" },
      },
    );
    await waitFor(() => {
      expect(document.documentElement.lang).toBe("ja");
    });
    fireEvent.change(screen.getByRole("combobox", { name: "表示テーマ" }), {
      target: { value: "dark" },
    });
    expect(document.documentElement.dataset["theme"]).toBe("dark");
  });

  it("creates a feature, navigates to it, and supports cancellation", async () => {
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    fireEvent.click(screen.getByRole("button", { name: "New feature" }));
    fireEvent.change(screen.getByLabelText("Slug"), {
      target: { value: "release" },
    });
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Release" },
    });
    fireEvent.change(screen.getByLabelText("Description"), {
      target: { value: "Ship it" },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Create feature" }));

    await waitFor(() => {
      expect(shellMocks.navigate).toHaveBeenCalledOnce();
    });
    expect(shellMocks.mutation.mutateAsync).toHaveBeenCalledWith({
      slug: "release",
      title: "Release",
      description: "Ship it",
    });
    expect(shellMocks.navigate).toHaveBeenCalledWith({
      to: "/features/$featureId",
      params: { featureId: "created" },
    });

    fireEvent.click(screen.getByRole("button", { name: "New feature" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("form", { name: "Create feature" }),
    ).not.toBeInTheDocument();
  });

  it("opens and closes server settings from the rail", () => {
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Server settings" }));
    expect(
      screen.getByRole("dialog", { name: "Server settings" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Done" }));
    expect(
      screen.queryByRole("dialog", { name: "Server settings" }),
    ).not.toBeInTheDocument();
  });
});
