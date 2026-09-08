# Repository instructions

PRX は、タスクと GitHub プルリクエストの依存グラフを扱うローカルファーストのツールである。React UI を埋め込んだ Go の CLI / サーバーと SQLite ストレージとして提供する。

- `docs/` に記載した CLI・JSON・状態の振る舞いは公開契約として扱う。変更するときは実装・テスト・ドキュメントを同時に更新する。
- `README.md` は並行ブランチが衝突しにくいよう最小限に保つ。ビルド・起動・開発に必要な内容とドキュメントへのリンクだけを置く。
- 恒久的な設計方針と背景は `docs/design/` に、自明でない検証方針は `docs/development.md` に、生成された CLI リファレンスは `docs/cli/` に置く。
- `README.md` は、変更によって記載内容が誤りになるときだけ編集する。
- RPC のスキーマと振る舞いは後方互換性なしに変更してよい。
- RPC の変更は Protocol Buffer の定義、サーバー、リポジトリ内のクライアント、テストにすべて反映する。
- 記録済みの公開契約や方針が変わるときは、同じ変更でドキュメントも更新する。
- 変更が触れる設計ドキュメントだけを読む。ディレクトリ全体を読む必要はない。
- Protocol Buffer・マイグレーション・SQL のソースを編集したら `make generate` を実行する。
- `gen/`、`internal/db/`、`web/src/gen/` 配下の生成ファイルは手で編集しない。
- `internal/webui/dist/` はビルド成果物である。`.gitkeep` だけを追跡し、資産は `make build` または `make web-build` で生成する。

## 記載言語

- コメント、`AGENTS.md` などのエージェント向け指示、`docs/` 配下のドキュメントは日本語で書く。既存の英語記述を編集するときも日本語に置き換える。
- 識別子、型名、CLI のコマンド名やフラグ、JSON キー、設定キーは原語のまま残す。
- CLI のヘルプ文言・標準出力・エラーメッセージは英語のままにする。`docs/cli/` の生成物と一致させるため、翻訳しない。
- WebUI の表示文字列は `web/src/i18n/` の en と ja の両方を維持する。
- `README.md` は英語、`README.ja.md` は日本語で、内容を同期する。片方を変更したらもう片方も更新する。

## Comments

コメントは 1 つあたり 3 行以内、1 行あたり 200 表示カラム以内とする。どちらの上限も Go と、`web/` 配下で ESLint が読むすべてのファイルで強制される。

- 隣接するコメント行は 1 つのコメントとして数える。空行を挟むか、説明対象のコードの隣へ段落を移すと分割される。
- ツールのディレクティブ（`//go:*`、`//nolint`、`eslint-disable*`、`@ts-*` など）はどちらの上限にも数えない。
- 3 行に収まらない背景説明は対応する `docs/design/` のドキュメントへ移し、コメントにはその参照だけを残す。
- 長いコメントを残したいときは、同じコメント内の独立した行に `commentlint:allow-long -- <理由>` を置く。これは行の長さの上限だけを免除し、1 つのコメントにマーカーは 1 つだけ置ける。

## Design documents

| ドキュメント | 変更前に読む対象 |
|---|---|
| `docs/design/README.md` | プロダクトの方向性、およびこのディレクトリに記録がない詳細をどのソースが持つか |
| `docs/design/architecture.md` | レイヤ構成、アダプタの責務、RPC 境界を越えるもの |
| `docs/design/cli-contract.md` | CLI コマンドの形、出力モード、JSON スキーマ、識別子、変更操作のルール |
| `docs/design/daemon.md` | LaunchAgent での常駐、多重起動防止と稼働発見、`prx daemon` と `prx open` |
| `docs/design/diagnostics.md` | `prx debug` とその RPC |
| `docs/design/agent-prompts.md` | エージェントのプロンプトテンプレート、その語彙、テンプレートの選択 |
| `docs/design/domain.md` | 表示状態の導出、ステータスの意味、依存関係、プロジェクトの所属 |
| `docs/design/archive.md` | アーカイブ済みのプロジェクトやフィーチャー、およびそれらが禁止する書き込み |
| `docs/design/persistence.md` | ストレージ、設定ファイル、デモモード、ドキュメント、実装計画 |
| `docs/design/github-sync.md` | プルリクエストの同一性、同期の範囲、スケジューリング、失敗時の扱い |
| `docs/design/github-credentials.md` | 認証情報の解決、フォールバック、シークレットの扱い |
| `docs/design/security.md` | ローカルの信頼境界、サーバーの公開範囲、ローカルファイルへのアクセス |
| `docs/design/webui.md` | WebUI の構造、表示状態、アクセシビリティのルール |

## Git workflow

- コミットとプッシュにユーザーの承認は不要である。
- 既存のプルリクエストを更新するときは、変更をコミットしてプッシュする。

## Settings storage

- CLI の振る舞いに影響する設定は、CLI から参照できる設定ファイルに保存する。この設定ファイルはまだ存在しない。そうした設定を実装するときに導入する。
- WebUI の表示だけに影響する設定は、ブラウザの Local Storage に保存する。
