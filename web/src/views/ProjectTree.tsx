import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";
import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { isActiveFeature } from "../feature-status";
import type { Feature, Project } from "../gen/prx/v1/prx_pb";
import {
  readCollapsedProjects,
  writeCollapsedProjects,
} from "../i18n/settings";
import {
  featuresInProject,
  featuresWithoutProject,
  unassignedProjectKey,
} from "../project";
import { EntityIcon } from "./EntityIcon";

interface TreeRow {
  key: string;
  title: string;
  // The unaffiliated row has no project to link to, so it is the one row
  // without an ID and the one that reaches its own page instead.
  projectId?: string;
  features: Feature[];
}

// The tree presents the working set: the projects still in play and the
// features inside them that are still in flight. Everything else is reached
// through the tabs on the pages the tree links to.
function treeRows(
  projects: Project[],
  features: Feature[],
  unassignedTitle: string,
): TreeRow[] {
  const rows: TreeRow[] = projects.map((project) => ({
    key: project.id,
    title: project.title,
    projectId: project.id,
    features: featuresInProject(features, project.id).filter(isActiveFeature),
  }));
  const unassigned = featuresWithoutProject(features).filter(isActiveFeature);
  if (unassigned.length)
    rows.push({
      key: unassignedProjectKey,
      title: unassignedTitle,
      features: unassigned,
    });
  return rows;
}

export function ProjectTree({
  headingId,
  projects,
  features,
}: {
  headingId: string;
  projects: Project[];
  features: Feature[];
}) {
  const { t } = useTranslation();
  const [collapsed, setCollapsed] = useState(readCollapsedProjects);
  const rows = treeRows(projects, features, t("project.unassignedTitle"));

  function toggle(key: string) {
    // Deriving the next list from the rows on screen is also what keeps a
    // deleted project from lingering in storage: an ID that no longer has a
    // row cannot be carried over. No separate sweep and no extra pass over the
    // project list are needed.
    const next = rows
      .map((row) => row.key)
      .filter((id) =>
        id === key ? !collapsed.includes(id) : collapsed.includes(id),
      );
    setCollapsed(next);
    writeCollapsedProjects(next);
  }

  return (
    <ul className="nav-tree" aria-labelledby={headingId}>
      {rows.map((row) => (
        <ProjectTreeRow
          key={row.key}
          row={row}
          expanded={!collapsed.includes(row.key)}
          onToggle={() => {
            toggle(row.key);
          }}
        />
      ))}
    </ul>
  );
}

function ProjectTreeRow({
  row,
  expanded,
  onToggle,
}: {
  row: TreeRow;
  expanded: boolean;
  onToggle: () => void;
}) {
  const { t } = useTranslation();
  const childrenId = `nav-tree-children-${row.key}`;
  const label = t("nav.toggleProject", { title: row.title });
  return (
    <li>
      <div className="nav-tree-row">
        {row.features.length ? (
          <button
            aria-controls={childrenId}
            aria-expanded={expanded}
            aria-label={label}
            className="nav-tree-toggle"
            onClick={onToggle}
            title={label}
            type="button"
          >
            <ChevronRight aria-hidden="true" focusable="false" size={14} />
          </button>
        ) : (
          <span className="nav-tree-toggle-spacer" />
        )}
        <ProjectRowLink row={row}>
          <EntityIcon kind="project" size={14} />
          <span>{row.title}</span>
          <b>{row.features.length}</b>
        </ProjectRowLink>
      </div>
      {/* The list stays in the DOM while collapsed so that aria-controls keeps
          pointing at something. */}
      <ul className="nav-tree-children" hidden={!expanded} id={childrenId}>
        {row.features.map((feature) => (
          <li key={feature.id}>
            <FeatureRowLink feature={feature} />
          </li>
        ))}
      </ul>
    </li>
  );
}

// The two destinations differ in their parameters, so the link is written
// twice while the row content is written once.
function ProjectRowLink({
  row,
  children,
}: {
  row: TreeRow;
  children: ReactNode;
}) {
  const className = "feature-link project-link";
  const search = { features: "active" } as const;
  return row.projectId === undefined ? (
    <Link
      to="/projects/unassigned"
      search={search}
      className={className}
      activeProps={{ "data-active": true }}
    >
      {children}
    </Link>
  ) : (
    <Link
      to="/projects/$projectId"
      params={{ projectId: row.projectId }}
      search={search}
      className={className}
      activeProps={{ "data-active": true }}
    >
      {children}
    </Link>
  );
}

function FeatureRowLink({ feature }: { feature: Feature }) {
  return (
    <Link
      to="/features/$featureId"
      params={{ featureId: feature.id }}
      className="feature-link"
      activeProps={{ "data-active": true }}
      // The row carries the same glyph a feature gets everywhere else, so its
      // state rides on the icon's color instead of a second mark of its own.
      data-state={
        feature.conflictCount ? "conflict" : feature.readyCount ? "ready" : ""
      }
    >
      <EntityIcon kind="feature" size={14} />
      <span>{feature.title}</span>
      <b>
        {feature.finishedCount}/{feature.taskCount}
      </b>
    </Link>
  );
}
