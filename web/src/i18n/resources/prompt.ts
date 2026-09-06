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
      restoreDefaults: "Restore built-in templates",
      saved: "Prompt templates saved.",
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
      restoreDefaults: "既定のテンプレートに戻す",
      saved: "プロンプトテンプレートを保存しました。",
    },
  },
} as const;
