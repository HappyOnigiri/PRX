package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/HappyOnigiri/PRX/internal/db"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
	// path は Open が解決した位置。プロセスが実際に開いたデータベースを
	// 診断で報告できるよう保持する。
	path string
	now  func() time.Time
}

const (
	taskStatusMigration = 3
	publicIDsMigration  = 5
)

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "prx", "prx.db"), nil
}

func Open(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, fmt.Errorf("resolve database path: %w", err)
		}
	}
	if !strings.HasPrefix(path, "file:") && path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}
	dsn := path
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	// _txlock=immediate はトランザクション開始時に書き込みロックを取る。既定の
	// deferred BEGIN だと、先に読む read-modify-write が書き込み時に
	// SQLITE_BUSY_SNAPSHOT で失敗し、busy_timeout では再試行されない。
	dsn += separator + "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate"
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite は :memory: への接続ごとに専用のデータベースを与えるため、プールの
	// 2 本目の接続は空でマイグレーション未適用のものに繋がってしまう。
	if path == ":memory:" {
		database.SetMaxOpenConns(1)
		database.SetMaxIdleConns(1)
	} else {
		database.SetMaxOpenConns(8)
		database.SetMaxIdleConns(4)
	}
	store := &Store{db: database, path: path, now: func() time.Time { return time.Now().UTC() }}
	if err := store.migrate(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) DB() *sql.DB  { return s.db }

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
	); err != nil {
		return fmt.Errorf("create migration metadata: %w", err)
	}
	if err := s.repairMigrationVersionCollisions(ctx); err != nil {
		return fmt.Errorf("repair migration metadata: %w", err)
	}
	files, err := migrationFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		version := file.version
		var exists int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=?`, version).
			Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		body, err := migrations.ReadFile("migrations/" + file.name)
		if err != nil {
			return err
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(body)); err == nil {
			_, err = tx.ExecContext(
				ctx,
				`INSERT INTO schema_migrations(version,applied_at) VALUES(?,?)`,
				version,
				s.now().Format(time.RFC3339Nano),
			)
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", file.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", file.name, err)
		}
	}
	return nil
}

// repairMigrationVersionCollisions は、かつて 2 つのブランチが別スキーマに使った
// バージョン 2・3 を整合させ、古いビルドが消した記録を復元する。
// docs/design/persistence.md のとおり、記録の是非はスキーマが決める。
func (s *Store) repairMigrationVersionCollisions(ctx context.Context) error {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT version FROM schema_migrations WHERE version IN (2, 3)`,
	)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	versions := make(map[int]bool, 2)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return err
		}
		versions[version] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	hasHost, err := s.pullRequestsHaveHostColumn(ctx)
	if err != nil {
		return err
	}
	hasTaskStatusVocabulary, err := s.tasksUseTaskStatusVocabulary(ctx)
	if err != nil {
		return err
	}
	staleVersions := make([]int, 0, 2)
	if versions[2] && !hasHost {
		staleVersions = append(staleVersions, 2)
	}
	if versions[3] && !hasTaskStatusVocabulary {
		staleVersions = append(staleVersions, 3)
	}
	hasPublicIDs, err := s.tablesHavePublicIDs(ctx)
	if err != nil {
		return err
	}
	var publicIDsRecorded int
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		publicIDsMigration,
	).Scan(&publicIDsRecorded); err != nil {
		return err
	}
	missingVersions := make([]int, 0, 2)
	// 古いバージョン 3 はすぐ上で削除しているので、記録の復元を求めるのは
	// 語彙を保持しているスキーマだけになる。
	if !versions[3] && hasTaskStatusVocabulary {
		missingVersions = append(missingVersions, taskStatusMigration)
	}
	if hasPublicIDs && publicIDsRecorded == 0 {
		missingVersions = append(missingVersions, publicIDsMigration)
	}
	if len(staleVersions) == 0 && len(missingVersions) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, version := range staleVersions {
		if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version=?`, version); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	for _, version := range missingVersions {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
			version,
			s.now().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) tablesHavePublicIDs(ctx context.Context) (bool, error) {
	featureHasPublicID, err := s.tableHasColumn(ctx, "features", "public_id")
	if err != nil {
		return false, err
	}
	taskHasPublicID, err := s.tableHasColumn(ctx, "tasks", "public_id")
	if err != nil {
		return false, err
	}
	return featureHasPublicID && taskHasPublicID, nil
}

func (s *Store) tableHasColumn(ctx context.Context, table, column string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &primaryKey); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *Store) pullRequestsHaveHostColumn(ctx context.Context) (bool, error) {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(pull_requests)`)
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &primaryKey); err != nil {
			return false, err
		}
		if name == "host" {
			return true, nil
		}
	}
	return false, rows.Err()
}

