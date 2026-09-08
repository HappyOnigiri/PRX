import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProjectSelectField } from "../src/views/ProjectSelectField";
import { makeProject } from "./factories";

const projects = [
  makeProject({ id: "P-1", title: "Delivery platform" }),
  makeProject({ id: "P-2", title: "Retired programme", archived: true }),
];

describe("ProjectSelectField", () => {
  afterEach(cleanup);

  it("offers the active projects and no empty membership", () => {
    const onChange = vi.fn();
    render(
      <ProjectSelectField
        projects={projects}
        currentProjectId="P-1"
        value="P-1"
        onChange={onChange}
      />,
    );
    const select = screen.getByLabelText("Project");
    expect(select).toHaveValue("P-1");
    expect(select).not.toHaveTextContent("No project");
    expect(
      screen.getByRole("option", { name: "Delivery platform" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "Retired programme" }),
    ).not.toBeInTheDocument();
    fireEvent.change(select, { target: { value: "P-1" } });
    expect(onChange).toHaveBeenCalledWith("P-1");
  });

  // 候補にない値を選ぶと、次に保存したとき feature がアーカイブ済み
  // プロジェクトから黙って外れてしまう。
  it("keeps the current membership among the options even when archived", () => {
    render(
      <ProjectSelectField
        projects={projects}
        currentProjectId="P-2"
        value="P-2"
        onChange={vi.fn()}
      />,
    );
    expect(screen.getByLabelText("Project")).toHaveValue("P-2");
    expect(
      screen.getByRole("option", { name: "Retired programme" }),
    ).toBeInTheDocument();
  });
});
