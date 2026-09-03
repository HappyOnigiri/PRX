package store

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/db"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

// migrationFile is one embedded migration and the version its name encodes.
type migrationFile struct {
	name    string
	version int
}

// migrationFiles lists the embedded migrations in application order. The
// migration runner and the diagnostic report read the same list, so the version
// a report calls embedded is the one the runner would apply.
func migrationFiles() ([]migrationFile, error) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	result := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, _ := strings.Cut(entry.Name(), "_")
		version, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("invalid migration name %q", entry.Name())
		}
		result = append(result, migrationFile{name: entry.Name(), version: version})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].name < result[j].name })
	return result, nil
}

// Path reports the database location this store resolved when it was opened.
func (s *Store) Path() string { return s.path }

// AppliedSchemaVersion reads the highest migration recorded in the database.
// schema_migrations is created by the migration runner rather than by the
// tracked schema, so sqlc has no model for it and the query is written here.
func (s *Store) AppliedSchemaVersion(ctx context.Context) (int, error) {
	var version int
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`,
	).Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

// EmbeddedSchemaVersion reports the highest migration this binary carries. A
// database ahead of it was written by a newer PRX.
func (s *Store) EmbeddedSchemaVersion() (int, error) {
	files, err := migrationFiles()
	if err != nil {
		return 0, err
	}
	highest := 0
	for _, file := range files {
		if file.version > highest {
			highest = file.version
		}
	}
	return highest, nil
}

// DatabaseFile reports the on-disk state of the database. An in-memory or
// DSN-style location has no single file, so it reports itself as inapplicable
// instead of describing a path that does not exist.
func (s *Store) DatabaseFile() domain.DebugDatabaseFile {
	if !isDatabaseFilePath(s.path) {
		return domain.DebugDatabaseFile{}
	}
	result := domain.DebugDatabaseFile{Applicable: true}
	info, err := os.Stat(s.path)
	if err != nil {
		result.WriteError = err.Error()
		return result
	}
	result.SizeBytes = info.Size()
	if wal, walErr := os.Stat(s.path + "-wal"); walErr == nil {
		result.WALPresent = true
		result.WALSizeBytes = wal.Size()
	}
	if _, shmErr := os.Stat(s.path + "-shm"); shmErr == nil {
		result.SHMPresent = true
	}
	// The probe deliberately omits O_CREATE: a read-only diagnostic must not
	// create the file it is reporting on. Opening for writing also catches the
	// cases a permission bit does not, such as a read-only volume or an ACL.
	file, err := os.OpenFile(s.path, os.O_WRONLY, 0)
	if err != nil {
		result.WriteError = err.Error()
		return result
	}
	_ = file.Close()
	result.Writable = true
	return result
}

// ListGitHubRepositoryAuthCache reports which credential last succeeded for each
// repository. The cache never holds credential material, so it is safe to
// report in full.
func (s *Store) ListGitHubRepositoryAuthCache(ctx context.Context) ([]domain.DebugAuthCacheEntry, error) {
	rows, err := db.New(s.db).ListGitHubRepositoryAuthCache(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.DebugAuthCacheEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.DebugAuthCacheEntry{
			Host:            row.Host,
			Owner:           row.Owner,
			Repository:      row.Repository,
			AuthMethodID:    row.AuthMethodID,
			LastSucceededAt: parseTime(row.LastSucceededAt),
		})
	}
	return result, nil
}

func isDatabaseFilePath(path string) bool {
	return path != "" && path != ":memory:" && !strings.HasPrefix(path, "file:")
}
