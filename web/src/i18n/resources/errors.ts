export const errors = {
  en: {
    error: {
      unknown: "An unknown domain error occurred.",
      archivedReadOnly:
        "Archived projects and features are read-only. Activate it before making changes.",
      crossFeatureDependency:
        "Tasks in different features cannot be connected.",
      cycle: "This dependency would create a cycle: {{path}}",
      duplicateDependency: "This dependency already exists.",
      duplicatePullRequest:
        "This pull request is already attached to another task.",
      duplicateImplementationPlan:
        "This task already has an implementation plan. Edit or remove it first.",
      documentReadFailed:
        "The Markdown file could not be read. Check that the path exists and is readable from the server's working directory.",
      documentTooLarge: "Markdown preview supports files up to 1 MiB.",
      documentNotText: "The local file is not valid UTF-8 text.",
      githubAuth: "GitHub authentication failed.",
      invalidConfig: "The GitHub configuration is invalid.",
      invalidDatabase: "The database path is invalid.",
      invalidDocument: "The reference is invalid.",
      invalidDocumentKind: "The reference type is invalid.",
      invalidDocumentUrl:
        "Enter a reference URL that starts with http:// or https://.",
      invalidImplementationPlan: "Enter a non-empty implementation plan.",
      implementationPlanTooLarge:
        "Implementation plans support content up to 1 MiB.",
      invalidKind: "The task type is invalid.",
      invalidParent: "Choose either a feature or a task for this reference.",
      invalidPullRequestUrl: "Enter a github.com pull request URL.",
      invalidSlug: "Use lowercase letters, numbers, and hyphens for the slug.",
      invalidStatus: "The selected status is invalid.",
      invalidTitle: "Enter a title.",
      notFound: "The requested item was not found.",
      pullRequestOnManualTask:
        "Manual tasks cannot have a pull request. Create a PR task instead.",
      referencesExist: "Remove dependent references before deleting this item.",
    },
  },
  ja: {
    error: {
      unknown: "不明なドメインエラーが発生しました。",
      archivedReadOnly:
        "アーカイブ済みのプロジェクトとフィーチャーは読み取り専用です。変更する前にアクティブに戻してください。",
      crossFeatureDependency: "異なるフィーチャーのタスクは接続できません。",
      cycle: "この依存関係を追加すると循環します: {{path}}",
      duplicateDependency: "この依存関係はすでに存在します。",
      duplicatePullRequest:
        "このプルリクエストは別のタスクに関連付けられています。",
      duplicateImplementationPlan:
        "このタスクには実装プランがすでにあります。先に編集または削除してください。",
      documentReadFailed:
        "Markdown ファイルを読み込めませんでした。パスが存在し、サーバーの作業ディレクトリから読み取れることを確認してください。",
      documentTooLarge: "1 MiB までの Markdown ファイルを表示できます。",
      documentNotText:
        "ローカルファイルを UTF-8 テキストとして読み込めません。",
      githubAuth: "GitHub の認証に失敗しました。",
      invalidConfig: "GitHub の設定が不正です。",
      invalidDatabase: "データベースのパスが正しくありません。",
      invalidDocument: "参照資料が正しくありません。",
      invalidDocumentKind: "参照資料の種類が正しくありません。",
      invalidDocumentUrl:
        "参照資料の URL は http:// または https:// で始まる必要があります。",
      invalidImplementationPlan: "空ではない実装プランを入力してください。",
      implementationPlanTooLarge: "実装プランは 1 MiB 以内で入力してください。",
      invalidKind: "タスクの種類が正しくありません。",
      invalidParent:
        "参照先はフィーチャーまたはタスクのどちらか一方を選んでください。",
      invalidPullRequestUrl:
        "github.com のプルリクエスト URL を入力してください。",
      invalidSlug: "スラッグには英小文字、数字、ハイフンを使用してください。",
      invalidStatus: "選択したステータスは正しくありません。",
      invalidTitle: "タイトルを入力してください。",
      notFound: "指定された項目が見つかりません。",
      pullRequestOnManualTask:
        "手動タスクにはプルリクエストを関連付けられません。PR タスクを作成してください。",
      referencesExist: "先に関連する参照を削除してください。",
    },
  },
} as const;
