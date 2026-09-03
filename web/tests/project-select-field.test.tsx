import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ProjectSelectField } from "../src/views/ProjectSelectField";
import { makeProject } from "./factories";

const projects = [
  makeProject({ id: "P-1", title: "Delivery platform" }),
  makeProject({ id: "P-2", title: "Retired programme", archived: true }),
];

describe("ProjectSelectField", () => {
  afterEach(cleanup);

  it("offers the active projects and no membership by default", () => {
    render(<ProjectSelectField projects={projects} />);
    const select = screen.getByLabelText("Project");
    expect(select).toHaveValue("");
    expect(
      screen.getByRole("option", { name: "Delivery platform" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "Retired programme" }),
    ).not.toBeInTheDocument();
  });

  // An uncontrolled select whose defaultValue is missing from its options
  // silently shows the first one, which would drop the feature out of the
  // archived project the next time the form is saved.
  it("keeps the current membership among the options even when archived", () => {
    render(<ProjectSelectField projects={projects} currentProjectId="P-2" />);
    expect(screen.getByLabelText("Project")).toHaveValue("P-2");
    expect(
      screen.getByRole("option", { name: "Retired programme" }),
    ).toBeInTheDocument();
  });
});
