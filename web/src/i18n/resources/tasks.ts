export const tasks = {
  en: {
    tasks: {
      loadingTitle: "Loading task search…",
      loadingDetail: "Reading active tasks and their latest GitHub state.",
      errorTitle: "Task search could not be loaded",
      noData: "No task data returned.",
      title: "Task search",
      searchLabel: "Search active tasks",
      searchPlaceholder: "task-status:ready or payments",
      searchSubmit: "Search",
      searchHint:
        "Use task-status:ready, block:ci-failed, github-status:error, or a word. Combine conditions with spaces.",
      resultCountLabel: "matching active tasks",
      listLabel: "Matching active tasks",
      emptyTitle: "No matching active tasks",
      emptyDetail:
        "Try a different word or remove one of the status conditions.",
      noPullRequest: "No GitHub pull request",
      invalidSearchTitle: "Search needs a correction",
      invalidQualifier:
        "{{key}} does not accept “{{value}}”. Use a supported status value.",
      unterminatedQuote: "Close the quotation mark before searching.",
    },
  },
  ja: {
    tasks: {
      loadingTitle: "タスク検索を読み込んでいます…",
      loadingDetail: "アクティブなタスクと最新の GitHub 状態を確認しています。",
      errorTitle: "タスク検索を読み込めませんでした",
      noData: "タスクデータが返されませんでした。",
      title: "タスク検索",
      searchLabel: "アクティブなタスクを検索",
      searchPlaceholder: "task-status:ready または payments",
      searchSubmit: "検索",
      searchHint:
        "task-status:ready、block:ci-failed、github-status:error、または語句を使えます。空白で条件を組み合わせます。",
      resultCountLabel: "件の該当タスク",
      listLabel: "該当するアクティブなタスク",
      emptyTitle: "該当するアクティブなタスクはありません",
      emptyDetail: "別の語句を試すか、ステータス条件を減らしてください。",
      noPullRequest: "GitHub プルリクエストなし",
      invalidSearchTitle: "検索条件を修正してください",
      invalidQualifier:
        "{{key}} に「{{value}}」は指定できません。対応する値を使ってください。",
      unterminatedQuote: "引用符を閉じてから検索してください。",
    },
  },
} as const;
