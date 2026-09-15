import { common } from "./resources/common";
import { dashboard } from "./resources/dashboard";
import { documentDialog } from "./resources/document-dialog";
import { domain } from "./resources/domain";
import { errors } from "./resources/errors";
import { inspector } from "./resources/inspector";
import { markdownPreview } from "./resources/markdown-preview";
import { project } from "./resources/project";
import { prompt } from "./resources/prompt";
import { shell } from "./resources/shell";
import { taskCard } from "./resources/task-card";
import { taskLabels } from "./resources/task-labels";
import { tasks } from "./resources/tasks";
import { update } from "./resources/update";
import { workspace } from "./resources/workspace";

export const resources = {
  en: {
    translation: {
      ...shell.en,
      ...common.en,
      ...dashboard.en,
      ...documentDialog.en,
      ...workspace.en,
      ...markdownPreview.en,
      ...inspector.en,
      ...domain.en,
      ...errors.en,
      ...tasks.en,
      ...taskCard.en,
      ...taskLabels.en,
      ...project.en,
      ...prompt.en,
      ...update.en,
    },
  },
  ja: {
    translation: {
      ...shell.ja,
      ...common.ja,
      ...dashboard.ja,
      ...documentDialog.ja,
      ...workspace.ja,
      ...markdownPreview.ja,
      ...inspector.ja,
      ...domain.ja,
      ...errors.ja,
      ...tasks.ja,
      ...taskCard.ja,
      ...taskLabels.ja,
      ...project.ja,
      ...prompt.ja,
      ...update.ja,
    },
  },
} as const;
