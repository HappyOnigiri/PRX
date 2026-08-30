export const documentDialog = {
  en: {
    documentDialog: {
      featureTitle: "Add feature reference",
      taskTitle: "Add task reference",
      description:
        "Register a link, a file path on this computer, or Markdown stored in PRX.",
      titleLabel: "Reference title (optional)",
      titlePlaceholder: "Architecture decision",
      tabs: {
        url: "URL",
        localFile: "Local file",
        markdown: "Markdown",
      },
      urlLabel: "Document URL",
      urlPlaceholder: "https://example.com/document",
      pathLabel: "File path",
      pathPlaceholder: "/Users/mika/project/docs/plan.md",
      chooseFile: "Choose file…",
      choosingFile: "Opening file chooser…",
      chooseCanceled:
        "No file was selected. You can choose again or enter a path.",
      markdownLabel: "Markdown content",
      markdownPlaceholder: "# Decision\n\nRecord the context and outcome…",
      implementationPlan: "Use as this task's implementation plan",
      submit: "Add reference",
      submitting: "Adding reference…",
    },
  },
  ja: {
    documentDialog: {
      featureTitle: "フィーチャー資料を追加",
      taskTitle: "タスク資料を追加",
      description:
        "URL、このコンピューター上のファイルパス、または PRX に保存する Markdown を登録します。",
      titleLabel: "資料タイトル（任意）",
      titlePlaceholder: "アーキテクチャ決定記録",
      tabs: {
        url: "URL",
        localFile: "ローカルファイル",
        markdown: "Markdown",
      },
      urlLabel: "資料の URL",
      urlPlaceholder: "https://example.com/document",
      pathLabel: "ファイルパス",
      pathPlaceholder: "/Users/mika/project/docs/plan.md",
      chooseFile: "ファイルを選択…",
      choosingFile: "ファイル選択画面を開いています…",
      chooseCanceled:
        "ファイルは選択されませんでした。もう一度選ぶか、パスを入力できます。",
      markdownLabel: "Markdown 本文",
      markdownPlaceholder: "# 決定\n\n背景と決定内容を記録…",
      implementationPlan: "このタスクの実装プランにする",
      submit: "資料を追加",
      submitting: "資料を追加しています…",
    },
  },
} as const;
