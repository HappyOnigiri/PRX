import { common } from "./resources/common";
import { dashboard } from "./resources/dashboard";
import { domain } from "./resources/domain";
import { errors } from "./resources/errors";
import { inspector } from "./resources/inspector";
import { markdownPreview } from "./resources/markdown-preview";
import { shell } from "./resources/shell";
import { tasks } from "./resources/tasks";
import { workspace } from "./resources/workspace";

export const resources = {
  en: {
    translation: {
      ...shell.en,
      ...common.en,
      ...dashboard.en,
      ...workspace.en,
      ...markdownPreview.en,
      ...inspector.en,
      ...domain.en,
      ...errors.en,
      ...tasks.en,
    },
  },
  ja: {
    translation: {
      ...shell.ja,
      ...common.ja,
      ...dashboard.ja,
      ...workspace.ja,
      ...markdownPreview.ja,
      ...inspector.ja,
      ...domain.ja,
      ...errors.ja,
      ...tasks.ja,
    },
  },
} as const;
