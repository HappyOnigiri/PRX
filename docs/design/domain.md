# ドメイン方針

サーバは [architecture.md](architecture.md) の方針に従い、保存された状態と外部の事実から表示状態を導出する。

task に種別はない。どの task も pull request を持てるし、pull request を持たない task は保存済みステータスによって完了に至る。

task の状態は 2 系統に分かれる。ステータスは 1 つで作業がどこまで進んだかを示し、ブロックラベルは 0〜3 個で進行を妨げている事情を示す。
進行を妨げる事情がステータスを上書きしないので、コンフリクトしている承認済み PR は承認済みのまま、コンフリクトのラベルを併せ持つ。
ブロックラベルは未終了の task すべてで評価する。実装が進んだ task の依存切れも見えるようにするためで、終了した task ではステータスもラベルも決着済みとして扱い、ラベルは付かない。
ラベルを増やしても依存の充足条件は変えない。ready の意味は「未着手系のステータスで、未解決の blocker がない」のままである。
承認済みはレビュー中より優先する。承認は変更要求が 1 件もないことを意味し、別のレビュー依頼が残っていることより読み手にとって重い。

task のステータスは手動で選ぶ。
未完了のステータスは、紐づいた pull request に譲る。そのため pull request を紐づければ、もう一度編集しなくてもその状態が提示される。
designing ステータスはもう一段譲り、登録された implementation plan に道を譲る。設計中と記録された task は、plan が登録された時点で designed として提示される。
完了系のステータスは pull request より優先される。手作業で決着させた task は、pull request が開いたままでも決着済みのままになる。
pull request のない in progress の task は、何も満たさず依存も解除しない。その task が指す作業がどこにも着地していないためである。
designing は作り方を決める段階であって作る段階ではないので、未着手の task と同じ readiness の問いを保ち続ける。
依存の充足判定には、表示上のラベルではなく素の完了状態を使う。
review、conflict、staleness といった表示用のフラグが、完了の定義をひそかに変えることはない。

feature も保存済みステータスと派生ステータスを分け、保存済みステータスに automatic を持つ。
保存済みステータスの既定は automatic で、automatic の feature は task を 1 つ以上持ち、そのすべてが完了した時点で completed として提示される。
automatic 以外の保存済みステータスは人の判断なのでそのまま提示する。作業を再開した feature は、task がすべて完了していても active のままになる。
feature に独自の派生語彙はない。派生値は、automatic を除いた保存済みステータスである。

project は feature の 1 つ上の単位で、feature をまとめ、それらが共有する document を保持する。
feature は作成時も移動後も、必ずちょうど 1 つの project に属する。依存関係は project にかかわらず 1 つの feature の内側に閉じる。
project が持つ状態は archive されているかどうかだけで、feature と task が共有する 2 層のステータス規則からは意図的に外してある。
feature を含むすべての読み取りは `Feature.ReadOnly` を返し、feature 自身かその project のどちらかが archive されていれば true になる。
クライアントはこの値をそのまま使う。
read-only な feature は archived のカテゴリに提示され、active な feature 一覧・overview・task 検索から外れる。これは feature を個別に archive した場合と同じ扱いである。
archive が禁じる書き込みは [archive.md](archive.md) に記録する。

依存関係は blocker から blocked に向かう。
依存関係の変更は、feature の所有関係と DAG の整合性を保つ。
循環の拒否には、呼び出し側が失敗を説明できるだけの文脈を含める。
blocked な task は batch での引き渡しのために未解決の blocker をすべて保持するが、blocked の理由としては、読み手が対処すべき最初の 1 件だけを示す。

現在の状態値、表示の優先順位、readiness の条件は、ドメインの実装とそのテストが所有する。
