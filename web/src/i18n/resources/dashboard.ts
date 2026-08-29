export const dashboard = {
  en: {
    dashboard: {
      loadingTitle: "Mapping dependencies…",
      loadingDetail: "Reading the local graph and latest GitHub state.",
      errorTitle: "The roadmap could not be loaded",
      noData: "No data returned.",
      eyebrow: "Operations / all repositories",
      titleStart: "What can move ",
      titleEmphasis: "now?",
      description:
        "Every queue is derived from the dependency graph—never manually marked ready.",
      nodesUnderControl: "nodes under control",
      roadmapStatus: "Roadmap status",
      executionQueue: "Execution queue",
      readyToStart: "Ready to start",
      noTaskTitle: "No task is ready yet",
      noTaskDetail:
        "Create a feature and connect its tasks, or clear an upstream blocker.",
      featureTelemetry: "Feature telemetry",
      deliveryLines: "Delivery lines",
      noFeaturesTitle: "No features yet",
      noFeaturesDetail: "Use “New feature” to draw the first delivery circuit.",
      featureReady: "{{count}} ready",
      featureBlocked: "{{count}} blocked",
      steady: "steady",
      queues: {
        ready: { title: "Ready now", detail: "Dependencies cleared" },
        review: { title: "Review line", detail: "Waiting on people" },
        conflicts: { title: "Conflicts", detail: "Needs intervention" },
        stale: { title: "Stale signal", detail: "Refresh GitHub data" },
      },
    },
  },
  ja: {
    dashboard: {
      loadingTitle: "依存関係を読み込んでいます…",
      loadingDetail: "ローカルグラフと最新の GitHub 状態を確認しています。",
      errorTitle: "ロードマップを読み込めませんでした",
      noData: "データが返されませんでした。",
      eyebrow: "オペレーション / 全リポジトリ",
      titleStart: "いま動かせるタスクは",
      titleEmphasis: "？",
      description:
        "各キューは依存関係グラフから自動算出され、手動で着手可能にはできません。",
      nodesUnderControl: "個のノードを管理中",
      roadmapStatus: "ロードマップの状態",
      executionQueue: "実行キュー",
      readyToStart: "着手可能",
      noTaskTitle: "着手できるタスクはまだありません",
      noTaskDetail:
        "フィーチャーを作成してタスクを接続するか、上流のブロッカーを解消してください。",
      featureTelemetry: "フィーチャーの状況",
      deliveryLines: "デリバリーライン",
      noFeaturesTitle: "フィーチャーはまだありません",
      noFeaturesDetail:
        "「フィーチャーを作成」から最初のデリバリーラインを作成できます。",
      featureReady: "{{count}} 件が着手可能",
      featureBlocked: "{{count}} 件がブロック中",
      steady: "安定",
      queues: {
        ready: { title: "着手可能", detail: "依存関係を解消済み" },
        review: { title: "レビュー待ち", detail: "レビュー担当者の対応待ち" },
        conflicts: { title: "コンフリクト", detail: "対応が必要" },
        stale: { title: "情報が古い", detail: "GitHub データの更新が必要" },
      },
    },
  },
} as const;
