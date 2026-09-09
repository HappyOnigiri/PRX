package launchd

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type recordedCommand struct {
	name string
	args []string
}

// newTestManager は実 launchctl と実ホームに触れない manager を組み立てる。
// docs/development.md のとおり、daemon のテストはどちらにも触れてはならない。
func newTestManager(t *testing.T, home string, reply func(args []string) ([]byte, error)) (
	*Manager, *[]recordedCommand,
) {
	t.Helper()
	recorded := &[]recordedCommand{}
	manager := &Manager{
		goos: "darwin",
		run: func(_ context.Context, name string, args ...string) ([]byte, error) {
			*recorded = append(*recorded, recordedCommand{name: name, args: args})
			if reply == nil {
				return nil, nil
			}
			return reply(args)
		},
		lookPath:   func(string) (string, error) { return "", exec.ErrNotFound },
		homeDir:    func() (string, error) { return home, nil },
		executable: func() (string, error) { return filepath.Join(home, "bin", "prx"), nil },
		uid:        501,
		parentPID:  func() int { return 1 },
		getenv:     func(string) string { return Label },
	}
	return manager, recorded
}

func TestRenderCarriesTheServeArgumentAndTheThrottleInterval(t *testing.T) {
	data, err := Render("/usr/local/bin/prx", "/Users/example", "/Users/example/Library/Logs/prx/serve.log")
	if err != nil {
		t.Fatal(err)
	}
	// plist だけが launchd の起動引数を決める。要素名の変更を漏らすと launchd は未知の
	// サブコマンドを KeepAlive のループで回し続け、テストでは検出できない失敗になる。
	for _, argument := range []string{"<string>/usr/local/bin/prx</string>", "<string>serve</string>"} {
		if !strings.Contains(string(data), argument) {
			t.Fatalf("rendered plist does not pass %s: %s", argument, data)
		}
	}
	for _, forbidden := range []string{"--addr", "--demo"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("rendered plist passes %s: %s", forbidden, data)
		}
	}
	if !strings.Contains(string(data), "<key>ThrottleInterval</key><integer>10</integer>") {
		t.Fatalf("rendered plist does not set ThrottleInterval=10: %s", data)
	}
	for _, key := range []string{"<key>HOME</key>", "<key>PATH</key>", "<key>StandardErrorPath</key>"} {
		if !strings.Contains(string(data), key) {
			t.Fatalf("rendered plist lacks %s: %s", key, data)
		}
	}
}

func TestRenderEscapesAndPreservesSpecialCharacterPaths(t *testing.T) {
	binary := `/tmp/prx & <binary> "quoted"`
	home := `/tmp/home & <user> "quoted"`
	logPath := `/tmp/logs & <prx> "serve".log`
	data, err := Render(binary, home, logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "&amp;") || !strings.Contains(string(data), "&lt;") {
		t.Fatalf("special characters were not XML escaped: %s", data)
	}
	values := make([]string, 0)
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, decodeErr := decoder.Token()
		if errors.Is(decodeErr, io.EOF) {
			break
		}
		if decodeErr != nil {
			t.Fatalf("rendered plist is not well-formed XML: %v", decodeErr)
		}
		if text, ok := token.(xml.CharData); ok {
			values = append(values, string(text))
		}
	}
	joined := strings.Join(values, "\n")
	expectedPath := filepath.Join(home, ".local", "bin") + ":/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
	for _, expected := range []string{binary, home, expectedPath, logPath} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("decoded plist does not preserve %q: %s", expected, joined)
		}
	}
}

