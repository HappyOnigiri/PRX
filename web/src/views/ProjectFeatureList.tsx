import { useTranslation } from "react-i18next";
import {
  featureTabIds,
  selectFeatureTab,
  type FeatureTabId,
} from "../feature-tabs";
import type { Feature } from "../gen/prx/v1/prx_pb";
import { FeatureListRow } from "./FeatureListRow";
import { TabList, TabPanel } from "./TabList";

// リストは所有者の feature をすべて受け取って自分で絞り込む。おかげで件数と
// 空表示の文言がタブに追従し、ページ側で二重に絞り込まずに済む。
interface ProjectFeatureListProps {
  features: Feature[];
  tab: FeatureTabId;
  onSelectTab: (tab: FeatureTabId) => void;
  // 1 ページにこのリストが 1 つしかない場合でも、DOM id は画面上の他のタブ
  // ストリップと区別できなければならない。
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
      {/* タブはリストを直接開く。セクションの名前は所有者のページタイトルが
          示し、件数は行そのものが示す。 */}
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
        finished: feature.finishedCount,
        total: feature.taskCount,
      })}
    />
  ));
}
