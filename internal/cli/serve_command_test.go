package cli

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/launchd"
)

// stubListener は bind の成否だけを扱うテスト用の listener。実ポートを掴むテストは
// 並列 CI で 7331 や 7332 を奪い合うので書かない。
type stubListener struct{ address string }

func (l stubListener) Accept() (net.Conn, error) { return nil, errors.New("stub listener") }

func (l stubListener) Close() error { return nil }

func (l stubListener) Addr() net.Addr { return stubAddr(l.address) }

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }

func (a stubAddr) String() string { return string(a) }

func TestPortListenPlanFallsBackOnlyForAuto(t *testing.T) {
	if plan := portListenPlan(config.ServerPortAuto); plan.address != "127.0.0.1:7331" || !plan.fallback {
		t.Fatalf("auto plan=%+v", plan)
	}
	if plan := portListenPlan(config.ServerPort(7400)); plan.address != "127.0.0.1:7400" || plan.fallback {
		t.Fatalf("fixed plan=%+v", plan)
	}
}

func TestListenServeFallsBackOnlyForAddressInUse(t *testing.T) {
	for _, test := range []struct {
		name        string
		plan        listenPlan
		bindErr     error
		wantAttempt []string
		wantCode    domain.DomainErrorCode
		wantErr     error
	}{
		{
			name:        "auto binds the preferred port",
			plan:        listenPlan{address: "127.0.0.1:7331", fallback: true},
			wantAttempt: []string{"127.0.0.1:7331"},
		},
		{
			name:        "auto escapes a busy preferred port",
			plan:        listenPlan{address: "127.0.0.1:7331", fallback: true},
			bindErr:     syscall.EADDRINUSE,
			wantAttempt: []string{"127.0.0.1:7331", "127.0.0.1:0"},
		},
		{
			name:        "auto keeps a privileged port failure",
			plan:        listenPlan{address: "127.0.0.1:80", fallback: true},
			bindErr:     syscall.EACCES,
			wantAttempt: []string{"127.0.0.1:80"},
			wantErr:     syscall.EACCES,
		},
		{
			name:        "a fixed port does not move",
			plan:        listenPlan{address: "127.0.0.1:7400"},
			bindErr:     syscall.EADDRINUSE,
			wantAttempt: []string{"127.0.0.1:7400"},
			wantCode:    domain.DomainErrorCodeAddressInUse,
		},
		{
			name:        "an explicit address does not move",
			plan:        listenPlan{address: "0.0.0.0:7331"},
			bindErr:     syscall.EADDRINUSE,
			wantAttempt: []string{"0.0.0.0:7331"},
			wantCode:    domain.DomainErrorCodeAddressInUse,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			attempted := make([]string, 0, 2)
			listen := func(_ context.Context, network, address string) (net.Listener, error) {
				if network != "tcp" {
					t.Fatalf("network=%q", network)
				}
				attempted = append(attempted, address)
				if test.bindErr != nil && address == test.plan.address {
					return nil, &net.OpError{Op: "listen", Err: test.bindErr}
				}
				return stubListener{address: address}, nil
			}
			listener, err := listenServe(context.Background(), listen, test.plan)
			if len(attempted) != len(test.wantAttempt) {
				t.Fatalf("attempted=%v, want %v", attempted, test.wantAttempt)
			}
			for index, address := range test.wantAttempt {
				if attempted[index] != address {
					t.Fatalf("attempted=%v, want %v", attempted, test.wantAttempt)
				}
			}
			switch {
			case test.wantCode != "":
				if domain.ErrorCode(err) != test.wantCode {
					t.Fatalf("error=%v, want code %q", err, test.wantCode)
				}
			case test.wantErr != nil:
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error=%v, want %v", err, test.wantErr)
				}
				if domain.ErrorCode(err) == domain.DomainErrorCodeAddressInUse {
					t.Fatalf("error=%v was reported as address_in_use", err)
				}
			default:
				if err != nil || listener == nil {
					t.Fatalf("listener=%v err=%v", listener, err)
				}
			}
		})
	}
}

// TestExecutableMatchesComparesInodeSizeAndModificationTime は 3 つの比較のどれが欠けても
// 置換を見逃すことを確かめる。見逃すと古いサーバーが古い埋め込みスキーマで応答し続ける。
func TestExecutableMatchesComparesInodeSizeAndModificationTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prx")
	if err := os.WriteFile(path, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	baseline := statExecutable(t, path)
	if !executableMatches(baseline, statExecutable(t, path)) {
		t.Fatal("an untouched executable was reported as replaced")
	}
	if err := os.WriteFile(path, []byte("binary+"), 0o700); err != nil {
		t.Fatal(err)
	}
	if executableMatches(baseline, statExecutable(t, path)) {
		t.Fatal("a resized executable was reported as matching")
	}
	if err := os.WriteFile(path, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	touched := baseline.ModTime().Add(time.Minute)
	if err := os.Chtimes(path, touched, touched); err != nil {
		t.Fatal(err)
	}
	if executableMatches(baseline, statExecutable(t, path)) {
		t.Fatal("an executable with a new modification time was reported as matching")
	}
	replaced := filepath.Join(t.TempDir(), "prx")
	if err := os.WriteFile(replaced, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(replaced, baseline.ModTime(), baseline.ModTime()); err != nil {
		t.Fatal(err)
	}
	if executableMatches(baseline, statExecutable(t, replaced)) {
		t.Fatal("a different inode with the same size and time was reported as matching")
	}
}

// TestWatchExecutablePathWarnsWhenTheServerIsNotManaged は管理外のサーバーが自己 kickstart
// しないことを確かめる。置換側が flock を取れずに死ぬと、手動起動の 1 つだけが残る。
func TestWatchExecutablePathWarnsWhenTheServerIsNotManaged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prx")
	if err := os.WriteFile(path, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	errOut := &strings.Builder{}
	s := &state{out: io.Discard, errOut: errOut}
	manager := launchd.New()
	if manager.Managed() {
		t.Skip("the test process is managed by launchd")
	}
	// 置換は監視が基準を取った後に起こす必要があるので、周期より長く待ってから書き換える。
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = os.WriteFile(path, []byte("replaced"), 0o700)
	}()
	// ctx は監視が終わらない場合の保険。管理外だと気づいた監視は自分で戻る。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.watchExecutablePath(ctx, manager, path, 5*time.Millisecond)
	if !strings.Contains(errOut.String(), "not managed by launchd") {
		t.Fatalf("stderr=%q", errOut.String())
	}
}

// TestWatchExecutablePathDisablesItselfWithoutABaseline は基準が取れないときに監視を
// 無効にすることを確かめる。stat の失敗を未変更の根拠にすると置換を見逃す。
func TestWatchExecutablePathDisablesItselfWithoutABaseline(t *testing.T) {
	errOut := &strings.Builder{}
	s := &state{out: io.Discard, errOut: errOut}
	s.watchExecutablePath(context.Background(), launchd.New(), filepath.Join(t.TempDir(), "missing"), time.Hour)
	if !strings.Contains(errOut.String(), "automatic restart after a replacement is disabled") {
		t.Fatalf("stderr=%q", errOut.String())
	}
}

func statExecutable(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}
