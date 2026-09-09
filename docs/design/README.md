# PRX の設計方針

このディレクトリの各文書は、それぞれ独立した読み単位である。
どの文書を読むべきかは、リポジトリ直下の `AGENTS.md` が変更内容ごとに示す。

## 文書一覧

| ドキュメント | 扱う範囲 |
|---|---|
| [architecture.md](architecture.md) | レイヤ構成、アダプタの責務、RPC 境界 |
| [cli-contract.md](cli-contract.md) | CLI コマンドの形、出力モード、JSON スキーマ、識別子、変更操作のルール |
| [daemon.md](daemon.md) | LaunchAgent での常駐、多重起動防止と稼働発見、`prx daemon` と `prx open` |
| [diagnostics.md](diagnostics.md) | `prx debug` とその RPC |
| [agent-prompts.md](agent-prompts.md) | エージェントのプロンプトテンプレート、その語彙、テンプレートの選択 |
| [domain.md](domain.md) | 表示状態の導出、ステータスの意味、依存関係、プロジェクトの所属 |
| [archive.md](archive.md) | アーカイブ済みのプロジェクトやフィーチャー、およびそれらが禁止する書き込み |
| [persistence.md](persistence.md) | ストレージ、設定ファイルと設定の置き場所、デモモード、ドキュメント、実装計画 |
| [github-sync.md](github-sync.md) | プルリクエストの同一性、同期の範囲、スケジューリング、失敗時の扱い |
| [github-credentials.md](github-credentials.md) | 認証情報の解決、フォールバック、シークレットの扱い |
| [security.md](security.md) | ローカルの信頼境界、サーバーの公開範囲、ローカルファイルへのアクセス |
| [webui.md](webui.md) | WebUI の構造、表示状態、アクセシビリティのルール |

## プロダクトの方向性

- リポジトリをまたいで 5〜100 件の pull request を調整するエンジニアを対象とする。
- すべての pull request を開かなくても、次に着手して安全なタスクとその阻害要因が分かるようにする。
- 依存グラフを永続的かつローカル完結に保ち、人にもコーディングエージェントにも使えるものにする。
- グラフの因果関係を第一の関心事とし、汎用的なプロジェクト指標は副次的に扱う。

## 一次情報

| 関心事 | 現在の挙動の一次情報 |
|---|---|
| CLI commands and flags | Cobra の定義と `docs/cli/` の生成リファレンス |
| Domain values and derivation details | `internal/domain`、アプリケーションコード、およびそのテスト |
| RPC fields | `proto/` 配下の Protocol Buffer 定義 |
| Persistence structure | migration とクエリ定義 |
| WebUI components and interactions | `web/src/` とブラウザテスト |

将来の機能はここに backlog として抱えず、それを所有するプランや pull request で管理する。
