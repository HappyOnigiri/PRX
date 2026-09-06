import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureStatus } from "../src/gen/prx/v1/prx_pb";
import { webUISettingsKey } from "../src/i18n/settings";
import { ProjectTree } from "../src/views/ProjectTree";
import { makeFeature, makeProject } from "./factories";

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    className,
  }: {
    children: ReactNode;
    className?: string;
  }) => (
    <a href="#test" className={className}>
      {children}
    </a>
  ),
}));

const projects = [
  makeProject({ id: "P-1", title: "Delivery platform" }),
  makeProject({ id: "P-2", title: "Search revamp" }),
];

const features = [
  makeFeature({ id: "F-1", title: "Checkout", projectId: "P-1" }),
  makeFeature({
    id: "F-2",
    title: "Legacy checkout",
    projectId: "P-1",
    archived: true,
  }),
  makeFeature({
    id: "F-3",
    title: "Finished indexing",
    projectId: "P-2",
    displayStatus: FeatureStatus.COMPLETED,
  }),
  makeFeature({ id: "F-4", title: "Loose end" }),
];

function tree() {
  return (
    <ProjectTree
      headingId="nav-projects-heading"
      projects={projects}
      features={features}
    />
  );
}

function childList(key: string) {
  return document.getElementById(`nav-tree-children-${key}`);
}

function stored(): unknown {
  const value = localStorage.getItem(webUISettingsKey);
  return value === null
    ? undefined
    : (JSON.parse(value) as { collapsedProjects?: unknown }).collapsedProjects;
}

describe("ProjectTree", () => {
  afterEach(cleanup);
  beforeEach(() => {
    localStorage.clear();
  });

  // The tree is the working set: projects still in play with the features
  // inside them that are still in flight. Everything else lives behind a tab.
  it("shows only the features in flight and folds the rest away", () => {
    render(tree());

    expect(screen.getByText("Checkout")).toBeInTheDocument();
    expect(screen.queryByText("Legacy checkout")).not.toBeInTheDocument();
    expect(screen.queryByText("Finished indexing")).not.toBeInTheDocument();
    expect(screen.getByText("No project")).toBeInTheDocument();
    expect(screen.getByText("Loose end")).toBeInTheDocument();

    // A project with nothing in flight keeps its row but offers no toggle.
    expect(
      screen.queryByRole("button", {
        name: "Expand or collapse Search revamp",
      }),
    ).not.toBeInTheDocument();
    expect(screen.getByText("Search revamp")).toBeInTheDocument();
  });

  it("omits the unaffiliated row when nothing is unaffiliated", () => {
    render(
      <ProjectTree
        headingId="nav-projects-heading"
        projects={projects}
        features={features.filter((feature) => feature.projectId !== "")}
      />,
    );
    expect(screen.queryByText("No project")).not.toBeInTheDocument();
  });

  it("stores only the collapsed rows and restores them on the next mount", () => {
    const { unmount } = render(tree());
    const toggle = screen.getByRole("button", {
      name: "Expand or collapse Delivery platform",
    });

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(childList("P-1")).not.toHaveAttribute("hidden");

    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(childList("P-1")).toHaveAttribute("hidden");
    expect(stored()).toEqual(["P-1"]);

    unmount();
    render(tree());
    expect(
      screen.getByRole("button", {
        name: "Expand or collapse Delivery platform",
      }),
    ).toHaveAttribute("aria-expanded", "false");
    expect(childList("P-1")).toHaveAttribute("hidden");
  });

  // Every write rebuilds the list from the rows on screen, so an ID that no
  // longer has a row cannot survive the next toggle.
  it("drops a stored ID that no longer names a row", () => {
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ collapsedProjects: ["P-1", "P-deleted"] }),
    );
    render(tree());

    fireEvent.click(
      screen.getByRole("button", { name: "Expand or collapse No project" }),
    );
    expect(stored()).toEqual(["P-1", "unassigned"]);
  });
});
