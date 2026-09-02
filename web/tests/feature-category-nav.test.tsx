import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureStatus } from "../src/gen/prx/v1/prx_pb";
import { webUISettingsKey } from "../src/i18n/settings";
import { router } from "../src/router";
import { makeFeature, makeSnapshot } from "./factories";

const snapshot = makeSnapshot({
  features: [
    makeFeature({ id: "active", title: "Open payments" }),
    makeFeature({
      id: "completed",
      title: "Finished payments",
      displayStatus: FeatureStatus.COMPLETED,
      finishedCount: 1,
      readyCount: 0,
    }),
    makeFeature({
      id: "archived",
      title: "Historical payments",
      archived: true,
      readyCount: 0,
    }),
  ],
});

vi.mock("../src/hooks", () => ({
  useSnapshot: () => ({
    data: snapshot,
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useAutoSync: () => ({
    status: { data: undefined, isError: false },
    checking: false,
    error: null,
  }),
  useDomainMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  }),
}));

// The graph runs ELK in a Web Worker that jsdom does not provide, and the
// workspace is only here to prove that it leaves the selection alone.
vi.mock("../src/views/FeatureGraph", () => ({
  FeatureGraph: () => <div />,
}));

// The rail lists the features of the selected category below the category rows
// themselves, which carry the counts and so are matched by their own class.
function railFeatureTitles(): (string | null)[] {
  return [...document.querySelectorAll(".rail .feature-link span")].map(
    (element) => element.textContent,
  );
}

function selectedCategories(): (string | null)[] {
  return [...document.querySelectorAll(".rail .nav-link[data-active]")].map(
    (element) => element.textContent,
  );
}

function storedCategory(): unknown {
  const value = localStorage.getItem(webUISettingsKey);
  return value === null
    ? undefined
    : (JSON.parse(value) as { featureCategory?: unknown }).featureCategory;
}

// The router is a module singleton whose load only settles once per file, so
// every step runs inside one mounted tree instead of remounting per case.
describe("Feature category navigation", () => {
  afterEach(cleanup);
  beforeEach(async () => {
    localStorage.clear();
    router.update({ history: createMemoryHistory({ initialEntries: ["/"] }) });
    await router.load();
  });

  it("moves one selection with the category routes and holds it elsewhere", async () => {
    render(<RouterProvider router={router} />);

    expect(selectedCategories()).toEqual([expect.stringContaining("Active")]);
    expect(railFeatureTitles()).toEqual(["Open payments"]);

    fireEvent.click(screen.getByRole("link", { name: /Completed features/ }));
    expect(
      await screen.findByRole("heading", { name: "Completed features" }),
    ).toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/completed");
    expect(selectedCategories()).toEqual([
      expect.stringContaining("Completed"),
    ]);
    expect(
      document.querySelector('.rail .nav-link[aria-current="page"]'),
    ).toHaveTextContent("Completed features");
    expect(railFeatureTitles()).toEqual(["Finished payments"]);
    expect(storedCategory()).toBe("completed");

    fireEvent.click(screen.getByRole("link", { name: /Archived features/ }));
    expect(
      await screen.findByRole("heading", { name: "Archived features" }),
    ).toBeInTheDocument();
    expect(selectedCategories()).toEqual([expect.stringContaining("Archived")]);
    expect(railFeatureTitles()).toEqual(["Historical payments"]);
    expect(storedCategory()).toBe("archived");

    fireEvent.click(screen.getByRole("link", { name: /Completed features/ }));
    expect(
      await screen.findByRole("heading", { name: "Completed features" }),
    ).toBeInTheDocument();

    // The overview is a screen, not a category, so it leaves the rail alone.
    fireEvent.click(screen.getByRole("link", { name: "Overview" }));
    expect(
      await screen.findByRole("heading", { name: /What can move/ }),
    ).toBeInTheDocument();
    expect(selectedCategories()).toEqual([
      expect.stringContaining("Completed"),
    ]);
    expect(railFeatureTitles()).toEqual(["Finished payments"]);

    await router.navigate({ to: "/tasks", search: { q: "" } });
    expect(await screen.findByRole("search")).toBeInTheDocument();
    expect(selectedCategories()).toEqual([
      expect.stringContaining("Completed"),
    ]);

    // A workspace holds the selection too, whichever category its feature is in.
    await router.navigate({
      to: "/features/$featureId",
      params: { featureId: "active" },
    });
    expect(
      await screen.findByRole("heading", { name: "Open payments" }),
    ).toBeInTheDocument();
    expect(selectedCategories()).toEqual([
      expect.stringContaining("Completed"),
    ]);
    expect(railFeatureTitles()).toEqual(["Finished payments"]);
    expect(storedCategory()).toBe("completed");
  });
});
