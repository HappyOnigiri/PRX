package filepicker

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type exitError int

func (e exitError) Error() string { return "exit error" }
func (e exitError) ExitCode() int { return int(e) }

func TestPickerCommandsAndResults(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		output      string
		runErr      error
		wantPath    string
		wantCancel  bool
		wantCommand string
	}{
		{
			name:        "macOS path",
			goos:        "darwin",
			output:      "/tmp/plan.md\n",
			wantPath:    "/tmp/plan.md",
			wantCommand: "osascript",
		},
		{
			name:        "macOS cancellation",
			goos:        "darwin",
			output:      "execution error: User canceled. (-128)",
			runErr:      exitError(1),
			wantCancel:  true,
			wantCommand: "osascript",
		},
		{
			name:        "Windows path",
			goos:        "windows",
			output:      "C:\\work\\plan.md\r\n",
			wantPath:    "C:\\work\\plan.md",
			wantCommand: "powershell.exe",
		},
		{name: "Windows cancellation", goos: "windows", wantCancel: true, wantCommand: "powershell.exe"},
		{
			name:        "Linux path",
			goos:        "linux",
			output:      "/tmp/plan.md\n",
			wantPath:    "/tmp/plan.md",
			wantCommand: "/usr/bin/zenity",
		},
		{
			name:        "Linux cancellation",
			goos:        "linux",
			runErr:      exitError(1),
			wantCancel:  true,
			wantCommand: "/usr/bin/zenity",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var command string
			picker := &Picker{
				goos: test.goos,
				lookPath: func(name string) (string, error) {
					if name == "zenity" {
						return "/usr/bin/zenity", nil
					}
					return "", errors.New("missing")
				},
				run: func(_ context.Context, name string, _ ...string) ([]byte, error) {
					command = name
					return []byte(test.output), test.runErr
				},
			}
			path, canceled, err := picker.SelectFile(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if path != test.wantPath || canceled != test.wantCancel || command != test.wantCommand {
				t.Fatalf("path=%q canceled=%v command=%q", path, canceled, command)
			}
		})
	}
}

func TestPickerLinuxFallbackAndUnavailablePlatforms(t *testing.T) {
	lookups := []string{}
	picker := &Picker{
		goos: "linux",
		lookPath: func(name string) (string, error) {
			lookups = append(lookups, name)
			if name == "kdialog" {
				return "/usr/bin/kdialog", nil
			}
			return "", errors.New("missing")
		},
		run: func(_ context.Context, name string, _ ...string) ([]byte, error) {
			if name != "/usr/bin/kdialog" {
				t.Fatalf("command=%q", name)
			}
			return []byte("/tmp/spec.md"), nil
		},
	}
	path, canceled, err := picker.SelectFile(context.Background())
	if err != nil || canceled || path != "/tmp/spec.md" {
		t.Fatalf("path=%q canceled=%v err=%v", path, canceled, err)
	}
	if !reflect.DeepEqual(lookups, []string{"zenity", "kdialog"}) {
		t.Fatalf("lookups=%v", lookups)
	}

	picker.lookPath = func(string) (string, error) { return "", errors.New("missing") }
	_, _, err = picker.SelectFile(context.Background())
	assertPickerError(t, err, KindUnavailable)
	picker.goos = "plan9"
	_, _, err = picker.SelectFile(context.Background())
	assertPickerError(t, err, KindUnavailable)
}

func TestPickerRejectsFailuresAndInvalidPaths(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		output string
		runErr error
		kind   Kind
	}{
		{name: "start failure", goos: "darwin", runErr: errors.New("missing command"), kind: KindFailed},
		{
			name:   "non-cancel exit",
			goos:   "darwin",
			output: "execution error: broken (-1)",
			runErr: exitError(1),
			kind:   KindFailed,
		},
		{name: "wrong exit code", goos: "linux", runErr: exitError(2), kind: KindFailed},
		{name: "relative Unix path", goos: "darwin", output: "notes.md", kind: KindInvalidResult},
		{name: "relative Windows path", goos: "windows", output: "notes.md", kind: KindInvalidResult},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			picker := &Picker{
				goos:     test.goos,
				lookPath: func(string) (string, error) { return "/usr/bin/zenity", nil },
				run:      func(context.Context, string, ...string) ([]byte, error) { return []byte(test.output), test.runErr },
			}
			_, _, err := picker.SelectFile(context.Background())
			assertPickerError(t, err, test.kind)
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	picker := &Picker{
		goos: "darwin",
		run:  func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("killed") },
	}
	_, _, err := picker.SelectFile(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestAbsolutePathVariantsAndConstructor(t *testing.T) {
	if New() == nil {
		t.Fatal("New returned nil")
	}
	for _, test := range []struct {
		goos, path string
		want       bool
	}{
		{"linux", "/tmp/a", true},
		{"linux", "tmp/a", false},
		{"windows", `C:\\tmp\\a`, true},
		{"windows", `d:/tmp/a`, true},
		{"windows", `\\\\server\\share\\a`, true},
		{"windows", `C:a`, false},
	} {
		if got := isAbsolute(test.goos, test.path); got != test.want {
			t.Fatalf("isAbsolute(%q, %q)=%v", test.goos, test.path, got)
		}
	}
}

func assertPickerError(t *testing.T, err error, kind Kind) {
	t.Helper()
	var pickerErr *Error
	if !errors.As(err, &pickerErr) || pickerErr.Kind != kind {
		t.Fatalf("err=%T %v", err, err)
	}
	if pickerErr.Error() == "" || errors.Unwrap(pickerErr) == nil {
		t.Fatalf("incomplete error=%v", pickerErr)
	}
}
