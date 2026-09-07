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

  it("offers the active projects and no empty membership", () => {
    render(<ProjectSelectField projects={projects} currentProjectId="P-1" />);
    const select = screen.getByLabelText("Project");
    expect(select).toHaveValue("P-1");
    expect(select).not.toHaveTextContent("No project");
    expect(
      screen.getByRole("option", { name: "Delivery platform" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "Retired programme" }),
    ).not.toBeInTheDocument();
  });

  // 非制御の select は defaultValue が options にないと黙って先頭を表示し、
  // 次にフォームを保存したとき feature がアーカイブ済みプロジェクトから
  // 外れてしまう。
  it("keeps the current membership among the options even when archived", () => {
    render(<ProjectSelectField projects={projects} currentProjectId="P-2" />);
    expect(screen.getByLabelText("Project")).toHaveValue("P-2");
    expect(
      screen.getByRole("option", { name: "Retired programme" }),
    ).toBeInTheDocument();
  });
});
