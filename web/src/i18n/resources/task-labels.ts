export const taskLabels = {
  en: {
    taskLabels: {
      loading: "Loading task labels…",
      statuses: "Statuses",
      blocks: "Blocking labels",
      text: "Text",
      color: "Color",
      inherit: "Inherited",
      inheritText: "Inherit text",
      inheritColor: "Inherit color",
      effective: "Effective: {{text}}",
      source: "Text: {{text}} · Color: {{color}}",
      sourceNames: {
        scope: "this scope",
        project: "project",
        global: "global",
        builtIn: "built-in",
      },
      preview: "Preview",
      contrastWarning: "This color may have low contrast in one theme.",
      global: {
        description:
          "Customize the labels used by every project. Leave a field empty to use the built-in value.",
      },
      project: {
        description:
          "Customize labels for this project. Empty fields inherit from global settings.",
      },
      feature: {
        description:
          "Customize labels for this feature. Empty fields inherit from the project and global settings.",
      },
    },
  },
  ja: {
    taskLabels: {
      loading: "タスクラベルを読み込み中…",
      statuses: "ステータス",
      blocks: "ブロックラベル",
      text: "表示文字列",
      color: "色",
      inherit: "継承",
      inheritText: "文字列を継承",
      inheritColor: "色を継承",
      effective: "実効値: {{text}}",
      source: "文字列: {{text}}・色: {{color}}",
      sourceNames: {
        scope: "この scope",
        project: "project",
        global: "global",
        builtIn: "組み込み",
      },
      preview: "プレビュー",
      contrastWarning:
        "この色は一方のテーマでコントラストが低い可能性があります。",
      global: {
        description:
          "すべてのプロジェクトで使うラベルを変更します。空欄にすると組み込み値を使います。",
      },
      project: {
        description:
          "このプロジェクトのラベルを変更します。空欄の項目は global から継承します。",
      },
      feature: {
        description:
          "この feature のラベルを変更します。空欄の項目は project、global の順に継承します。",
      },
    },
  },
} as const;
