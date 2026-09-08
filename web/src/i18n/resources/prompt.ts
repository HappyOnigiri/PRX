export const prompt = {
  en: {
    promptSettings: {
      description:
        "Templates used for the prompt copied from a task. The CLI prints the same text.",
      loading: "Loading prompt templates…",
      design: "Design prompt",
      designHint: "Used while the task has no implementation plan.",
      implementation: "Implementation prompt",
      implementationHint: "Used once the task has an implementation plan.",
      placeholders:
        "Available placeholders: {{list}}. {{required}} is required.",
      batch: "Batch implementation prompt",
      batchHint:
        "Used for the tasks selected on a feature and copied as one prompt.",
      batchPlaceholders:
        "Available placeholders: {{list}}. {{required}} is required.",
      restoreDefaults: "Restore built-in templates",
    },
    batchPrompt: {
      open: "Copy batch prompt",
      title: "Copy batch implementation prompt",
      description:
        "One prompt covering the tasks you select. It names each task so the agent fetches its instructions from PRX. A dependent task is implemented after the work it waits for, with its pull request stacked on that work. Edit the wording in Settings › Prompts.",
      empty: "No task in this feature is ready to implement.",
      selectAll: "Select all",
      clearAll: "Clear selection",
      includeBlocked: "Include dependent tasks",
      afterTasks: "after {{tasks}}",
      selectedCount: "{{selected}} of {{total}} selected",
      copy: "Copy prompt",
      copied: "Copied a prompt for {{selected}} tasks.",
      failed: "The prompt could not be copied.",
    },
  },
  ja: {
    promptSettings: {
      description:
        "タスクからコピーするプロンプトのテンプレートです。CLIも同じ文面を出力します。",
      loading: "プロンプトテンプレートを読み込んでいます…",
      design: "設計プロンプト",
      designHint: "実装プランがないタスクで使います。",
      implementation: "実装プロンプト",
      implementationHint: "実装プランがあるタスクで使います。",
      placeholders:
        "使用できるプレースホルダー: {{list}}（{{required}} は必須）",
      batch: "一括実装プロンプト",
      batchHint:
        "フィーチャーで選んだタスクを 1 つのプロンプトにまとめるときに使います。",
      batchPlaceholders:
        "使用できるプレースホルダー: {{list}}（{{required}} は必須）",
      restoreDefaults: "既定のテンプレートに戻す",
    },
    batchPrompt: {
      open: "一括プロンプトをコピー",
      title: "一括実装プロンプトをコピー",
      description:
        "選んだタスクをまとめた 1 つのプロンプトです。各タスクの指示はエージェントが PRX から取得します。依存関係があるタスクは依存元の完了後に実装され、その PR は依存元の PR に積まれます。文面は設定 › プロンプトで編集できます。",
      empty: "このフィーチャーに実装へ着手できるタスクはありません。",
      selectAll: "すべて選択",
      clearAll: "選択を解除",
      includeBlocked: "依存タスクも含める",
      afterTasks: "{{tasks}} の後",
      selectedCount: "{{total}} 件中 {{selected}} 件を選択",
      copy: "プロンプトをコピー",
      copied: "{{selected}} 件のタスクのプロンプトをコピーしました。",
      failed: "プロンプトをコピーできませんでした。",
    },
  },
} as const;
