// Package launchd は macOS の LaunchAgent として `prx serve` を常駐させる。plist 全体を
// PRX が所有し、byte 一致でその陳腐化を判定する。docs/design/daemon.md を参照。
package launchd

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// Label は LaunchAgent の識別子で、plist 名と launchctl の対象名の両方に使う。
const Label = "com.user.prx"

// plistTemplate は plist の全体。ProgramArguments が `serve` の 1 要素だけなので、--addr も
// --demo も launchd 経由では渡らない。ThrottleInterval と各キーの理由は
// docs/design/daemon.md を参照。
const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>{{.Label | x}}</string>
<key>ProgramArguments</key>
<array><string>{{.Binary | x}}</string><string>serve</string></array>
<key>RunAtLoad</key><true/>
<key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
<key>ThrottleInterval</key><integer>10</integer>
<key>EnvironmentVariables</key>
<dict>
<key>HOME</key><string>{{.Home | x}}</string>
<key>PATH</key><string>{{.Path | x}}</string>
</dict>
<key>StandardOutPath</key><string>{{.Log | x}}</string>
<key>StandardErrorPath</key><string>{{.Log | x}}</string>
</dict></plist>
`

// ErrUnsupported は LaunchAgent を扱えない OS を表す。コマンドは全 OS で登録するので、
// 対応の有無は実行時にこのエラーで判定する。
var ErrUnsupported = errors.New("the PRX daemon requires macOS")

// ErrServiceMissing は launchctl が PRX の LaunchAgent を認識していないことを示す。
// 呼び出し側は操作不能な launchctl のエラーではなく `prx daemon install` を案内できる。
var ErrServiceMissing = errors.New("the PRX LaunchAgent is not installed")

// Kind は、launchctl の出力を晒さずに呼び出し側が提示できる失敗の分類。
type Kind string

const (
	KindUnsupported Kind = "unsupported"
	KindFailed      Kind = "failed"
	KindUnsafe      Kind = "unsafe_plist"
)

// Error は LaunchAgent の操作の失敗を表す。launchctl の生出力は運ばない。
type Error struct {
	Kind Kind
	Op   string
	Err  error
}

func (e *Error) Error() string { return fmt.Sprintf("launchd %s %s: %v", e.Op, e.Kind, e.Err) }

func (e *Error) Unwrap() error { return e.Err }

// PlistStatus は現在の PRX から LaunchAgent を再生成できるかを示す。plist の不在や
// 読み取り不能は Unknown とし、未導入を stale と誤判定しない。
type PlistStatus string

const (
	PlistUnknown PlistStatus = "unknown"
	PlistCurrent PlistStatus = "current"
	PlistStale   PlistStatus = "stale"
)

type (
	commandRunner func(context.Context, string, ...string) ([]byte, error)
	pathLookup    func(string) (string, error)
)

// Manager は 1 つの OS 向けに LaunchAgent を操作する。外部プロセスと環境の読み取りを
// フィールドに集め、テストが実 launchctl と実ホームに触れずに済むようにする。
type Manager struct {
	goos       string
	run        commandRunner
	lookPath   pathLookup
	homeDir    func() (string, error)
	executable func() (string, error)
	uid        int
	parentPID  func() int
	getenv     func(string) string
}

// New は現在の OS 向けの manager を返す。os/user は使わない。ホームの解決は
// os.UserHomeDir に限り、テストが HOME の差し替えだけで隔離できるようにする。
func New() *Manager {
	return &Manager{
		goos: runtime.GOOS,
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
		lookPath:   exec.LookPath,
		homeDir:    os.UserHomeDir,
		executable: os.Executable,
		uid:        os.Getuid(),
		parentPID:  os.Getppid,
		getenv:     os.Getenv,
	}
}

// Supported は LaunchAgent を扱える OS かを返す。
func (m *Manager) Supported() bool { return m.goos == "darwin" }

// Managed はこのプロセスを launchd が起動したかを返す。管理外のプロセスが自己
// kickstart すると、置換側が flock を取れずに死んで手動プロセスだけが残る。
func (m *Manager) Managed() bool {
	return m.Supported() && m.parentPID() == 1 && m.getenv("XPC_SERVICE_NAME") == Label
}

// PlistPath は LaunchAgent の位置を返す。
func (m *Manager) PlistPath() (string, error) {
	home, err := m.homeDir()
	if err != nil {
		return "", &Error{Kind: KindFailed, Op: "resolve home", Err: err}
	}
	return filepath.Join(home, "Library", "LaunchAgents", Label+".plist"), nil
}

// LogPath は launchd が serve の stdout と stderr を書き込む先を返す。ローテーションは
// 実装していない既知の負債で、docs/design/daemon.md に記録してある。
func (m *Manager) LogPath() (string, error) {
	home, err := m.homeDir()
	if err != nil {
		return "", &Error{Kind: KindFailed, Op: "resolve home", Err: err}
	}
	return filepath.Join(home, "Library", "Logs", "prx", "serve.log"), nil
}

// ResolveBinary は LaunchAgent が起動する prx を返す。PATH 上のコマンドを優先し、
// 開発ビルドやテストバイナリでは os.Executable にフォールバックする。既存 plist の
// 診断も同じ解決を使い、install と判定の対象を揃える。
func (m *Manager) ResolveBinary() (string, error) {
	if binary, err := m.lookPath("prx"); err == nil {
		return binary, nil
	}
	binary, err := m.executable()
	if err != nil {
		return "", &Error{Kind: KindFailed, Op: "resolve executable", Err: err}
	}
	return binary, nil
}

// Render は plist 全体を組み立てる。補間する値はすべて XML エスケープを通すので、
// ホームディレクトリ名に & を含むユーザーでも plist が壊れない。
func Render(binary, home, logPath string) ([]byte, error) {
	parsed, err := template.New("plist").Funcs(template.FuncMap{"x": xmlEscapeText}).Parse(plistTemplate)
	if err != nil {
		return nil, &Error{Kind: KindFailed, Op: "parse plist template", Err: err}
	}
	// launchd の環境は極小で PATH はほぼ空になる。gh と /usr/bin/security を見つけられないと
	// GitHub 認証が daemon でだけ失敗するので、通常のログインシェルと同じ場所を並べる。
	pathValue := filepath.Join(home, ".local", "bin") + ":/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
	values := map[string]string{"Label": Label, "Binary": binary, "Home": home, "Path": pathValue, "Log": logPath}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, values); err != nil {
		return nil, &Error{Kind: KindFailed, Op: "render plist", Err: err}
	}
	return rendered.Bytes(), nil
}

// xmlEscapeText は template FuncMap の要素。補間する全値をエスケープするので、
// 生成後の再解析は不要。
func xmlEscapeText(value string) (string, error) {
	var escaped bytes.Buffer
	if err := xml.EscapeText(&escaped, []byte(value)); err != nil {
		return "", err
	}
	return escaped.String(), nil
}

// PlistStatus はディスク上の LaunchAgent 全体を、現在の install が書く内容と比較する。
// plist 全体を PRX が所有するので byte 単位で比較し、差分は install の再実行で直す。
func (m *Manager) PlistStatus() (PlistStatus, error) {
	if !m.Supported() {
		return PlistUnknown, &Error{Kind: KindUnsupported, Op: "inspect plist", Err: ErrUnsupported}
	}
	path, err := m.PlistPath()
	if err != nil {
		return PlistUnknown, err
	}
	data, err := readRegularFile(path)
	if err != nil {
		return PlistUnknown, err
	}
	expected, err := m.renderCurrent()
	if err != nil {
		return PlistUnknown, err
	}
	if bytes.Equal(data, expected) {
		return PlistCurrent, nil
	}
	return PlistStale, nil
}

// rejectUnsafePlist は symlink と非通常ファイルを拒否する。plist が symlink だと、
// install が任意のファイルを 0600 で上書きし得る。不在は呼び出し側が区別できるよう
// os.ErrNotExist を包んだまま返す。
func rejectUnsafePlist(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return &Error{Kind: KindFailed, Op: "stat plist", Err: err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &Error{Kind: KindUnsafe, Op: "stat plist", Err: errors.New("the LaunchAgent plist is a symlink")}
	}
	if !info.Mode().IsRegular() {
		return &Error{
			Kind: KindUnsafe, Op: "stat plist", Err: errors.New("the LaunchAgent plist is not a regular file"),
		}
	}
	return nil
}

func readRegularFile(path string) ([]byte, error) {
	if err := rejectUnsafePlist(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &Error{Kind: KindFailed, Op: "read plist", Err: err}
	}
	return data, nil
}

func (m *Manager) renderCurrent() ([]byte, error) {
	home, err := m.homeDir()
	if err != nil {
		return nil, &Error{Kind: KindFailed, Op: "resolve home", Err: err}
	}
	binary, err := m.ResolveBinary()
	if err != nil {
		return nil, err
	}
	logPath, err := m.LogPath()
	if err != nil {
		return nil, err
	}
	return Render(binary, home, logPath)
}

// Install は plist を書いて launchctl へ登録する。plist はアトミックに置き換える。
// 部分的に書かれた plist を launchd が読むと、登録できないまま install が成功する。
func (m *Manager) Install(ctx context.Context) error {
	if !m.Supported() {
		return &Error{Kind: KindUnsupported, Op: "install", Err: ErrUnsupported}
	}
	path, err := m.PlistPath()
	if err != nil {
		return err
	}
	logPath, err := m.LogPath()
	if err != nil {
		return err
	}
	data, err := m.renderCurrent()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return &Error{Kind: KindFailed, Op: "create log directory", Err: err}
	}
	if err := writeAtomically(path, data); err != nil {
		return err
	}
	domain := fmt.Sprintf("gui/%d", m.uid)
	// 既存の登録があるときの bootstrap は失敗するので、まず bootout する。未登録の
	// bootout は失敗するのが正常なので、その結果は無視する。
	_, _ = m.run(ctx, "launchctl", "bootout", domain, path)
	if output, err := m.run(ctx, "launchctl", "bootstrap", domain, path); err != nil {
		return m.launchctlError("bootstrap", output, err)
	}
	return nil
}

func writeAtomically(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return &Error{Kind: KindFailed, Op: "create LaunchAgents directory", Err: err}
	}
	if err := rejectUnsafePlist(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".prx-plist-*")
	if err != nil {
		return &Error{Kind: KindFailed, Op: "create plist", Err: err}
	}
	name := temporary.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return &Error{Kind: KindFailed, Op: "write plist", Err: err}
	}
	if err := temporary.Close(); err != nil {
		return &Error{Kind: KindFailed, Op: "write plist", Err: err}
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return &Error{Kind: KindFailed, Op: "restrict plist permissions", Err: err}
	}
	if err := os.Rename(name, path); err != nil {
		return &Error{Kind: KindFailed, Op: "replace plist", Err: err}
	}
	return nil
}

// Uninstall は登録を解除して plist を消す。service が既に居ない場合も成功にする。
func (m *Manager) Uninstall(ctx context.Context) error {
	if !m.Supported() {
		return &Error{Kind: KindUnsupported, Op: "uninstall", Err: ErrUnsupported}
	}
	path, err := m.PlistPath()
	if err != nil {
		return err
	}
	domain := fmt.Sprintf("gui/%d", m.uid)
	output, runErr := m.run(ctx, "launchctl", "bootout", domain, path)
	if runErr != nil && !serviceMissing(output) {
		return m.launchctlError("bootout", output, runErr)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return &Error{Kind: KindFailed, Op: "remove plist", Err: err}
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return &Error{Kind: KindFailed, Op: "open LaunchAgents directory", Err: err}
	}
	defer func() { _ = directory.Close() }()
	if err := directory.Sync(); err != nil {
		return &Error{Kind: KindFailed, Op: "sync LaunchAgents directory", Err: err}
	}
	return nil
}

// Start は未稼働のときだけ launchd に起動を依頼する。-k を使わないので、稼働中の
// サーバーを終了させずに繰り返し実行できる。
func (m *Manager) Start(ctx context.Context) error { return m.kickstart(ctx, "start") }

// Kickstart は稼働中のサーバーを置き換える。launchctl の -k が終了させてから launchd が
// 起動し直す。稼働だけ確かめたいときは、実行中のサーバーを残す Start を使う。
func (m *Manager) Kickstart(ctx context.Context) error { return m.kickstart(ctx, "restart", "-k") }

func (m *Manager) kickstart(ctx context.Context, operation string, flags ...string) error {
	if !m.Supported() {
		return &Error{Kind: KindUnsupported, Op: operation, Err: ErrUnsupported}
	}
	args := append([]string{"kickstart"}, flags...)
	args = append(args, fmt.Sprintf("gui/%d/%s", m.uid, Label))
	output, err := m.run(ctx, "launchctl", args...)
	if err != nil {
		return m.launchctlError(operation, output, err)
	}
	return nil
}

// launchctlError は launchctl の出力を診断にだけ使い、呼び出し側には晒さない。
// service の不在だけは、install を案内できるようセンチネルへ置き換える。
func (m *Manager) launchctlError(operation string, output []byte, err error) error {
	if serviceMissing(output) {
		return &Error{Kind: KindFailed, Op: operation, Err: ErrServiceMissing}
	}
	return &Error{Kind: KindFailed, Op: operation, Err: err}
}

func serviceMissing(output []byte) bool {
	message := strings.ToLower(string(output))
	return strings.Contains(message, "could not find service") ||
		strings.Contains(message, "no such process") ||
		strings.Contains(message, "service not found")
}
