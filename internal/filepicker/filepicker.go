// Package filepicker はローカルの PRX サーバー向けに OS のファイル選択画面を開く。
package filepicker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"unicode"
)

// Kind は、コマンド出力を晒さずに呼び出し側が提示できる失敗の分類。
type Kind string

const (
	KindUnavailable   Kind = "unavailable"
	KindFailed        Kind = "failed"
	KindInvalidResult Kind = "invalid_result"
)

// Error はネイティブの選択画面の失敗を表す。
type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("local file picker %s: %v", e.Kind, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

type (
	commandRunner func(context.Context, string, ...string) ([]byte, error)
	pathLookup    func(string) (string, error)
)

// Picker は 1 つの OS 向けにネイティブの選択画面を開く。
type Picker struct {
	goos     string
	run      commandRunner
	lookPath pathLookup
}

// New は現在の OS 向けの picker を返す。
func New() *Picker {
	return &Picker{
		goos: runtime.GOOS,
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
		lookPath: exec.LookPath,
	}
}

// SelectFile は選択画面を開き、絶対パスを返す。取り消された場合は canceled=true。
func (p *Picker) SelectFile(ctx context.Context) (path string, canceled bool, err error) {
	command, args, cancelCodes, err := p.command()
	if err != nil {
		return "", false, err
	}
	output, runErr := p.run(ctx, command, args...)
	if runErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", false, ctxErr
		}
		if isCanceled(runErr, output, cancelCodes) {
			return "", true, nil
		}
		return "", false, &Error{Kind: KindFailed, Err: runErr}
	}
	path = strings.TrimRight(string(output), "\r\n")
	if path == "" {
		return "", true, nil
	}
	if !isAbsolute(p.goos, path) {
		return "", false, &Error{Kind: KindInvalidResult, Err: errors.New("chooser returned a non-absolute path")}
	}
	return path, false, nil
}

func (p *Picker) command() (string, []string, []int, error) {
	switch p.goos {
	case "darwin":
		return "osascript", []string{
			"-e", `POSIX path of (choose file with prompt "Select a reference file")`,
		}, []int{1}, nil
	case "windows":
		script := "Add-Type -AssemblyName System.Windows.Forms; " +
			"$dialog = New-Object System.Windows.Forms.OpenFileDialog; " +
			"if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { " +
			"[Console]::Out.Write($dialog.FileName) }"
		return "powershell.exe", []string{
			"-NoLogo",
			"-NoProfile",
			"-NonInteractive",
			"-STA",
			"-Command",
			script,
		}, nil, nil
	case "linux":
		if path, err := p.lookPath("zenity"); err == nil {
			return path, []string{"--file-selection", "--title=Select a reference file"}, []int{1}, nil
		}
		if path, err := p.lookPath("kdialog"); err == nil {
			return path, []string{"--getopenfilename", ".", "*", "Select a reference file"}, []int{1}, nil
		}
		return "", nil, nil, &Error{
			Kind: KindUnavailable,
			Err:  errors.New("install zenity or kdialog, or enter the path manually"),
		}
	default:
		return "", nil, nil, &Error{Kind: KindUnavailable, Err: fmt.Errorf("unsupported operating system %q", p.goos)}
	}
}

type exitCoder interface{ ExitCode() int }

func isCanceled(err error, output []byte, codes []int) bool {
	var exitErr exitCoder
	if !errors.As(err, &exitErr) {
		return false
	}
	for _, code := range codes {
		if exitErr.ExitCode() != code {
			continue
		}
		if strings.Contains(string(output), "execution error") &&
			!strings.Contains(string(output), "User canceled") &&
			!strings.Contains(string(output), "(-128)") {
			return false
		}
		return true
	}
	return false
}

func isAbsolute(goos, path string) bool {
	if goos != "windows" {
		return strings.HasPrefix(path, "/")
	}
	if strings.HasPrefix(path, `\\`) {
		return true
	}
	return len(path) >= 3 && unicode.IsLetter(rune(path[0])) && path[1] == ':' && (path[2] == '\\' || path[2] == '/')
}