// tasksUseTaskStatusVocabulary は tasks テーブルがマイグレーション 3 の status
// 語彙を既に持つかを返す。制約全体を照合せず、マイグレーション 1 と区別できる
// 値 1 つの有無だけを見る。
func (s *Store) tasksUseTaskStatusVocabulary(ctx context.Context) (bool, error) {
	var definition sql.NullString
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='tasks'`,
	).Scan(&definition); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	normalized := strings.Join(strings.Fields(strings.ToLower(definition.String)), "")
	return strings.Contains(normalized, "'not_started'"), nil
}

func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}

func nullableTime(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	t := parseTime(value.String)
	return &t
}

// domainFeature は保存された 2 列を、アプリケーションが使う単一の status に畳む。
// status 列が意味を持つのは status_auto が 0 のときだけで、その CHECK 制約は
// 自動値より前からあるため。
func domainFeature(value db.Feature, projectID string) domain.Feature {
	status := domain.FeatureStatus(value.Status)
	if value.StatusAuto != 0 {
		status = domain.FeatureStatusAuto
	}
	return domain.Feature{
		ID:          value.PublicID,
		StorageID:   value.ID,
		ProjectID:   projectID,
		Title:       value.Title,
		Description: value.Description,
		Status:      status,
		Archived:    value.Archived != 0,
		CreatedAt:   parseTime(value.CreatedAt),
		UpdatedAt:   parseTime(value.UpdatedAt),
	}
}

// storedFeatureStatus はドメインの status を保存用の 2 値に分ける。
func storedFeatureStatus(status domain.FeatureStatus) (column string, auto int64) {
	if status == domain.FeatureStatusAuto {
		return string(domain.FeatureStatusActive), 1
	}
	return string(status), 0
}

func domainProject(value db.Project) domain.Project {
	return domain.Project{
		ID:          value.PublicID,
		StorageID:   value.ID,
		Title:       value.Title,
		Description: value.Description,
		Archived:    value.Archived != 0,
		CreatedAt:   parseTime(value.CreatedAt),
		UpdatedAt:   parseTime(value.UpdatedAt),
	}
}

func publicFeatureIDs(values []db.Feature) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		result[value.ID] = value.PublicID
	}
	return result
}

func publicProjectIDs(values []db.Project) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		result[value.ID] = value.PublicID
	}
	return result
}

func publicTaskIDs(values []db.Task) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		result[value.ID] = value.PublicID
	}
	return result
}

func domainTask(value db.Task, featureID string) domain.Task {
	return domain.Task{
		ID:               value.PublicID,
		StorageID:        value.ID,
		FeatureID:        featureID,
		StorageFeatureID: value.FeatureID,
		Title:            value.Title,
		Scope:            value.Scope,
		Status:           domain.TaskStatus(value.Status),
		Assignee:         value.Assignee,
		CreatedAt:        parseTime(value.CreatedAt),
		UpdatedAt:        parseTime(value.UpdatedAt),
		// domain.Derive を通らない経路（task create / task update）でも JSON の
		// block_labels を null にしないため、ここで空配列を入れる。
		BlockLabels: []domain.TaskBlockLabel{},
	}
}

func domainDependency(value db.Dependency, taskIDs map[string]string) domain.Dependency {
	return domain.Dependency{
		BlockerTaskID: taskIDs[value.BlockerTaskID],
		BlockedTaskID: taskIDs[value.BlockedTaskID],
		CreatedAt:     parseTime(value.CreatedAt),
	}
}

func domainDocument(
	value db.Document,
	projectIDs, featureIDs, taskIDs map[string]string,
) domain.Document {
	return domain.Document{
		ID:                   value.ID,
		ProjectID:            projectIDs[value.ProjectID.String],
		FeatureID:            featureIDs[value.FeatureID.String],
		TaskID:               taskIDs[value.TaskID.String],
		Kind:                 domain.DocumentKind(value.Kind),
		Title:                value.Title,
		Locator:              value.Locator.String,
		Content:              value.Content.String,
		IsImplementationPlan: value.IsImplementationPlan != 0,
		CreatedAt:            parseTime(value.CreatedAt),
		UpdatedAt:            parseTime(value.UpdatedAt),
	}
}

func domainListedDocument(
	value db.ListDocumentsRow,
	projectIDs, featureIDs, taskIDs map[string]string,
) domain.Document {
	return domain.Document{
		ID:                   value.ID,
		ProjectID:            projectIDs[value.ProjectID.String],
		FeatureID:            featureIDs[value.FeatureID.String],
		TaskID:               taskIDs[value.TaskID.String],
		Kind:                 domain.DocumentKind(value.Kind),
		Title:                value.Title,
		Locator:              value.Locator.String,
		IsImplementationPlan: value.IsImplementationPlan != 0,
		CreatedAt:            parseTime(value.CreatedAt),
		UpdatedAt:            parseTime(value.UpdatedAt),
	}
}

func domainPullRequest(value db.PullRequest, taskIDs map[string]string) domain.PullRequest {
	result := domain.PullRequest{
		TaskID:          taskIDs[value.TaskID],
		Host:            value.Host,
		Owner:           value.Owner,
		Repository:      value.Repository,
		Number:          value.Number,
		URL:             value.Url,
		NodeID:          value.NodeID,
		Author:          value.Author,
		State:           domain.PullRequestState(value.State),
		Draft:           value.Draft != 0,
		ReviewState:     domain.ReviewState(value.ReviewState),
		Mergeability:    domain.Mergeability(value.Mergeability),
		GitHubUpdatedAt: nullableTime(value.GithubUpdatedAt),
		LastSyncedAt:    nullableTime(value.LastSyncedAt),
		SyncError:       value.SyncError,
		Stale:           value.Stale != 0,

		ReviewRequestPending: value.ReviewRequestPending != 0,
		ChangesRequestedAt:   nullableTime(value.ChangesRequestedAt),
		LastPushedAt:         nullableTime(value.LastPushedAt),
	}
	if err := json.Unmarshal([]byte(value.AssigneesJson), &result.Assignees); err != nil {
		result.Assignees = []string{}
	}
	result.DisplayState = domain.PRDisplayState(&result)
	return result
}

func mapNotFound(err error, entity, id string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NewError(domain.DomainErrorCodeNotFound, "%s %q was not found", entity, id)
	}
	return err
}
