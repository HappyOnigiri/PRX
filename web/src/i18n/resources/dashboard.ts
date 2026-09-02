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
    active: {
      loadingTitle: "Loading the working set…",
      loadingDetail: "Reading features still in flight from the local graph.",
      errorTitle: "The working set could not be loaded",
      eyebrow: "Features in flight",
      title: "Active features",
      description:
        "Every feature with unfinished tasks. One leaves this list when its tasks all finish or you archive it.",
      featureCount: "active features",
      listLabel: "Active feature list",
      emptyTitle: "Nothing is in flight",
      emptyDetail:
        "Draw a new feature, or return a completed one to active work, to fill this list.",
      progress: "{{merged}}/{{total}} merged",
    },
    completed: {
      loadingTitle: "Loading completed features…",
      loadingDetail: "Reading finished work from the local database.",
      errorTitle: "Completed features could not be loaded",
      eyebrow: "Finished work",
      title: "Completed features",
      description:
        "Features whose tasks are all finished. Change the status to return one to active work.",
      featureCount: "completed features",
      listLabel: "Completed feature list",
      emptyTitle: "No completed features",
      emptyDetail:
        "A feature appears here once every one of its tasks is finished, or when you set its status to completed.",
      progress: "{{merged}}/{{total}} merged",
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
    active: {
      loadingTitle: "進行中の作業を読み込んでいます…",
      loadingDetail: "ローカルグラフから進行中のフィーチャーを確認しています。",
      errorTitle: "進行中の作業を読み込めませんでした",
      eyebrow: "進行中のフィーチャー",
      title: "進行中",
      description:
        "終了していないタスクが残るフィーチャーです。タスクがすべて終了するかアーカイブすると、この一覧から外れます。",
      featureCount: "件の進行中",
      listLabel: "進行中フィーチャー一覧",
      emptyTitle: "進行中のフィーチャーはありません",
      emptyDetail:
        "フィーチャーを作成するか、完了済みフィーチャーを進行中へ戻すとここへ表示されます。",
      progress: "{{merged}}/{{total}} マージ済み",
    },
    completed: {
      loadingTitle: "完了済みフィーチャーを読み込んでいます…",
      loadingDetail: "ローカルデータベースから完了した作業を確認しています。",
      errorTitle: "完了済みフィーチャーを読み込めませんでした",
      eyebrow: "完了した作業",
      title: "完了済み",
      description:
        "タスクがすべて終了したフィーチャーです。ステータスを変更すると進行中へ戻せます。",
      featureCount: "件の完了",
      listLabel: "完了済みフィーチャー一覧",
      emptyTitle: "完了済みフィーチャーはありません",
      emptyDetail:
        "タスクがすべて終了したフィーチャー、またはステータスを完了にしたフィーチャーがここへ表示されます。",
      progress: "{{merged}}/{{total}} マージ済み",
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
