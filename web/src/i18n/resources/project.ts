export const project = {
  en: {
    project: {
      membership: "Project",
      noMembership: "No project",
      title: "Projects",
      listLabel: "Project list",
      loadingTitle: "Loading projects…",
      loadingDetail: "Reading the current snapshot.",
      errorTitle: "Projects are unavailable",
      emptyTitle: "No projects yet",
      emptyDetail:
        "Create a project to group related features and their shared documents.",
      emptyArchivedTitle: "No archived projects",
      emptyArchivedDetail: "Archived projects appear here.",
      tabsLabel: "Project status",
      tabs: { active: "Active", archived: "Archived" },
      featureCount_one: "{{count}} feature",
      featureCount_other: "{{count}} features",
      archivedBadge: "Archived",
      workspaceLoading: "Loading project…",
      notFound: "Project not found",
      returnList: "Return to projects",
      noDescription: "No project description yet.",
      featuresLabel: "Features in this project",
      featureTabsLabel: "Feature status",
      featureTabs: {
        active: "Active",
        completed: "Completed",
        archived: "Archived",
      },
      emptyFeatures: {
        active: {
          title: "No features in flight",
          detail:
            "Assign a feature here from the feature's edit dialog, or return a completed one to active work.",
        },
        completed: {
          title: "No completed features",
          detail:
            "A feature appears here once every one of its tasks is finished, or when you set its status to completed.",
        },
        archived: {
          title: "No archived features",
          detail: "Features you archive appear here without leaving the list.",
        },
      },
      unassignedTitle: "No project",
      progress: "{{finished}}/{{total}} finished",
      editProject: "Edit project",
      manageProject: "Manage project",
      archivedLabel: "Archived · read-only",
      archivedDetail:
        "This project and every feature in it are read-only. Activate the project to edit them.",
    },
    projectCreate: {
      formLabel: "Create project",
      title: "Create project",
      titlePlaceholder: "Delivery platform",
      descriptionPlaceholder: "What do these features have in common?",
      submit: "Create project",
    },
    projectEdit: {
      formLabel: "Edit project",
      title: "Edit project",
      submit: "Save project",
      manageLabel: "Manage archived project",
      manageTitle: "Project details",
      archivedEyebrow: "Archived · read-only",
      lifecycle: "Project lifecycle",
      activeLifecycleDetail:
        "Archive this project to make it and its features read-only, or delete it permanently.",
      archivedLifecycleDetail:
        "Activate this project to let its features accept changes again, or delete it.",
      archive: "Archive project",
      restore: "Activate project",
      delete: "Delete project",
      archiveTitle: "Archive {{title}}?",
      archiveDescription:
        "The project and every feature in it become read-only, and those features leave Overview and Active circuits until it is activated again.",
      confirmArchive: "Archive project",
      deleteTitle: "Delete {{title}}?",
      deleteDescription:
        "This deletes the project and its {{count}} references. Its features are kept and simply leave the project. This cannot be undone.",
      confirmDelete: "Delete permanently",
    },
  },
  ja: {
    project: {
      membership: "プロジェクト",
      noMembership: "プロジェクトなし",
      title: "プロジェクト",
      listLabel: "プロジェクト一覧",
      loadingTitle: "プロジェクトを読み込んでいます…",
      loadingDetail: "現在のスナップショットを読み取っています。",
      errorTitle: "プロジェクトを取得できません",
      emptyTitle: "プロジェクトはまだありません",
      emptyDetail:
        "関連するフィーチャーと共通資料をまとめるプロジェクトを作成します。",
      emptyArchivedTitle: "アーカイブ済みのプロジェクトはありません",
      emptyArchivedDetail: "アーカイブしたプロジェクトはここに表示されます。",
      tabsLabel: "プロジェクトの状態",
      tabs: { active: "進行中", archived: "アーカイブ済み" },
      featureCount_one: "フィーチャー {{count}} 件",
      featureCount_other: "フィーチャー {{count}} 件",
      archivedBadge: "アーカイブ済み",
      workspaceLoading: "プロジェクトを読み込んでいます…",
      notFound: "プロジェクトが見つかりません",
      returnList: "プロジェクト一覧に戻る",
      noDescription: "プロジェクトの説明はまだありません。",
      featuresLabel: "このプロジェクトのフィーチャー",
      featureTabsLabel: "フィーチャーの状態",
      featureTabs: {
        active: "進行中",
        completed: "完了済み",
        archived: "アーカイブ済み",
      },
      emptyFeatures: {
        active: {
          title: "進行中のフィーチャーはありません",
          detail:
            "フィーチャーの編集ダイアログから所属させるか、完了済みフィーチャーを進行中へ戻します。",
        },
        completed: {
          title: "完了済みフィーチャーはありません",
          detail:
            "タスクがすべて終了したフィーチャー、またはステータスを完了にしたフィーチャーがここへ表示されます。",
        },
        archived: {
          title: "アーカイブ済みフィーチャーはありません",
          detail:
            "アーカイブしたフィーチャーは、一覧から消えずにここへ表示されます。",
        },
      },
      unassignedTitle: "プロジェクトなし",
      progress: "{{finished}}/{{total}} 完了",
      editProject: "プロジェクトを編集",
      manageProject: "プロジェクトを管理",
      archivedLabel: "アーカイブ済み・読み取り専用",
      archivedDetail:
        "このプロジェクトと配下のフィーチャーは読み取り専用です。編集するにはプロジェクトをアクティブに戻してください。",
    },
    projectCreate: {
      formLabel: "プロジェクトを作成",
      title: "プロジェクトを作成",
      titlePlaceholder: "デリバリー基盤",
      descriptionPlaceholder: "これらのフィーチャーに共通することを記述",
      submit: "プロジェクトを作成",
    },
    projectEdit: {
      formLabel: "プロジェクトを編集",
      title: "プロジェクトを編集",
      submit: "プロジェクトを保存",
      manageLabel: "アーカイブ済みプロジェクトを管理",
      manageTitle: "プロジェクトの詳細",
      archivedEyebrow: "アーカイブ済み・読み取り専用",
      lifecycle: "プロジェクトのライフサイクル",
      activeLifecycleDetail:
        "アーカイブするとプロジェクトと配下のフィーチャーが読み取り専用になります。不要な場合は削除します。",
      archivedLifecycleDetail:
        "アクティブに戻すと配下のフィーチャーを再び編集できます。不要な場合は削除します。",
      archive: "プロジェクトをアーカイブ",
      restore: "プロジェクトをアクティブに戻す",
      delete: "プロジェクトを削除",
      archiveTitle: "{{title}} をアーカイブしますか？",
      archiveDescription:
        "プロジェクトと配下のフィーチャーが読み取り専用になり、アクティブに戻すまで概要と進行中のフィーチャーから外れます。",
      confirmArchive: "アーカイブする",
      deleteTitle: "{{title}} を削除しますか？",
      deleteDescription:
        "プロジェクトと {{count}} 件の資料を削除します。配下のフィーチャーは残り、所属だけが外れます。この操作は取り消せません。",
      confirmDelete: "完全に削除",
    },
  },
} as const;
