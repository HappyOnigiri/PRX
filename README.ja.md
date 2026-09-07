# PRX

[English version is README.md](README.md)

PRX は、多数の GitHub プルリクエストにまたがる施策のためのローカルファーストな依存関係コントロールルームである。
正規化した DAG を SQLite に保存し、次に着手して安全なタスクを導出する。
プルリクエストの状態は GitHub から直接取得して更新する。
非対話型の CLI と、ConnectRPC を用いた React ワークスペースは、同じアプリケーションのルールを公開する。

## 必要なもの

- Go 1.27 以降
- `.tool-versions` で定義された Node.js のバージョン
- ルートの `package.json` の `packageManager` で定義された pnpm のバージョン
- Playwright による開発時チェックのための Chromium ブラウザ
- 実際の同期を行う場合、`config.yaml`、`GITHUB_TOKEN`、`GH_TOKEN`、または認証済みの `gh` CLI のいずれかにある GitHub の認証情報

`sqlc`、Buf、Protocol Buffer のジェネレーターは Go のツール依存として固定している。
`make lint` は `.tool-versions` に記載したリリースの `golangci-lint` を `bin/` へインストールする。
これらのツールをグローバルにインストールする必要はない。
配布されるバイナリは、トークンを与えれば Node.js・pnpm・`gh` を必要としない。

## ビルドと起動

```sh
make install
prx serve
```

`make install` はロック済みの web 依存をインストールし、`<version>-dev` として識別されるローカルバイナリをビルドする。
バイナリは `~/.local/bin/prx` にインストールされる。
別の場所へインストールするには `INSTALL_DIR` を設定する。
インストール先が `PATH` に含まれていることを確認し、<http://127.0.0.1:7331> を開く。
本番用の web ビルドはバイナリに埋め込まれているため、フロントエンドのプロセスを別に起動する必要はない。

`prx serve --demo` は、以下のデータベースと設定を無視して再起動時にリセットされる一時的なデモを起動する。

デフォルトのデータベースは、OS のユーザー設定ディレクトリ配下に保存される。
別のデータベースを使うには `--db /path/to/prx.db` または `PRX_DB` を指定する。
サーバーは `--addr` を明示しない限り `127.0.0.1:7331` にのみバインドする。

## 開発

```sh
make dev
```

本番サーバーと同じ URL である <http://127.0.0.1:7331> を開く。
Vite は WebUI の変更を hot module replacement で反映し、RPC リクエストをポート 7332 の開発用 Go サーバーへプロキシする。
Air は Go のソースが変わるとサーバーを再ビルドして再起動する。
開発サーバーを起動する前に、ポート 7331 を使っている既存の `prx serve` プロセスを停止する。

開発サーバーは `prx serve` と同じ実際の GitHub 設定を使う。
`make dev` の前に、`prx config`、`GITHUB_TOKEN`、`GH_TOKEN`、または認証済みの `gh` CLI で設定しておく。
設定ファイルを選ぶには `--config /path/to/config.yaml` または `PRX_CONFIG` を指定する。

変更を引き渡す前に `make ci` を実行する。

## ドキュメント

- `prx -h`、および任意のサブコマンドへの `-h` が CLI を説明する。
- [docs/cli/prx.md](docs/cli/prx.md) は同じリファレンスを Markdown にしたもので、`make generate` が生成する。
- [docs/design/](docs/design/README.md) は設計上の決定、契約、運用上の境界、トレードオフを、関心ごとに 1 ドキュメントで扱う。
- [docs/development.md](docs/development.md) は、ビルド定義からは読み取れない検証とリリースのルールを扱う。
