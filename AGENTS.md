# Repository instructions

PRX は、タスクと GitHub プルリクエストの依存グラフを扱うローカルファーストのツールである。React UI を埋め込んだ Go の CLI / サーバーと SQLite ストレージとして提供する。

- `docs/` に記載した CLI・JSON・状態の振る舞いは公開契約として扱う。変更するときは実装・テスト・ドキュメントを同時に更新する。
- 恒久的な設計方針と背景は `docs/design/` に、自明でない検証方針は `docs/development.md` に、生成された CLI リファレンスは `docs/cli/` に置く。
- `README.md` は並行ブランチが衝突しにくいよう最小限に保ち、記載が誤りになるときだけ編集する。
- RPC のスキーマと振る舞いは後方互換性なしに変更してよい。変更は Protocol Buffer の定義・サーバー・リポジトリ内のクライアント・テストにまとめて反映する。
- Protocol Buffer・マイグレーション・SQL のソースを編集したら `make generate` を実行する。`gen/`、`internal/db/`、`web/src/gen/` 配下の生成物は手で編集しない。
- `internal/webui/dist/` は `make build` または `make web-build` の成果物で、`.gitkeep` だけを追跡する。
- コミットとプッシュにユーザーの承認は不要である。

## 記載言語

- コメント、エージェント向け指示、`docs/` 配下のドキュメントは日本語で書く。既存の英語記述を編集するときも日本語に置き換える。
- 識別子、型名、CLI のコマンド名やフラグ、JSON キー、設定キーは原語のまま残す。
- CLI のヘルプ文言・標準出力・エラーメッセージは英語のままにする。`docs/cli/` の生成物と一致させるため、翻訳しない。
- WebUI の表示文字列は `web/src/i18n/` の en と ja の両方を維持する。
- `README.md` は英語、`README.ja.md` は日本語で、内容を同期する。

## Comments

コメントは 1 つあたり 3 行以内、1 行あたり 200 表示カラム以内とする。Go と `web/` の lint が強制し、隣接するコメント行は 1 つとして数える。`//go:*`、`//nolint`、`eslint-disable*`、`@ts-*` などのツールディレクティブはどちらの上限にも数えない。

- 3 行に収まらない背景説明は `docs/design/` へ移し、コメントには参照だけを残す。空行を挟むか段落を対象コードの隣へ移せば別のコメントに分かれる。
- 長い行を残したいときは、同じコメント内の独立した行に `commentlint:allow-long -- <理由>` を置く。行長の上限だけを免除し、1 つのコメントに 1 つだけ置ける。

## Design documents

変更が触れるものだけを読む。

| ドキュメント | 変更前に読む対象 |
|---|---|
| `docs/design/README.md` | プロダクトの方向性、およびこのディレクトリに記録がない詳細をどのソースが持つか |
| `docs/design/architecture.md` | レイヤ構成、アダプタの責務、RPC 境界を越えるもの |
| `docs/design/cli-contract.md` | CLI コマンドの形、出力モード、JSON スキーマ、識別子、変更操作のルール |
| `docs/design/diagnostics.md` | `prx debug` とその RPC |
| `docs/design/agent-prompts.md` | エージェントのプロンプトテンプレート、その語彙、テンプレートの選択 |
| `docs/design/domain.md` | 表示状態の導出、ステータスの意味、依存関係、プロジェクトの所属 |
| `docs/design/archive.md` | アーカイブ済みのプロジェクトやフィーチャー、およびそれらが禁止する書き込み |
| `docs/design/persistence.md` | ストレージ、設定ファイルと設定の置き場所、デモモード、ドキュメント、実装計画 |
| `docs/design/github-sync.md` | プルリクエストの同一性、同期の範囲、スケジューリング、失敗時の扱い |
| `docs/design/github-credentials.md` | 認証情報の解決、フォールバック、シークレットの扱い |
| `docs/design/security.md` | ローカルの信頼境界、サーバーの公開範囲、ローカルファイルへのアクセス |
| `docs/design/webui.md` | WebUI の構造、表示状態、アクセシビリティのルール |
