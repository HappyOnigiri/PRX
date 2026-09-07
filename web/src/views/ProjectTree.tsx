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
import { featuresInProject } from "../project";
import { EntityIcon } from "./EntityIcon";

interface TreeRow {
  projectId: string;
  title: string;
  features: Feature[];
}

// ツリーは作業中の集合を示す。進行中のプロジェクトと、その中でまだ動いている
// feature だけ。それ以外はツリーのリンク先ページのタブから辿る。
function treeRows(projects: Project[], features: Feature[]): TreeRow[] {
  return projects.map((project) => ({
    projectId: project.id,
    title: project.title,
    features: featuresInProject(features, project.id).filter(isActiveFeature),
  }));
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
  const [collapsed, setCollapsed] = useState(readCollapsedProjects);
  const rows = treeRows(projects, features);

  function toggle(key: string) {
    // 次のリストを画面上の行から導くことで、削除済みプロジェクトがストレージ
    // に残るのも防げる。行のない ID は引き継げないので、別途の掃除は要らない。
    const next = rows
      .map((row) => row.projectId)
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
          key={row.projectId}
          row={row}
          expanded={!collapsed.includes(row.projectId)}
          onToggle={() => {
            toggle(row.projectId);
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
  const childrenId = `nav-tree-children-${row.projectId}`;
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
      {/* 折りたたみ中もリストを DOM に残し、aria-controls の参照先を保つ。 */}
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

function ProjectRowLink({
  row,
  children,
}: {
  row: TreeRow;
  children: ReactNode;
}) {
  return (
    <Link
      to="/projects/$projectId"
      params={{ projectId: row.projectId }}
      search={{ features: "active" } as const}
      className="feature-link project-link"
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
      // 行は feature がどこでも使うのと同じグリフを持つので、状態は別の印では
      // なくアイコンの色に乗せる。
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
