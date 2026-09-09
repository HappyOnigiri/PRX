# PRX

[English](README.md) | 日本語 | [简体中文](README.zh-CN.md)

**PRX** は、多数の GitHub プルリクエストに分かれた施策の進行を、手元で見通せるようにするツールです。
タスクどうしの依存関係を登録しておくだけで、いま着手できるタスクと、何かを待っているタスクを自動で切り分けます。

## 特長

- **着手できるタスクが分かる** — 依存元がすべて片付いたタスクだけを取り出せるので、着手順を毎回考え直す必要がありません。
- **止まっている箇所が分かる** — 施策全体を図で見渡し、どのタスクが何を待っているかをたどれます。レビュー待ちや放置されたタスクも一覧できます。
- **プルリクエストの状態が反映される** — レビュー中、コンフリクト、マージ済みといった状態を GitHub から取得し、タスクの進行に反映します。
- **エージェントに渡す指示文を作れる** — タスクの内容と依存関係をまとめたプロンプトを、テンプレートから組み立ててコピーできます。
- **手元で完結する** — データは自分のマシンに保存され、サーバーは既定でローカルからのアクセスだけを受け付けます。

## インストール

### macOS

Apple Silicon 向けのバイナリを配布しています。

```sh
curl -fsSL https://github.com/HappyOnigiri/PRX/releases/latest/download/install.sh | bash
```

更新も同じコマンドで行えます。

### Linux / WSL2

ソースからビルドします。macOS でも同じ手順が使えます。Windows ネイティブには対応していません。

```sh
git clone https://github.com/HappyOnigiri/PRX.git
cd PRX
make install
```

`make install` は WebUI を含めてビルドし、`~/.local/bin/prx` に配置します。別の場所に入れる場合は `INSTALL_DIR` を指定してください。
常駐を扱う `prx daemon` と `prx open` は macOS だけの機能なので、サーバーは `prx serve` で起動します。

## 使い方

ブラウザからは http://localhost:7331/ を開きます。ポートが使用中で別のポートに移っていても、`prx open` なら実際に待ち受けているアドレスを開きます。

同じデータは `prx` コマンドからも取得・登録できます。AI エージェントに現在の状況を読ませたり、タスクや依存関係を登録させたりできます。

```sh
prx ready      # 着手できるタスクを取り出す
prx graph F-1  # 施策全体をタスクと依存関係ごと見る
prx prompt T-1 # タスクに渡す指示文を組み立てる
```

コマンドやオプションの詳細は、`prx -h` と `prx <command> -h` を参照してください。

## その他の機能

- **GitHub と同期:** `prx config`、`GITHUB_TOKEN`、`GH_TOKEN`、認証済みの `gh` CLI のいずれかで認証情報を用意します。同期しない場合でも、タスクと依存関係の管理はそのまま使えます。
- **ポートの固定:** `prx config server update PORT`。稼働中のサーバーを新しいポートへ移すには、続けて `prx daemon restart` を実行します。
- **前景での起動:** `prx serve` は、どの OS でも前景でサーバーを起動します。
- **デモ:** `prx serve --demo` は、サンプルデータの入ったデモを起動します。自分のデータには影響しないので、まず触ってみたいときに使えます。

## アンインストール

```sh
curl -fsSL https://github.com/HappyOnigiri/PRX/releases/latest/download/uninstall.sh | bash
```

## 開発

```sh
make dev  # 開発サーバーを起動する: http://127.0.0.1:7331
make demo # 同じものを、隔離されたデモデータで起動する
make ci   # 変更を引き渡す前のチェック一式を実行する
```

`make demo` は Go の変更で API を再起動するため、そのたびにデモデータを作り直します。

## ドキュメント

- [docs/cli/prx.md](docs/cli/prx.md): CLI リファレンスの Markdown 版です。
- [docs/design/](docs/design/README.md): 設計上の判断と、その理由をまとめています。
- [docs/development.md](docs/development.md): 検証とリリースのルールをまとめています。

## コントリビュート

コントリビュートを歓迎します！
不具合報告やアイデアは [Issues](https://github.com/HappyOnigiri/PRX/issues) へ、改善は [Pull Request](https://github.com/HappyOnigiri/PRX/pulls) でお寄せください。
ドキュメントの改善や翻訳も歓迎です。
