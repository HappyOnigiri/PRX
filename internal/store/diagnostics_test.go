package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/store"
)

func TestStoreReportsResolvedPathAndSchemaVersions(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "diagnostics.db")
	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if database.Path() != path {
		t.Fatalf("path=%q, want %q", database.Path(), path)
	}
	applied, err := database.AppliedSchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	embedded, err := database.EmbeddedSchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	// A freshly opened database has every embedded migration applied, so the two
	// versions must agree; a difference is what identifies a database written by
	// another PRX build.
	if applied != embedded || embedded != 9 {
		t.Fatalf("applied=%d embedded=%d, want 9", applied, embedded)
	}
}

func TestStoreReportsDatabaseFileState(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "file-state.db")
	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	file := database.DatabaseFile()
	if !file.Applicable || file.SizeBytes == 0 || !file.Writable || file.WriteError != "" {
		t.Fatalf("database file=%+v", file)
	}
	// The store opens SQLite in WAL mode, so the log file is expected beside it.
	if !file.WALPresent {
		t.Fatalf("write-ahead log was not detected: %+v", file)
	}
	if _, statErr := os.Stat(path + "-wal"); statErr != nil {
		t.Fatalf("stat wal: %v", statErr)
	}
}

func TestStoreReportsAnInMemoryDatabaseAsInapplicable(t *testing.T) {
	database, err := store.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if file := database.DatabaseFile(); file.Applicable {
		t.Fatalf("an in-memory database has no file to report: %+v", file)
	}
}

func TestStoreReportsAnUnwritableDatabaseFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "read-only.db")
	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })

	file := database.DatabaseFile()
	if file.Writable || file.WriteError == "" {
		t.Fatalf("database file=%+v", file)
	}
}

func TestStoreListsAuthenticationCacheInStableOrder(t *testing.T) {
	ctx := context.Background()
	database, _ := openTestService(t)
	rows := [][4]string{
		{"github.com", "acme", "web", "work"},
		{"ghe.example.com", "acme", "api", "enterprise"},
		{"github.com", "acme", "api", "work"},
	}
	for _, row := range rows {
		if err := database.UpsertGitHubRepositoryAuthCache(ctx, row[0], row[1], row[2], row[3]); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := database.ListGitHubRepositoryAuthCache(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries=%+v", entries)
	}
	want := []string{"ghe.example.com/acme/api", "github.com/acme/api", "github.com/acme/web"}
	for index, entry := range entries {
		got := entry.Host + "/" + entry.Owner + "/" + entry.Repository
		if got != want[index] {
			t.Fatalf("entry %d = %q, want %q", index, got, want[index])
		}
		if entry.AuthMethodID == "" || entry.LastSucceededAt.IsZero() {
			t.Fatalf("entry %d is incomplete: %+v", index, entry)
		}
	}
}
