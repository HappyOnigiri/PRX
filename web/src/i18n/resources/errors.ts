export const errors = {
  en: {
    error: {
      crossFeatureDependency:
        "Tasks in different features cannot be connected.",
      cycle: "This dependency would create a cycle: {{path}}",
      duplicateDependency: "This dependency already exists.",
      duplicatePullRequest:
        "This pull request is already attached to another task.",
      documentReadFailed:
        "The Markdown file could not be read. Check that the path exists and is readable from the server's working directory.",
      documentTooLarge: "Markdown preview supports files up to 1 MiB.",
      githubAuth: "GitHub authentication failed.",
      invalidDatabase: "The database path is invalid.",
      invalidDocument: "The reference is invalid.",
      invalidDocumentKind: "The reference type is invalid.",
      invalidDocumentUrl:
        "Enter a reference URL that starts with http:// or https://.",
      invalidKind: "The task type is invalid.",
      invalidParent: "Choose either a feature or a task for this reference.",
      invalidPullRequestUrl: "Enter a github.com pull request URL.",
      invalidSeed: "The demo seed is invalid.",
      invalidSlug: "Use lowercase letters, numbers, and hyphens for the slug.",
      invalidStatus: "The selected status is invalid.",
      invalidTitle: "Enter a title.",
      notFound: "The requested item was not found.",
      prTaskCompletesOnMerge:
        "A PR task completes when its pull request is merged.",
      pullRequestOnManualTask:
        "Manual tasks cannot have a pull request. Create a PR task instead.",
      referencesExist: "Remove dependent references before deleting this item.",
    },
  },
  ja: {
    error: {
      crossFeatureDependency: "異なるフィーチャーのタスクは接続できません。",
      cycle: "この依存関係を追加すると循環します: {{path}}",
      duplicateDependency: "この依存関係はすでに存在します。",
      duplicatePullRequest:
        "このプルリクエストは別のタスクに関連付けられています。",
      documentReadFailed:
        "Markdown ファイルを読み込めませんでした。パスが存在し、サーバーの作業ディレクトリから読み取れることを確認してください。",
      documentTooLarge: "1 MiB までの Markdown ファイルを表示できます。",
      githubAuth: "GitHub の認証に失敗しました。",
      invalidDatabase: "データベースのパスが正しくありません。",
      invalidDocument: "参照資料が正しくありません。",
      invalidDocumentKind: "参照資料の種類が正しくありません。",
      invalidDocumentUrl:
        "参照資料の URL は http:// または https:// で始まる必要があります。",
      invalidKind: "タスクの種類が正しくありません。",
      invalidParent:
        "参照先はフィーチャーまたはタスクのどちらか一方を選んでください。",
      invalidPullRequestUrl:
        "github.com のプルリクエスト URL を入力してください。",
      invalidSeed: "デモデータのシードが正しくありません。",
      invalidSlug: "スラッグには英小文字、数字、ハイフンを使用してください。",
      invalidStatus: "選択したステータスは正しくありません。",
      invalidTitle: "タイトルを入力してください。",
      notFound: "指定された項目が見つかりません。",
      prTaskCompletesOnMerge:
        "PR タスクはプルリクエストがマージされると完了します。",
      pullRequestOnManualTask:
        "手動タスクにはプルリクエストを関連付けられません。PR タスクを作成してください。",
      referencesExist: "先に関連する参照を削除してください。",
    },
  },
} as const;
