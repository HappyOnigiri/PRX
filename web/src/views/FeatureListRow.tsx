import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { Feature } from "../gen/prx/v1/prx_pb";
import { featureStatusLabel, featureStatusToken } from "../i18n/domain";
import { EntityIcon } from "./EntityIcon";
import { StatusBadge } from "./StatusBadge";

interface FeatureListRowProps {
  feature: Feature;
  // 進捗の文言は行が現れるページ側のものなので、呼び出し元で解決し、行は
  // 特定の翻訳名前空間に縛られないようにする。
  progressLabel: string;
}

export function FeatureListRow({
  feature,
  progressLabel,
}: FeatureListRowProps) {
  const { t } = useTranslation();
  return (
    <Link
      to="/features/$featureId"
      params={{ featureId: feature.id }}
      className="feature-list-row"
    >
      {/* タイトル付きの行ではどこでも状態を行頭に置く。読み手は状態の列を
          縦に追える。 */}
      <div className="feature-list-row-title">
        <StatusBadge
          className={`status-${featureStatusToken(feature.displayStatus)}`}
          label={featureStatusLabel(feature.displayStatus, t)}
        />
        <b>
          <EntityIcon kind="feature" size={15} />
          {feature.title}
        </b>
      </div>
      <div className="progress-track" aria-hidden="true">
        <i
          style={{
            width: `${feature.taskCount ? (feature.finishedCount / feature.taskCount) * 100 : 0}%`,
          }}
        />
      </div>
      <span>{progressLabel}</span>
    </Link>
  );
}
