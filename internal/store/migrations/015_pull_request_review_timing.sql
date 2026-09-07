-- 「レビュー中」ステータスの導出には、レビュー依頼が未応答のまま残っているか、
-- 最新レビューの後に push があったかが要る。畳み込み済みの review_state からは
-- どちらも判別できないため、pull_requests に 3 列を足す。
-- docs/design/github-sync.md を参照。
ALTER TABLE pull_requests ADD COLUMN review_request_pending INTEGER NOT NULL DEFAULT 0 CHECK(review_request_pending IN (0,1));
ALTER TABLE pull_requests ADD COLUMN changes_requested_at TEXT;
ALTER TABLE pull_requests ADD COLUMN last_pushed_at TEXT;
