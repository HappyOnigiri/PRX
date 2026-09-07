package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// レポートはデータベースを開けない実行のためにあるので、オープンの失敗が
// その理由を説明するコマンド自体を失敗させてはならない。
func TestDebugSucceedsWhenTheServiceCannotBeOpened(t *testing.T) {
	var out, errOut bytes.Buffer
	err := Execute(
		context.Background(),
		[]string{"--db", "/nonexistent/directory/prx.db", "debug"},
		&out,
		&errOut,
		func(context.Context, ServiceOptions) (Service, io.Closer, error) {
			return nil, nil, &ServiceOpenError{
				DatabasePath: "/nonexistent/directory/prx.db",
				Err:          errors.New("create database directory: permission denied"),
			}
		},
	)
	if err != nil {
		t.Fatalf("debug failed: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "storage_unavailable") ||
		!strings.Contains(text, "create database directory: permission denied") {
		t.Fatalf("report did not explain the failure:\n%s", text)
	}
	// オープンが試みた解決済みの場所は、読み手が最初に必要とする情報。
	if !strings.Contains(text, "database_path: /nonexistent/directory/prx.db") {
		t.Fatalf("report omitted the attempted database path:\n%s", text)
	}
}

func TestDebugJSONReportsTheSameProblems(t *testing.T) {
	var out, errOut bytes.Buffer
	err := Execute(
		context.Background(),
		[]string{"--json", "--db", "unreadable.db", "debug"},
		&out,
		&errOut,
		func(context.Context, ServiceOptions) (Service, io.Closer, error) {
			return nil, nil, errors.New("open sqlite: unable to open database file")
		},
	)
	if err != nil {
		t.Fatalf("debug failed: %v", err)
	}
	var report domain.DebugReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, out.String())
	}
	if len(report.Problems) == 0 || report.Problems[0].Code != domain.DebugProblemCodeStorageUnavailable {
		t.Fatalf("problems=%+v", report.Problems)
	}
	if report.Storage.Error == "" || report.Build.Version == "" {
		t.Fatalf("report=%+v", report)
	}
}

// 更新をかけると、送ってもらう対象の実行エラーが消え、全 pull request の
// 鮮度も書き換わってしまうので、診断コマンドは決して更新を起動しない。
func TestDebugDoesNotStartAnAutomaticSync(t *testing.T) {
	service := &recordingDebugService{}
	root, state := newRootWithState(
		io.Discard,
		io.Discard,
		func(context.Context, ServiceOptions) (Service, io.Closer, error) { return service, nil, nil },
	)
	debug, _, err := root.Find([]string{"debug"})
	if err != nil {
		t.Fatal(err)
	}
	if err := root.PersistentPreRunE(debug, nil); err != nil {
		t.Fatal(err)
	}
	defer root.PersistentPostRun(debug, nil)
	if service.syncIfDueCalls != 0 {
		t.Fatalf("debug started %d automatic refreshes", service.syncIfDueCalls)
	}
	if state.service == nil {
		t.Fatal("debug did not open the service it can use")
	}
}

func TestDebugRecordsHowEachLocationWasSelected(t *testing.T) {
	// 設定ストアは渡されたパスの隣にロックファイルを作るので、環境変数の値は
	// 一時ディレクトリを指す。
	t.Setenv("PRX_CONFIG", filepath.Join(t.TempDir(), "from-environment.yaml"))
	var got ServiceOptions
	root, _ := newRootWithState(
		io.Discard,
		io.Discard,
		func(_ context.Context, options ServiceOptions) (Service, io.Closer, error) {
			got = options
			return &recordingDebugService{}, nil, nil
		},
	)
	root.SetArgs([]string{"--db", "explicit.db", "debug"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got.DatabasePathSource != "flag" || got.ConfigPathSource != "env" {
		t.Fatalf("options=%+v", got)
	}
}

type recordingDebugService struct {
	Service
	syncIfDueCalls int
}

func (s *recordingDebugService) SyncIfDue(context.Context) (bool, domain.GitHubSyncStatus, error) {
	s.syncIfDueCalls++
	return false, domain.GitHubSyncStatus{}, nil
}

func (*recordingDebugService) Debug(context.Context) (domain.DebugReport, error) {
	return domain.DebugReport{Build: domain.NewDebugBuild("0.0.0-test")}, nil
}
