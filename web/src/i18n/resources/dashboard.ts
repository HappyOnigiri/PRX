export const dashboard = {
  en: {
    dashboard: {
      loadingTitle: "Mapping dependencies…",
      loadingDetail: "Reading the local graph and latest GitHub state.",
      errorTitle: "The roadmap could not be loaded",
      noData: "No data returned.",
      titleStart: "What can move ",
      titleEmphasis: "now?",
      syncNow: "Sync GitHub",
      syncingNow: "Syncing GitHub…",
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
        syncError: {
          title: "Sync errors",
          detail: "Needs GitHub attention",
        },
      },
    },
    archived: {
      loadingTitle: "Loading archived features…",
      loadingDetail: "Reading feature history from the local database.",
      errorTitle: "Archived features could not be loaded",
      eyebrow: "Feature history",
      title: "Archived features",
      description:
        "Review completed graphs, restore them to active work, or delete them permanently.",
      featureCount: "archived features",
      listLabel: "Archived feature list",
      emptyTitle: "No archived features",
      emptyDetail:
        "Features you archive will appear here without affecting active queues.",
      progress: "{{merged}}/{{total}} merged",
    },
  },
  ja: {
    dashboard: {
      loadingTitle: "依存関係を読み込んでいます…",
      loadingDetail: "ローカルグラフと最新の GitHub 状態を確認しています。",
      errorTitle: "ロードマップを読み込めませんでした",
      noData: "データが返されませんでした。",
      titleStart: "いま動かせるタスクは",
      titleEmphasis: "？",
      syncNow: "GitHub と同期",
      syncingNow: "GitHub と同期中…",
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
        syncError: {
          title: "同期エラー",
          detail: "GitHub の対応が必要",
        },
      },
    },
    archived: {
      loadingTitle: "アーカイブ済みフィーチャーを読み込んでいます…",
      loadingDetail:
        "ローカルデータベースからフィーチャー履歴を確認しています。",
      errorTitle: "アーカイブ済みフィーチャーを読み込めませんでした",
      eyebrow: "フィーチャー履歴",
      title: "アーカイブ済み",
      description:
        "完了したグラフを参照し、進行中へ復元するか完全に削除できます。",
      featureCount: "件のアーカイブ",
      listLabel: "アーカイブ済みフィーチャー一覧",
      emptyTitle: "アーカイブ済みフィーチャーはありません",
      emptyDetail:
        "アーカイブしたフィーチャーは、進行中のキューに影響せずここへ表示されます。",
      progress: "{{merged}}/{{total}} マージ済み",
    },
  },
} as const;
