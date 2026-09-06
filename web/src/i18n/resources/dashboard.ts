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
      readyToStart: "Ready to start",
      noTaskTitle: "No task is ready yet",
      noTaskDetail:
        "Create a feature and connect its tasks, or clear an upstream blocker.",
      metaProject: "Project",
      metaFeature: "Feature",
      metaAssignee: "Assignee",
      queues: {
        ready: { title: "Ready now" },
        review: { title: "Review line" },
        conflicts: { title: "Conflicts" },
        syncError: { title: "Sync errors" },
      },
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
      readyToStart: "着手可能",
      noTaskTitle: "着手できるタスクはまだありません",
      noTaskDetail:
        "フィーチャーを作成してタスクを接続するか、上流のブロッカーを解消してください。",
      metaProject: "プロジェクト",
      metaFeature: "フィーチャー",
      metaAssignee: "担当",
      queues: {
        ready: { title: "着手可能" },
        review: { title: "レビュー待ち" },
        conflicts: { title: "コンフリクト" },
        syncError: { title: "同期エラー" },
      },
    },
  },
} as const;
