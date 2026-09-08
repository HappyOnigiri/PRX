// Package browser はローカルの WebUI を OS の既定のブラウザで開く。
package browser

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

// ErrUnsupported はブラウザを開けない OS を表す。
var ErrUnsupported = errors.New("opening a browser requires macOS")

// Kind は、コマンド出力を晒さずに呼び出し側が提示できる失敗の分類。
type Kind string

const (
	KindUnsupported Kind = "unsupported"
	KindFailed      Kind = "failed"
)

// Error はブラウザ起動の失敗を表す。
type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string { return fmt.Sprintf("open browser %s: %v", e.Kind, e.Err) }

func (e *Error) Unwrap() error { return e.Err }

// Opener は 1 つの OS 向けにブラウザを開く。
type Opener struct {
	goos string
	run  func(context.Context, string, ...string) ([]byte, error)
}

// New は現在の OS 向けの opener を返す。
func New() *Opener {
	return &Opener{
		goos: runtime.GOOS,
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
	}
}

// Supported はブラウザを開ける OS かを返す。
func (o *Opener) Supported() bool { return o.goos == "darwin" }

// Open は URL を既定のブラウザで開く。
func (o *Opener) Open(ctx context.Context, url string) error {
	if !o.Supported() {
		return &Error{Kind: KindUnsupported, Err: ErrUnsupported}
	}
	if _, err := o.run(ctx, "open", url); err != nil {
		return &Error{Kind: KindFailed, Err: err}
	}
	return nil
}
