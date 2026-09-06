import { useTranslation } from "react-i18next";
import {
  featureTabIds,
  selectFeatureTab,
  type FeatureTabId,
} from "../feature-tabs";
import type { Feature } from "../gen/prx/v1/prx_pb";
import { FeatureListRow } from "./FeatureListRow";
import { TabList, TabPanel } from "./TabList";

// The list receives every feature of its owner and narrows them itself, so the
// counts and the empty wording follow the tab without the page having to
// filter twice.
interface ProjectFeatureListProps {
  features: Feature[];
  tab: FeatureTabId;
  onSelectTab: (tab: FeatureTabId) => void;
  // A page may hold only one of these lists, but the DOM ids still have to be
  // distinct from every other tab strip on the screen.
  idPrefix: string;
}

export function ProjectFeatureList(props: ProjectFeatureListProps) {
  const { t } = useTranslation();
  const tabs = featureTabIds.map((id) => ({
    id,
    label: t(`project.featureTabs.${id}`),
  }));
  return (
    <section className="feature-list" aria-label={t("project.featuresLabel")}>
      <header className="project-section-head">
        <h2>{t("project.featuresTitle")}</h2>
        <span>
          {t("project.featureCount", {
            count: selectFeatureTab(props.features, props.tab).length,
          })}
        </span>
      </header>
      <TabList
        tabs={tabs}
        active={props.tab}
        onSelect={props.onSelectTab}
        idPrefix={props.idPrefix}
        className="workspace-tabs"
        tabClassName="workspace-tab"
        label={t("project.featureTabsLabel")}
      />
      {featureTabIds.map((id) => (
        <TabPanel
          key={id}
          active={id === props.tab}
          className="workspace-tab-panel"
          idPrefix={props.idPrefix}
          tab={id}
        >
          <FeatureTabRows
            features={selectFeatureTab(props.features, id)}
            tab={id}
          />
        </TabPanel>
      ))}
    </section>
  );
}

function FeatureTabRows({
  features,
  tab,
}: {
  features: Feature[];
  tab: FeatureTabId;
}) {
  const { t } = useTranslation();
  if (features.length === 0)
    return (
      <div className="empty compact">
        <h3>{t(`project.emptyFeatures.${tab}.title`)}</h3>
        <p>{t(`project.emptyFeatures.${tab}.detail`)}</p>
      </div>
    );
  return features.map((feature) => (
    <FeatureListRow
      key={feature.id}
      feature={feature}
      progressLabel={t("project.progress", {
        merged: feature.mergedCount,
        total: feature.taskCount,
      })}
    />
  ));
}
