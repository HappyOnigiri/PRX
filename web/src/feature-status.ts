import { FeatureStatus, type Feature } from "./gen/prx/v1/prx_pb";

// 概要・サイドバー・task キューはいずれも進行中の作業を示すので、read-only と
// 完了済みの feature は除く。判定はどれも archived ではなくサーバーの readOnly
// を見る。docs/design/webui.md を参照。
export function isActiveFeature(feature: Feature): boolean {
  return !feature.readOnly && feature.displayStatus !== FeatureStatus.COMPLETED;
}

// read-only かつ完了済みの feature はアーカイブ扱いなので、完了リストには作業
// 対象に残っているものだけを含める。
export function isCompletedFeature(feature: Feature): boolean {
  return !feature.readOnly && feature.displayStatus === FeatureStatus.COMPLETED;
}

export function isArchivedFeature(feature: Feature): boolean {
  return feature.readOnly;
}

// unfinishedTaskCount は、自動完了ルールが未完了とみなす task の数を、サーバー
// の集計値を使って返す。
export function unfinishedTaskCount(feature: Feature): number {
  return feature.taskCount - feature.finishedCount;
}
