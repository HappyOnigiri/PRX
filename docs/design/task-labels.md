# task ラベルの階層設定

task の表示ステータスとブロックラベルは、安定した値を保ったまま表示文字列と色を変更できる。
設定キーは `status.not_started`、`status.designing`、`status.designed`、
`status.in_progress`、`status.implemented`、`status.in_review`、
`status.approved`、`status.completed`、`status.merged`、`status.closed`、
`status.unknown` と、`block.dependency_unresolved`、`block.conflict`、
`block.changes_requested`、`block.ci_failed`、`block.unknown` である。

各キーの `text` と `color` は独立した上書きで、`global`、`project`、`feature` の順に解決する。
項目を空文字列で更新すると、その scope の上書きを解除して親へ戻る。どの scope にも値が
なければ組み込み値を使う。feature を別 project へ移しても feature 固有の値は保持され、
継承している項目だけが移動先の値へ追従する。

`text` は前後空白を除去した Unicode コードポイント 32 文字以内で、改行と制御文字を含められない。`color` は `#RRGGBB` のみを受け付け、小文字へ正規化する。カスタム文字列は翻訳せず、表示言語を変更してもそのまま表示する。

WebUI の task カード、グラフノード、インスペクタ、手動ステータス選択肢は snapshot の
feature ごとの実効 appearance を共有する。文字が上書きされているときは全文を 1 つの
ラベルとして表示し、色だけの上書きでは既存の工程と状態の分割表示を保つ。色は文字、
アイコン、枠線へ適用し、背景や依存未解決チップの形は変えない。ライト・ダークのどちらか
でコントラストが 4.5:1 を下回る色は編集画面で警告するが、保存は拒否しない。

CLI の JSON 出力と `status`・`block` の値は安定値を維持する。人間向けの task 一覧・詳細・
グラフだけが実効文字列を表示する。`prx task update --status` は安定値を優先し、対象 task
の実効ラベルには前後空白除去と Unicode 大小文字無視で照合する。ラベル照合の対象は手動
設定可能な `not_started`、`designing`、`in_progress`、`completed`、`closed` の 5 状態だけ
で、複数一致は候補を示す曖昧エラーにする。

global の値は設定 YAML の `task_labels`、project と feature の値はそれぞれの `task_label_overrides_json` 列へ保存する。RPC は raw override と解決済み appearance を分けて返し、部分更新の未指定項目は変更せず、空文字列は解除として扱う。
