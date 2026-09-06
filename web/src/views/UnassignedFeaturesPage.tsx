import { useNavigate, useSearch } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { featuresWithoutProject } from "../project";
import { ProjectFeatureList } from "./ProjectFeatureList";
import { useProjectSnapshot } from "./useProjectSnapshot";

// The features that belong to no project need a page of their own, because the
// sidebar's unaffiliated row has to lead somewhere and a project page can only
// show its own members. It borrows the project workspace's frame without its
// header, which is all about a project the page does not have.
export function UnassignedFeaturesPage() {
  const { t } = useTranslation();
  const { features: tab } = useSearch({ from: "/projects/unassigned" });
  const navigate = useNavigate();
  const snapshot = useProjectSnapshot();
  if (snapshot.message) return snapshot.message;

  return (
    <div className="workspace">
      <header className="workspace-head">
        <div className="workspace-title">
          <h1>{t("project.unassignedTitle")}</h1>
          <p className="eyebrow">{t("project.unassignedDescription")}</p>
        </div>
      </header>
      <div className="workspace-body project-body">
        <ProjectFeatureList
          features={featuresWithoutProject(snapshot.data.features)}
          tab={tab}
          onSelectTab={(next) => {
            void navigate({
              to: "/projects/unassigned",
              search: { features: next },
            });
          }}
          idPrefix="unassigned-features"
        />
      </div>
    </div>
  );
}
