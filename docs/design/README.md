# PRX の設計方針

このディレクトリの各文書は、それぞれ独立した読み単位である。
どの文書を読むべきかは、リポジトリ直下の `AGENTS.md` が変更内容ごとに示す。

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
