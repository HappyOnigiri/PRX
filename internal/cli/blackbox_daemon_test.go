package cli_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/runstate"
)

// serverLog は子プロセスの stderr を安全に読める形で集める。exec は非 *os.File の
// writer へ別の goroutine から書き込むので、読み手と排他しないと競合する。
type serverLog struct {
	mu   sync.Mutex
	text strings.Builder
}

func (l *serverLog) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.text.Write(data)
}

func (l *serverLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.text.String()
}

// daemonEnvironment は launchd と稼働記録を実ユーザーの環境から隔離する。docs/development.md の
// とおり、daemon のテストは実 launchctl と実 ~/Library/LaunchAgents に触れてはならない。
type daemonEnvironment struct {
	home          string
	runDirectory  string
	databasePath  string
	configPath    string
	launchctlLog  string
	variables     []string
	runStatePath  string
	pathDirectory string
}

func newDaemonEnvironment(t *testing.T) daemonEnvironment {
	t.Helper()
	root := t.TempDir()
	environment := daemonEnvironment{
		home:          filepath.Join(root, "home"),
		runDirectory:  filepath.Join(root, "run"),
		databasePath:  filepath.Join(root, "prx.db"),
		configPath:    filepath.Join(root, "config.yaml"),
		launchctlLog:  filepath.Join(root, "launchctl.log"),
		pathDirectory: filepath.Join(root, "bin"),
	}
	environment.runStatePath = filepath.Join(environment.runDirectory, "serve.json")
	if err := os.MkdirAll(environment.pathDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(environment.home, 0o700); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> " + environment.launchctlLog + "\nexit 0\n"
	launchctl := filepath.Join(environment.pathDirectory, "launchctl")
	if err := os.WriteFile(launchctl, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	environment.variables = []string{
		"HOME=" + environment.home,
		"PRX_RUN_DIR=" + environment.runDirectory,
		"PATH=" + environment.pathDirectory,
	}
	return environment
}

func (e daemonEnvironment) args(extra ...string) []string {
	return append([]string{"--db", e.databasePath, "--config", e.configPath, "--json"}, extra...)
}

// TestBlackBoxDaemonCommandsStayInsideTheInjectedEnvironment は daemon コマンドの JSON 契約と、
// plist が差し替えた HOME の下だけに書かれることを確かめる。start と restart は launchd に
// サーバーを起こさせる必要があるので、実機確認に委ねている。
func TestBlackBoxDaemonCommandsStayInsideTheInjectedEnvironment(t *testing.T) {
	binary := buildCLI(t)
	environment := newDaemonEnvironment(t)
	run := func(args ...string) map[string]json.RawMessage {
		result := executeCLIWithEnv(t, binary, "", environment.variables, environment.args(args...)...)
		return assertNormalSuccess(t, result)
	}
	status := run("daemon")
	assertDirectObjectKeys(t, status, "supported", "installed", "plist_path", "plist_status", "running",
		"pid", "address", "url", "started_at", "uptime_seconds", "version", "binary_matches", "log_path")
	if !daemonSupported(t, status) {
		// 非対応 OS には plist も launchctl もないので、状態の語彙ではなく変更系が
		// daemon_unsupported で退くことだけを確かめる。
		assertDaemonUnsupported(t, binary, environment, "daemon", "stop")
		return
	}
	if !strings.Contains(string(status["plist_status"]), "unknown") {
		t.Fatalf("plist status before install=%s", status["plist_status"])
	}
	assertDirectObjectKeys(t, run("daemon", "stop"), "stopped", "already_stopped")
	// install は launchd が起こしたサーバーの記録を待つ。launchctl はスタブなので、
	// 記録は launchd の代わりにこのプロセスが用意する。
	lock := holdManagedRunState(t, environment)
	defer func() { _ = lock.Release() }()
	assertDirectObjectKeys(t, run("daemon", "install"), "installed", "label", "plist_path", "address", "url")
	plist := filepath.Join(environment.home, "Library", "LaunchAgents", "com.user.prx.plist")
	info, err := os.Stat(plist)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("plist info=%v err=%v", info, err)
	}
	body, err := os.ReadFile(plist)
	if err != nil {
		t.Fatal(err)
	}
	// plist が --addr も --demo も渡さないことが、loopback 外公開と demo を daemon から
	// 締め出す構造的な保証になる。
	for _, forbidden := range []string{"--addr", "--demo"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("plist passes %s: %s", forbidden, body)
		}
	}
	if installed := run("daemon"); !strings.Contains(string(installed["plist_status"]), "current") {
		t.Fatalf("plist status after install=%s", installed["plist_status"])
	}
	assertDirectObjectKeys(t, run("daemon", "uninstall"), "uninstalled", "label", "plist_path")
	if _, err := os.Stat(plist); !os.IsNotExist(err) {
		t.Fatalf("plist remains after uninstall: %v", err)
	}
	log, err := os.ReadFile(environment.launchctlLog)
	if err != nil || !strings.Contains(string(log), "bootstrap") || !strings.Contains(string(log), "bootout") {
		t.Fatalf("launchctl log=%q err=%v", log, err)
	}
}