func TestInstallWritesAPrivatePlistAndBootstrapsIt(t *testing.T) {
	home := t.TempDir()
	manager, recorded := newTestManager(t, home, nil)
	if err := manager.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	path, err := manager.PlistPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("plist info=%v err=%v", info, err)
	}
	logPath, err := manager.LogPath()
	if err != nil {
		t.Fatal(err)
	}
	if directory, err := os.Stat(filepath.Dir(logPath)); err != nil || directory.Mode().Perm() != 0o700 {
		t.Fatalf("log directory info=%v err=%v", directory, err)
	}
	if len(*recorded) != 2 || (*recorded)[0].args[0] != "bootout" || (*recorded)[1].args[0] != "bootstrap" {
		t.Fatalf("launchctl calls=%+v", *recorded)
	}
	if want := fmt.Sprintf("gui/%d", manager.uid); (*recorded)[1].args[1] != want {
		t.Fatalf("bootstrap domain=%q, want %q", (*recorded)[1].args[1], want)
	}
	if status, err := manager.PlistStatus(); status != PlistCurrent || err != nil {
		t.Fatalf("status after install=%v err=%v", status, err)
	}
	if err := os.WriteFile(path, []byte("<plist/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if status, err := manager.PlistStatus(); status != PlistStale || err != nil {
		t.Fatalf("status after rewrite=%v err=%v", status, err)
	}
	if err := manager.Uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("plist remains after uninstall: %v", err)
	}
	if status, err := manager.PlistStatus(); status != PlistUnknown || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("status after uninstall=%v err=%v", status, err)
	}
}

// TestPlistStatusRefusesAnUnsafePlist は symlink を Unknown + エラーにする。install が
// 任意のファイルを 0600 で上書きしないための入口の検査である。
func TestPlistStatusRefusesAnUnsafePlist(t *testing.T) {
	home := t.TempDir()
	manager, _ := newTestManager(t, home, nil)
	path, err := manager.PlistPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, "secret")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	status, err := manager.PlistStatus()
	var typed *Error
	if status != PlistUnknown || !errors.As(err, &typed) || typed.Kind != KindUnsafe {
		t.Fatalf("symlink status=%v err=%v", status, err)
	}
	if err := manager.Install(context.Background()); !errors.As(err, &typed) || typed.Kind != KindUnsafe {
		t.Fatalf("install over a symlink err=%v", err)
	}
	if data, readErr := os.ReadFile(target); readErr != nil || string(data) != "secret" {
		t.Fatalf("symlink target was overwritten: %q %v", data, readErr)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if status, err := manager.PlistStatus(); status != PlistUnknown || !errors.As(err, &typed) ||
		typed.Kind != KindUnsafe {
		t.Fatalf("directory status=%v err=%v", status, err)
	}
}

// TestStartAsksLaunchdWithoutKillingTheRunningServer は Start と Kickstart の -k の有無を
// 確認する。`prx daemon start` が稼働中のサーバーを終了させないための区別である。
func TestStartAsksLaunchdWithoutKillingTheRunningServer(t *testing.T) {
	manager, recorded := newTestManager(t, t.TempDir(), nil)
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := manager.Kickstart(context.Background()); err != nil {
		t.Fatal(err)
	}
	target := fmt.Sprintf("gui/%d/%s", manager.uid, Label)
	if len(*recorded) != 2 {
		t.Fatalf("launchctl calls=%+v", *recorded)
	}
	if got := (*recorded)[0].args; len(got) != 2 || got[0] != "kickstart" || got[1] != target {
		t.Fatalf("start argv=%q, want [kickstart %s]", got, target)
	}
	if got := (*recorded)[1].args; len(got) != 3 || got[1] != "-k" || got[2] != target {
		t.Fatalf("kickstart argv=%q, want [kickstart -k %s]", got, target)
	}
}

func TestOperationsReportAnUninstalledServiceAndOtherFailures(t *testing.T) {
	manager, _ := newTestManager(t, t.TempDir(), func([]string) ([]byte, error) {
		return []byte("Could not find service"), errors.New("exit status 3")
	})
	for name, err := range map[string]error{
		"start":     manager.Start(context.Background()),
		"kickstart": manager.Kickstart(context.Background()),
	} {
		if !errors.Is(err, ErrServiceMissing) {
			t.Fatalf("%s error=%v, want ErrServiceMissing", name, err)
		}
		if strings.Contains(err.Error(), "Could not find service") {
			t.Fatalf("%s error leaks the launchctl output: %v", name, err)
		}
	}
	// 未登録の bootout はエラーになるのが正常なので、Uninstall は plist を消して成功する。
	path, err := manager.PlistPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("plist"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.Uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("plist remains after uninstalling a missing service: %v", err)
	}
	missingDirectory, _ := newTestManager(t, t.TempDir(), nil)
	if err := missingDirectory.Uninstall(context.Background()); err != nil {
		t.Fatalf("uninstall without a LaunchAgents directory: %v", err)
	}

	failing, _ := newTestManager(t, t.TempDir(), func([]string) ([]byte, error) {
		return []byte("failed"), errors.New("exit status 9")
	})
	if err := failing.Install(context.Background()); err == nil {
		t.Fatal("failed bootstrap succeeded")
	}
	if err := failing.Uninstall(context.Background()); err == nil {
		t.Fatal("failed bootout succeeded")
	}
	if err := failing.Start(context.Background()); errors.Is(err, ErrServiceMissing) || err == nil {
		t.Fatalf("failed start error=%v", err)
	}
}

