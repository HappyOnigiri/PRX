# 常駐サーバの方針

`prx serve` を常駐させる目的は、ログイン後に何も打たずに WebUI へ到達できることである。
対象は macOS だけで、常駐の仕組みは LaunchAgent が所有する。他の OS では常駐を操作する `prx daemon` のサブコマンドと `prx open` が `daemon_unsupported` を返し、`prx serve` を直接使う。
状態を表示する `prx daemon` は他の OS でも成功する。稼働記録は flock だけに依存するので、`prx serve` の稼働は OS を問わず報告できる事実である。plist に由来するフィールドだけが空になる。

## 起動の所有者は launchd である

PRX 自身が daemon プロセスを spawn することはない。起動する権限は常に launchd に渡す。
`prx daemon start` と `prx daemon restart` は launchd へ依頼するだけで、依頼の成功は稼働の証明にならない。どちらも稼働記録が書かれるまで待って結果を報告する。
`prx daemon install` も同じである。plist は `RunAtLoad` なので登録は起動を伴い、待たずに成功を返すと直後の `prx open` が未稼働として失敗する。

plist の `ProgramArguments` は実行ファイルと `serve` の 2 要素だけである。
`--addr` も `--demo` も含めないことが、loopback 外への公開と demo を常駐から締め出す構造的な保証になる。前景で動く入口を二重化しない。`prx serve` はすでに SIGTERM で graceful shutdown する前景サーバである。

`KeepAlive` は `SuccessfulExit: false` で、正常終了を再起動の対象にしない。
`ThrottleInterval` は 10 にする。`server.port` を固定してそのポートが使用中だと serve は起動に失敗し、間隔が短いと launchd が秒単位で再起動してログが際限なく育つ。

`EnvironmentVariables` の `HOME` と `PATH` は必須である。
PRX のパス解決は `HOME` に依存し、`gh` と Keychain のヘルパーは `PATH` に依存する。launchd が渡す環境は極小なので、これを書かないと GitHub 認証が常駐サーバでだけ失敗するサイレントな差分になる。

## plist は PRX が全体を所有する

plist は PRX が生成した全体をそのまま書き、判定も byte 一致で行う。差分は `prx daemon install` の再実行で修復する。
`prx daemon` の `plist_status` は `current`・`stale`・`unknown` の 3 値で、plist の不在や読み取り不能は `unknown` とする。未導入を陳腐化と誤判定しないためである。
plist が symlink や通常ファイル以外のときは読み書きを拒否する。そうしないと install が任意のファイルを 0600 で上書きし得る。
書き込みは一時ファイルへ書いてから rename する。部分的に書かれた plist を launchd が読むと、登録されないまま install が成功してしまう。
ログのローテーションは実装していない既知の負債である。

## 多重起動防止と稼働発見の真実は 1 つの flock である

稼働記録は `<設定ディレクトリ>/prx/run/serve.json` に置き、`PRX_RUN_DIR` で差し替えられる。
このファイルは flock と記録を兼ねる。読み手は共有ロックを試し、取れたら未稼働（残っている内容は stale）、取れなかったら稼働中で内容が有効と判定する。
これにより「アドレスを書いた本人が今も生きている」ことがロック 1 つで保証され、pid の再利用もポートの再利用も誤判定しない。異常終了で内容が残っても、ロックの不在で識別できる。

ロックの取得はデータベースを開く前に行う。排他ロックは短い間隔で数回試してから諦める。読み手が一瞬だけ取る共有ロックと衝突しただけで退くと、サーバが 1 つも残らないまま launchd も再起動しない。
ロックを取れなかった `serve` は stderr にその旨を書いて終了コード 0 で終わる。
非 0 で終わると `KeepAlive{SuccessfulExit: false}` の下で launchd が再起動ループに入る。望んだ状態、すなわちサーバが 1 つ稼働していることは満たされているので、これは失敗ではない。

記録の書き換えは in-place で行い、rename は使わない。rename は新しい inode を作るので、保持中のロックは古い inode に残り、読み手は新しいファイルにロックを取れて未稼働と誤判定する。設定ファイルとは正反対の扱いになる。
正常終了時は内容を空にするだけで unlink はしない。unlink とほぼ同時に新しいインスタンスが同じパスを作ってロックしていると、それを消してしまう。

ロックと記録を伴うのは通常の `prx serve` だけである。`--addr` を指定した起動と `--demo` はアドホックなインスタンスで、稼働中サーバを名乗らず、記録も書かない。
`prx open` は記録が持つ実アドレスを開く。ポートを推測して dial することはない。異常終了で残った番号を開くと、たまたまその番号を掴んだ別プロセスに繋がる。

## 停止は SIGTERM である

`prx daemon stop` は記録された pid へ SIGTERM を送る。`launchctl bootout` は使わない。登録が次のログインまで消えてしまうからである。
pid の信頼性はロックが保証する。ロックを保持したまま生きているプロセスだけがこの記録を書ける。
対象がいなければ、失敗させずに停止済みとして報告する。

## バイナリ置換は自動で反映する

常駐サーバは起動時に自分の実行ファイルの inode・サイズ・mtime を記録し、周期的に検査する。
不一致を検出したら launchd 管理下でのみ自身の再起動を依頼する。管理外のプロセスが自己 kickstart すると、置換側がロックを取れずに終了し、手動で起動したプロセスだけが残る。
これを入れないと、新しい CLI がデータベースを移行した後も古いサーバが古い埋め込みスキーマで応答し続けるサイレントな版ずれが残る。
再起動を依頼する launchctl の context はサーバの context から派生させない。`launchctl kickstart -k` は自身へ SIGTERM を送るので、依頼の完了前に取り消されてしまう。

## CLI は常駐を経由しない

CLI コマンドは常駐サーバを経由せず、従来どおり直接 SQLite を開く。常駐は長寿命の `prx serve` にすぎない。
`prx daemon` と `prx open` はデータベースも設定も開かない。どちらも launchd の登録と稼働記録だけを見る。