// holdManagedRunState は launchd 管理下のサーバーが稼働している状態を作る。ロックを保持
// している間だけ記録の内容が有効になるので、Release まで稼働中として観測される。
func holdManagedRunState(t *testing.T, environment daemonEnvironment) *runstate.Lock {
	t.Helper()
	t.Setenv(runstate.DirEnvironmentVariable, environment.runDirectory)
	lock, held, err := runstate.Acquire()
	if err != nil || !held {
		t.Fatalf("acquire held=%v err=%v", held, err)
	}
	state := runstate.State{
		PID: os.Getpid(), Address: "127.0.0.1:7331", URL: "http://127.0.0.1:7331",
		StartedAt: time.Now().UTC(), LaunchdManaged: true,
	}
	if err := lock.Write(state); err != nil {
		_ = lock.Release()
		t.Fatal(err)
	}
	return lock
}

// assertDaemonUnsupported は非対応 OS で変更系のコマンドが daemon_unsupported で失敗する
// ことを確かめる。
func assertDaemonUnsupported(t *testing.T, binary string, environment daemonEnvironment, args ...string) {
	t.Helper()
	result := executeCLIWithEnv(t, binary, "", environment.variables, environment.args(args...)...)
	if result.exit == 0 {
		t.Fatalf("%v succeeded on an unsupported OS: %q", args, result.stdout)
	}
	if code := decodeFailure(t, []byte(result.stderr), result.stderr).ErrorCode; code != "daemon_unsupported" {
		t.Fatalf("%v error=%q", args, result.stderr)
	}
}

func daemonSupported(t *testing.T, status map[string]json.RawMessage) bool {
	t.Helper()
	var supported bool
	if err := json.Unmarshal(status["supported"], &supported); err != nil {
		t.Fatal(err)
	}
	return supported
}

// TestBlackBoxOpenRefusesToGuessAnAddress は稼働記録がないときに `prx open` が失敗し、
// 稼働中でないサーバーのポートを開かないことを確かめる。
func TestBlackBoxOpenRefusesToGuessAnAddress(t *testing.T) {
	binary := buildCLI(t)
	environment := newDaemonEnvironment(t)
	result := executeCLIWithEnv(t, binary, "", environment.variables, environment.args("open", "--print")...)
	if result.exit == 0 {
		t.Fatalf("open succeeded without a server: %q", result.stdout)
	}
	failure := decodeFailure(t, []byte(result.stderr), result.stderr)
	if failure.ErrorCode != "daemon_not_running" && failure.ErrorCode != "daemon_unsupported" {
		t.Fatalf("open error=%+v", failure)
	}
}

// TestBlackBoxDaemonCommandsDoNotOpenStorage は daemon と open がデータベースを作らない
// ことを確かめる。どちらも launchd と稼働記録だけを見るので、ストレージを触る理由がない。
func TestBlackBoxDaemonCommandsDoNotOpenStorage(t *testing.T) {
	binary := buildCLI(t)
	environment := newDaemonEnvironment(t)
	for _, args := range [][]string{{"daemon"}, {"daemon", "stop"}, {"open", "--print"}} {
		executeCLIWithEnv(t, binary, "", environment.variables, environment.args(args...)...)
	}
	if _, err := os.Stat(environment.databasePath); !os.IsNotExist(err) {
		t.Fatalf("daemon commands opened the database: %v", err)
	}
	if _, err := os.Stat(environment.configPath); !os.IsNotExist(err) {
		t.Fatalf("daemon commands wrote the configuration: %v", err)
	}
}

