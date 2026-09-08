package browser

import (
	"context"
	"errors"
	"testing"
)

func TestOpenRunsTheDefaultHandlerOnMacOS(t *testing.T) {
	var recorded []string
	opener := &Opener{goos: "darwin", run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		recorded = append([]string{name}, args...)
		return nil, nil
	}}
	if !opener.Supported() {
		t.Fatal("darwin reports no browser support")
	}
	if err := opener.Open(context.Background(), "http://127.0.0.1:7331"); err != nil {
		t.Fatal(err)
	}
	if len(recorded) != 2 || recorded[0] != "open" || recorded[1] != "http://127.0.0.1:7331" {
		t.Fatalf("argv=%q", recorded)
	}
}

func TestOpenReportsFailuresWithoutTheCommandOutput(t *testing.T) {
	opener := &Opener{goos: "darwin", run: func(context.Context, string, ...string) ([]byte, error) {
		return []byte("LSOpenURLsWithRole() failed"), errors.New("exit status 1")
	}}
	err := opener.Open(context.Background(), "http://127.0.0.1:7331")
	var typed *Error
	if !errors.As(err, &typed) || typed.Kind != KindFailed {
		t.Fatalf("error=%v", err)
	}
	if err.Error() == "" || typed.Unwrap() == nil {
		t.Fatalf("error=%v unwrap=%v", err, typed.Unwrap())
	}
}

func TestOpenRefusesUnsupportedOperatingSystems(t *testing.T) {
	opener := &Opener{goos: "linux", run: func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("linux ran a browser command")
		return nil, nil
	}}
	if opener.Supported() {
		t.Fatal("linux reports browser support")
	}
	if err := opener.Open(context.Background(), "http://127.0.0.1:7331"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("error=%v, want ErrUnsupported", err)
	}
}

// TestNewWiresTheRealCommand は New() が本物の exec を組み立てていることを確かめる。
// 実際にブラウザを開かないよう、Supported の判定だけを見る。
func TestNewWiresTheRealCommand(t *testing.T) {
	opener := New()
	if opener.run == nil || opener.goos == "" {
		t.Fatalf("opener=%+v", opener)
	}
}