func TestUnsupportedOperatingSystemsRefuseEveryOperation(t *testing.T) {
	manager, recorded := newTestManager(t, t.TempDir(), nil)
	manager.goos = "linux"
	if manager.Supported() || manager.Managed() {
		t.Fatal("linux reports LaunchAgent support")
	}
	for name, err := range map[string]error{
		"install":   manager.Install(context.Background()),
		"uninstall": manager.Uninstall(context.Background()),
		"start":     manager.Start(context.Background()),
		"kickstart": manager.Kickstart(context.Background()),
	} {
		if !errors.Is(err, ErrUnsupported) {
			t.Fatalf("%s error=%v, want ErrUnsupported", name, err)
		}
	}
	if status, err := manager.PlistStatus(); status != PlistUnknown || !errors.Is(err, ErrUnsupported) {
		t.Fatalf("status=%v err=%v", status, err)
	}
	if len(*recorded) != 0 {
		t.Fatalf("unsupported operations ran launchctl: %+v", *recorded)
	}
}

// TestNewWiresTheRealEnvironment は New() が本物の exec と環境の読み取りを組み立てて
// いることを、PATH に置いた stub launchctl で実証する。他のケースは注入した runner で
// argv を検証しており、この 1 本だけが配線そのものを確かめる。
func TestNewWiresTheRealEnvironment(t *testing.T) {
	if !New().Supported() {
		t.Skip("LaunchAgent management is only supported on macOS")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	directory := filepath.Join(home, "bin")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(home, "argv")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> " + record + "\nexit 0\n"
	if err := os.WriteFile(filepath.Join(directory, "launchctl"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	manager := New()
	path, err := manager.PlistPath()
	if err != nil || !strings.HasPrefix(path, home) {
		t.Fatalf("plist path=%q err=%v", path, err)
	}
	if err := manager.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	argv, err := os.ReadFile(record)
	if err != nil || !strings.Contains(string(argv), "bootstrap") {
		t.Fatalf("launchctl argv=%q err=%v", argv, err)
	}
	if status, err := manager.PlistStatus(); status != PlistCurrent || err != nil {
		t.Fatalf("status=%v err=%v", status, err)
	}
	// os.Executable へのフォールバックは PATH に prx がないときに使われる。
	binary, err := manager.ResolveBinary()
	if err != nil || binary == "" {
		t.Fatalf("binary=%q err=%v", binary, err)
	}
	if err := manager.Uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	if manager.Managed() {
		t.Fatal("a test process reports itself as managed by launchd")
	}
}

func TestResolveBinaryPrefersThePathCommand(t *testing.T) {
	manager, _ := newTestManager(t, t.TempDir(), nil)
	manager.lookPath = func(name string) (string, error) {
		if name != "prx" {
			return "", exec.ErrNotFound
		}
		return "/usr/local/bin/prx", nil
	}
	if binary, err := manager.ResolveBinary(); err != nil || binary != "/usr/local/bin/prx" {
		t.Fatalf("binary=%q err=%v", binary, err)
	}
	manager.lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	manager.executable = func() (string, error) { return "", errors.New("no executable") }
	if _, err := manager.ResolveBinary(); err == nil {
		t.Fatal("resolve succeeded without an executable")
	}
	manager.homeDir = func() (string, error) { return "", errors.New("no home") }
	if _, err := manager.PlistPath(); err == nil {
		t.Fatal("plist path succeeded without a home directory")
	}
	if _, err := manager.LogPath(); err == nil {
		t.Fatal("log path succeeded without a home directory")
	}
}
