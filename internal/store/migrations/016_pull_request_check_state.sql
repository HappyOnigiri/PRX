-- CI の失敗をブロックラベルとして出すには、pull request の最新コミットに対する
-- ステータスチェックのロールアップが要る。個々のチェックは保持せず 1 値だけを持つ。
-- docs/design/github-sync.md を参照。
ALTER TABLE pull_requests ADD COLUMN check_state TEXT NOT NULL DEFAULT 'unknown' CHECK(check_state IN ('unknown','none','pending','success','failure'));
