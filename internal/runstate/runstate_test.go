package runstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPathFollowsTheOverrideAndTheConfigDirectory(t *testing.T) {
	override := t.TempDir()
	t.Setenv(DirEnvironmentVariable, override)
	path, err := Path()
	if err != nil || path != filepath.Join(override, "serve.json") {
		t.Fatalf("path=%q err=%v", path, err)
	}
	t.Setenv(DirEnvironmentVariable, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err = Path()
	if err != nil {
		t.Fatal(err)
	}
	if dir, dirErr := Dir(); dirErr != nil || filepath.Dir(path) != dir {
		t.Fatalf("dir=%q err=%v path=%q", dir, dirErr, path)
	}
	if want := filepath.Join("prx", "run", "serve.json"); filepath.Base(filepath.Dir(filepath.Dir(path))) != "prx" {
		t.Fatalf("path=%q, want it to end with %q", path, want)
	}
}

// TestReadTrustsTheLockRatherThanTheContent は残された内容を稼働の根拠にしないことを
// 確かめる。異常終了で残った古いアドレスを `prx open` が開くと、たまたまその番号を
// 掴んだ別プロセスに繋がる。
func TestReadTrustsTheLockRatherThanTheContent(t *testing.T) {
	t.Setenv(DirEnvironmentVariable, t.TempDir())
	if status, _, err := Read(); status != StatusNotRunning || err != nil {
		t.Fatalf("missing file status=%v err=%v", status, err)
	}
	lock, held, err := Acquire()
	if err != nil || !held {
		t.Fatalf("acquire held=%v err=%v", held, err)
	}
	if status, _, readErr := Read(); status != StatusRunningAddressUnknown || readErr != nil {
		t.Fatalf("empty content status=%v err=%v", status, readErr)
	}
	started := time.Now().UTC().Truncate(time.Second)
	want := State{
		PID: os.Getpid(), Address: "127.0.0.1:7331", URL: "http://127.0.0.1:7331", StartedAt: started,
		Version: "1.2.3", Executable: "/usr/local/bin/prx", DatabasePath: "/tmp/prx.db", LaunchdManaged: true,
	}
	if err := lock.Write(want); err != nil {
		t.Fatal(err)
	}
	status, got, err := Read()
	if err != nil || status != StatusRunning {
		t.Fatalf("written status=%v err=%v", status, err)
	}
	want.SchemaVersion = SchemaVersion
	if got != want {
		t.Fatalf("state=%+v, want %+v", got, want)
	}
	// 内容を短い記録で上書きしても、in-place の Truncate が前の内容の残骸を残さない。
	if err := lock.Write(State{PID: 1, Address: "127.0.0.1:1", URL: "http://127.0.0.1:1"}); err != nil {
		t.Fatal(err)
	}
	if _, got, err = Read(); err != nil || got.Address != "127.0.0.1:1" || got.Version != "" {
		t.Fatalf("rewritten state=%+v err=%v", got, err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if status, _, err := Read(); status != StatusNotRunning || err != nil {
		t.Fatalf("released status=%v err=%v", status, err)
	}
}

func TestReadReportsRunningWhenTheContentIsCorrupt(t *testing.T) {
	t.Setenv(DirEnvironmentVariable, t.TempDir())
	lock, held, err := Acquire()
	if err != nil || !held {
		t.Fatalf("acquire held=%v err=%v", held, err)
	}
	defer func() { _ = lock.Release() }()
	if err := os.WriteFile(lock.Path(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if status, state, readErr := Read(); status != StatusRunningAddressUnknown || state.PID != 0 || readErr != nil {
		t.Fatalf("corrupt status=%v state=%+v err=%v", status, state, readErr)
	}
}

func TestAcquireCorrectsPermissionsAndRefusesASecondHolder(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "run")
	t.Setenv(DirEnvironmentVariable, directory)
	path := filepath.Join(directory, "serve.json")
	if err := os.MkdirAll(directory, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o666); err != nil {
		t.Fatal(err)
	}
	lock, held, err := Acquire()
	if err != nil || !held {
		t.Fatalf("acquire held=%v err=%v", held, err)
	}
	defer func() { _ = lock.Release() }()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("run state mode=%v err=%v", info, err)
	}
	// 同一プロセス内の 2 度目の Acquire は別の fd を開くので、flock は保持者を 1 つに絞る。
	second, secondHeld, err := Acquire()
	if err != nil || secondHeld {
		t.Fatalf("second acquire held=%v err=%v", secondHeld, err)
	}
	if second != nil {
		t.Fatal("second acquire returned a lock")
	}
}
