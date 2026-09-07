-- feature の格納 status に自動モードを加えるが、SQLite は status の CHECK 制約を
-- その場で拡張できず、features を作り直すと tasks や documents、さらにその子まで
-- 巻き込む。そこで自動かどうかは専用の列で持つ。status_auto = 1 は feature が
-- 表示上の status を配下の task から導くことを意味し、status 列は 'active' に
-- 正規化されて、明示的な status に置き換わるまで意味を持たない。
ALTER TABLE features ADD COLUMN status_auto INTEGER NOT NULL DEFAULT 1 CHECK(status_auto IN (0,1));

-- 既存の paused・completed・cancelled は人が選んだ status なので、手動指定と
-- して残す。active の feature は自動モードに移す。ADD COLUMN は既存行に対して
-- CHECK を評価しないため、この補正は制約を満たすためではなく意味を揃えるもの。
UPDATE features SET status_auto = 0 WHERE status <> 'active';
