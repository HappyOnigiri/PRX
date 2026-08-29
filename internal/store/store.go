package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/HappyOnigiri/PRX/internal/db"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db  *sql.DB
	now func() time.Time
}

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
	// _txlock=immediate takes the write lock when the transaction starts. With the
	// default deferred BEGIN, a read-modify-write transaction that reads first
	// fails with SQLITE_BUSY_SNAPSHOT on write, which busy_timeout does not retry.
	dsn += separator + "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate"
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite gives every connection to :memory: its own private database, so a
	// second pooled connection would reach an empty, unmigrated one.
	if path == ":memory:" {
		database.SetMaxOpenConns(1)
		database.SetMaxIdleConns(1)
	} else {
		database.SetMaxOpenConns(8)
		database.SetMaxIdleConns(4)
	}
	store := &Store{db: database, now: func() time.Time { return time.Now().UTC() }}
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
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, _ := strings.Cut(entry.Name(), "_")
		version, err := strconv.Atoi(prefix)
		if err != nil {
			return fmt.Errorf("invalid migration name %q", entry.Name())
		}
		var exists int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=?`, version).
			Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		body, err := migrations.ReadFile("migrations/" + entry.Name())
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
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
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

func domainFeature(value db.Feature) domain.Feature {
	return domain.Feature{
		ID:          value.ID,
		Slug:        value.Slug,
		Title:       value.Title,
		Description: value.Description,
		Status:      domain.FeatureStatus(value.Status),
		Archived:    value.Archived != 0,
		CreatedAt:   parseTime(value.CreatedAt),
		UpdatedAt:   parseTime(value.UpdatedAt),
	}
}

func domainTask(value db.Task) domain.Task {
	return domain.Task{
		ID:        value.ID,
		FeatureID: value.FeatureID,
		Title:     value.Title,
		Scope:     value.Scope,
		Kind:      domain.TaskKind(value.Kind),
		Status:    domain.TaskStatus(value.Status),
		Assignee:  value.Assignee,
		CreatedAt: parseTime(value.CreatedAt),
		UpdatedAt: parseTime(value.UpdatedAt),
	}
}

func domainDependency(value db.Dependency) domain.Dependency {
	return domain.Dependency{
		BlockerTaskID: value.BlockerTaskID,
		BlockedTaskID: value.BlockedTaskID,
		CreatedAt:     parseTime(value.CreatedAt),
	}
}

func domainDocument(value db.Document) domain.Document {
	return domain.Document{
		ID:        value.ID,
		FeatureID: value.FeatureID.String,
		TaskID:    value.TaskID.String,
		Kind:      domain.DocumentKind(value.Kind),
		Title:     value.Title,
		Value:     value.Value,
		CreatedAt: parseTime(value.CreatedAt),
	}
}

func domainPullRequest(value db.PullRequest) domain.PullRequest {
	result := domain.PullRequest{
		TaskID:          value.TaskID,
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
	}
	if err := jsonUnmarshal([]byte(value.AssigneesJson), &result.Assignees); err != nil {
		result.Assignees = []string{}
	}
	result.DisplayState = domain.PullRequestDisplayState(domain.PRDisplayState(&result))
	return result
}

func mapNotFound(err error, entity, id string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NewError(domain.DomainErrorCodeNotFound, "%s %q was not found", entity, id)
	}
	return err
}