// TestBlackBoxDaemonCommandsRefuseDemo は demo との併用が構造的に不可能であることを
// 確かめる。--demo は serve だけのフラグで、plist もそれを渡さないので、launchd が
// 起動したサーバーが一時データで動くことはない。
func TestBlackBoxDaemonCommandsRefuseDemo(t *testing.T) {
	binary := buildCLI(t)
	environment := newDaemonEnvironment(t)
	for _, args := range [][]string{{"daemon", "--demo"}, {"open", "--demo"}, {"daemon", "install", "--demo"}} {
		result := executeCLIWithEnv(
			t, binary, "", environment.variables, append([]string{"--json"}, args...)...,
		)
		if result.exit == 0 {
			t.Fatalf("%v succeeded with --demo: %q", args, result.stdout)
		}
		if !strings.Contains(result.stderr, "unknown flag: --demo") {
			t.Fatalf("%v error=%q", args, result.stderr)
		}
	}
}

// TestBlackBoxManagedServeOwnsTheRunState は稼働記録を書くのが通常起動だけであること、
// 2 つ目の serve が exit 0 で退くこと、SIGTERM が exit 0 になることを確かめる。
// 最後の 2 つは LaunchAgent の KeepAlive{SuccessfulExit:false} の前提である。
func TestBlackBoxManagedServeOwnsTheRunState(t *testing.T) {
	binary := buildCLI(t)
	environment := newDaemonEnvironment(t)
	server := startManagedServe(t, binary, environment)
	state := waitForRunStateFile(t, environment.runStatePath)
	if state["address"] == nil || state["url"] == nil || state["pid"] == nil {
		t.Fatalf("run state=%v", state)
	}
	var url string
	if err := json.Unmarshal(state["url"], &url); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatalf("run state url=%q, want a loopback address", url)
	}

	printed := executeCLIWithEnv(t, binary, "", environment.variables, environment.args("open", "--print")...)
	if printed.exit == 0 {
		if !strings.Contains(printed.stdout, url) {
			t.Fatalf("open --print stdout=%q, want %q", printed.stdout, url)
		}
	} else if code := decodeFailure(t, []byte(printed.stderr), printed.stderr).ErrorCode; code != "daemon_unsupported" {
		t.Fatalf("open --print error=%q", code)
	}

	second := executeCLIWithEnv(t, binary, "", environment.variables, environment.args("serve")...)
	if second.exit != 0 || !strings.Contains(second.stderr, "already running") {
		t.Fatalf("second serve exit=%d stderr=%q", second.exit, second.stderr)
	}

	if err := server.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := server.Wait(); err != nil {
		t.Fatalf("serve did not exit 0 on SIGTERM: %v", err)
	}
	// 正常終了は内容を空にするだけで unlink しない。unlink とほぼ同時に新しい
	// インスタンスが同じパスを作ってロックしていると、それを消してしまう。
	body, err := os.ReadFile(environment.runStatePath)
	if err != nil || len(body) != 0 {
		t.Fatalf("run state after shutdown=%q err=%v", body, err)
	}
}

func TestBlackBoxAdHocServeDoesNotClaimTheRunState(t *testing.T) {
	binary := buildCLI(t)
	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "explicit address", args: []string{"serve", "--addr", "127.0.0.1:0"}},
		{name: "demo", args: []string{"serve", "--demo"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			environment := newDaemonEnvironment(t)
			arguments := environment.args(test.args...)
			if test.name == "demo" {
				arguments = append([]string{"--json"}, test.args...)
			}
			server := exec.CommandContext(context.Background(), binary, arguments...)
			server.Env = append(os.Environ(), environment.variables...)
			log := &serverLog{}
			server.Stderr = log
			if err := server.Start(); err != nil {
				t.Fatal(err)
			}
			waitForListening(t, log)
			if _, err := os.Stat(environment.runStatePath); !os.IsNotExist(err) {
				t.Fatalf("an ad hoc server claimed the run state: %v", err)
			}
			_ = server.Process.Signal(syscall.SIGTERM)
			_ = server.Wait()
		})
	}
}

func startManagedServe(t *testing.T, binary string, environment daemonEnvironment) *exec.Cmd {
	t.Helper()
	server := exec.CommandContext(context.Background(), binary, environment.args("serve")...)
	server.Env = append(os.Environ(), environment.variables...)
	log := &serverLog{}
	server.Stderr = log
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = server.Process.Signal(syscall.SIGKILL)
		_ = server.Wait()
	})
	waitForListening(t, log)
	return server
}

func waitForListening(t *testing.T, log *serverLog) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !strings.Contains(log.String(), "PRX listening on") {
		if time.Now().After(deadline) {
			t.Fatalf("serve did not start: %s", log.String())
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func waitForRunStateFile(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		body, err := os.ReadFile(path)
		if err == nil && len(body) > 0 {
			return decodeObject(t, body, string(body))
		}
		if time.Now().After(deadline) {
			t.Fatalf("run state %s was never written: %v", path, err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
