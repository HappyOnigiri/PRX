import {
  isActiveFeature,
  isArchivedFeature,
  isCompletedFeature,
} from "./feature-status";
import type { Feature } from "./gen/prx/v1/prx_pb";

// feature 一覧は表示先の project ページでステータス別に絞り込む。タブはブラウザ
// 側の状態ではなく search parameter にしてあり、リロード・履歴・共有リンクで同じ
// 表示を再現できる。task 検索がクエリでそうしているのと同じ。
export const featureTabIds = ["active", "completed", "archived"] as const;
export type FeatureTabId = (typeof featureTabIds)[number];

const predicates: Record<FeatureTabId, (feature: Feature) => boolean> = {
  active: isActiveFeature,
  completed: isCompletedFeature,
  archived: isArchivedFeature,
};

export function selectFeatureTab(
  features: Feature[],
  tab: FeatureTabId,
): Feature[] {
  return features.filter(predicates[tab]);
}

function isFeatureTabId(value: unknown): value is FeatureTabId {
  return featureTabIds.includes(value as FeatureTabId);
}

// 不明な値や未指定はルートを失敗させず作業対象へフォールバックする。手で書き換
// えたリンクや古いリンクでもページが開く。
export function validateFeatureTabSearch(search: Record<string, unknown>): {
  features: FeatureTabId;
} {
  const value = search["features"];
  return { features: isFeatureTabId(value) ? value : "active" };
}
