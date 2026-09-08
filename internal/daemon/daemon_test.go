package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/runstate"
)

func TestInspectReportsAnUnsupportedOperatingSystemWithoutTouchingTheDisk(t *testing.T) {
	// PRX_RUN_DIR を存在しない場所に向けても観測が失敗しないことを兼ねて確かめる。
	t.Setenv(runstate.DirEnvironmentVariable, filepath.Join(t.TempDir(), "missing"))
	status := Inspect(&launchd.Manager{}, "1.2.3")
	if status.Supported || status.Installed || status.Running || status.Error != "" {
		t.Fatalf("status=%+v", status)
	}
	if input := status.DebugInput(false); input.Supported || input.PlistStatus != "" {
		t.Fatalf("debug input=%+v", input)
	}
}

// TestInspectReportsARunningServerOnAnUnsupportedOperatingSystem は常駐が使えない OS でも
// 稼働記録を報告することを確かめる。記録は flock だけに依存するので OS を問わない。
func TestInspectReportsARunningServerOnAnUnsupportedOperatingSystem(t *testing.T) {
	t.Setenv(runstate.DirEnvironmentVariable, filepath.Join(t.TempDir(), "run"))
	lock, held, err := runstate.Acquire()
	if err != nil || !held {
		t.Fatalf("acquire held=%v err=%v", held, err)
	}
	defer func() { _ = lock.Release() }()
	if err := lock.Write(runstate.State{
		PID: os.Getpid(), Address: "127.0.0.1:7331", URL: "http://127.0.0.1:7331",
		StartedAt: time.Now().UTC(), Version: "1.2.3",
	}); err != nil {
		t.Fatal(err)
	}
	status := Inspect(&launchd.Manager{}, "1.2.3")
	if status.Supported || status.Installed || status.PlistPath != "" {
		t.Fatalf("status=%+v", status)
	}
	if !status.Running || status.State.URL != "http://127.0.0.1:7331" || status.Error != "" {
		t.Fatalf("status=%+v", status)
	}
}

// TestBinaryMatchesTreatsALaterExecutableAsAReplacement は稼働中サーバーの版ずれを
// 検出する。新しい CLI がデータベースを移行した後も古いサーバーが応答し続けるのを
// 読み手に知らせるための判定である。
func TestBinaryMatchesTreatsALaterExecutableAsAReplacement(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "prx")
	if err := os.WriteFile(executable, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC()
	if err := os.Chtimes(executable, started.Add(-time.Hour), started.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	state := runstate.State{Version: "1.2.3", Executable: executable, StartedAt: started}
	if !binaryMatches(state, "1.2.3") {
		t.Fatal("an untouched executable was reported as replaced")
	}
	if binaryMatches(state, "1.2.4") {
		t.Fatal("a different version was reported as matching")
	}
	if err := os.Chtimes(executable, started.Add(time.Minute), started.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if binaryMatches(state, "1.2.3") {
		t.Fatal("an executable replaced after the start was reported as matching")
	}
	// stat できないことを不一致の根拠にはしない。誤検出は解決策のない問題として報告される。
	if err := os.Remove(executable); err != nil {
		t.Fatal(err)
	}
	if !binaryMatches(state, "1.2.3") {
		t.Fatal("a missing executable was reported as replaced")
	}
}

func TestDebugInputCarriesTheRunningServerAndTheDemoMarker(t *testing.T) {
	status := Status{
		Supported: true, Installed: true, PlistStatus: launchd.PlistCurrent,
		PlistPath: "/tmp/com.user.prx.plist", LogPath: "/tmp/serve.log", Running: true, BinaryMatches: true,
		State: runstate.State{PID: 4242, Address: "127.0.0.1:7331", Version: "1.2.3"},
	}
	input := status.DebugInput(false)
	if input.PID != 4242 || input.Address != "127.0.0.1:7331" || input.Version != "1.2.3" {
		t.Fatalf("debug input=%+v", input)
	}
	if !input.Installed || input.PlistStatus != string(launchd.PlistCurrent) || !input.BinaryMatches {
		t.Fatalf("debug input=%+v", input)
	}
	if !status.DebugInput(true).Demo {
		t.Fatal("the demo marker was lost")
	}
}

// TestInspectReadsThePlistAndTheRunState は 3 つの観測結果を確かめる。稼働の判定は
// 記録の内容ではなくロックが決めるので、内容のない記録も稼働として扱う。
func TestInspectReadsThePlistAndTheRunState(t *testing.T) {
	manager := launchd.New()
	if !manager.Supported() {
		t.Skip("LaunchAgent management is only supported on macOS")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(runstate.DirEnvironmentVariable, filepath.Join(home, "run"))
	status := Inspect(manager, "1.2.3")
	if !status.Supported || status.Installed || status.Running || status.Error != "" {
		t.Fatalf("status=%+v", status)
	}
	if status.PlistPath == "" || status.LogPath == "" || status.PlistStatus != launchd.PlistUnknown {
		t.Fatalf("status=%+v", status)
	}
	lock, held, err := runstate.Acquire()
	if err != nil || !held {
		t.Fatalf("acquire held=%v err=%v", held, err)
	}
	defer func() { _ = lock.Release() }()
	if status = Inspect(manager, "1.2.3"); !status.Running || !status.AddressUnknown {
		t.Fatalf("status before bind=%+v", status)
	}
	executable := filepath.Join(home, "prx")
	if err := os.WriteFile(executable, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := lock.Write(runstate.State{
		PID: os.Getpid(), Address: "127.0.0.1:7331", URL: "http://127.0.0.1:7331",
		StartedAt: time.Now().UTC().Add(time.Hour), Version: "1.2.3", Executable: executable,
	}); err != nil {
		t.Fatal(err)
	}
	status = Inspect(manager, "1.2.3")
	if !status.Running || status.AddressUnknown || !status.BinaryMatches {
		t.Fatalf("status after bind=%+v", status)
	}
	if status.State.Address != "127.0.0.1:7331" {
		t.Fatalf("status=%+v", status)
	}
	if status = Inspect(manager, "1.2.4"); status.BinaryMatches {
		t.Fatalf("an older server was reported as matching: %+v", status)
	}
}
