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

// migrationFile は埋め込みマイグレーション 1 件と、その名前が示すバージョン。
type migrationFile struct {
	name    string
	version int
}

// migrationFiles は埋め込みマイグレーションを適用順に列挙する。
// マイグレーション実行側と診断レポートは同じ一覧を読むため、
// レポートが embedded と呼ぶバージョンは実行側が適用するものと一致する。
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

// Path は、この store を開いたときに解決したデータベースの位置を返す。
func (s *Store) Path() string { return s.path }

// AppliedSchemaVersion はデータベースに記録された最大のマイグレーションを読む。
// schema_migrations は管理対象スキーマではなくマイグレーション実行側が作るため、
// sqlc のモデルがなくクエリをここに直書きしている。
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

// EmbeddedSchemaVersion はこのバイナリが持つ最大のマイグレーションを返す。
// これより進んだデータベースは、より新しい PRX が書いたものである。
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

// DatabaseFile はデータベースのディスク上の状態を返す。インメモリや DSN 形式の
// 位置には単一のファイルがないため、存在しないパスを示す代わりに
// 対象外として報告する。
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
	// O_CREATE は意図的に外す。読み取り専用の診断が対象ファイルを作ってはならない。
	// 書き込みで開けば、読み取り専用ボリュームや ACL など
	// パーミッションビットでは分からない場合も検出できる。
	file, err := os.OpenFile(s.path, os.O_WRONLY, 0)
	if err != nil {
		result.WriteError = err.Error()
		return result
	}
	_ = file.Close()
	result.Writable = true
	return result
}

// ListGitHubRepositoryAuthCache はリポジトリごとに最後に成功した認証情報を返す。
// このキャッシュは認証情報の中身を保持しないため、全体をそのまま報告してよい。
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
