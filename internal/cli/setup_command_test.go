package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/runstate"
	"github.com/HappyOnigiri/PRX/internal/tui"
)

func TestSetupWithoutTerminalDoesNotOpenService(t *testing.T) {
	previousTerminal := setupIsTerminal
	previousOpenTTY := setupOpenTTY
	setupIsTerminal = func(int) bool { return false }
	setupOpenTTY = func() (*os.File, error) { return nil, errors.New("no tty") }
	t.Cleanup(func() {
		setupIsTerminal = previousTerminal
		setupOpenTTY = previousOpenTTY
	})

	opened := false
	var errOut bytes.Buffer
	err := Execute(context.Background(), []string{"setup"}, io.Discard, &errOut,
		func(context.Context, ServiceOptions) (Service, io.Closer, error) {
			opened = true
			return nil, nil, nil
		},
	)
	if runtime.GOOS == "darwin" {
		if err == nil || !strings.Contains(err.Error(), "needs a terminal") {
			t.Fatalf("error=%v", err)
		}
	} else if err != nil {
		t.Fatalf("error=%v", err)
	}
	if opened {
		t.Fatal("setup opened the application service")
	}
}

func TestSetupRejectsJSONOutput(t *testing.T) {
	var errOut bytes.Buffer
	err := Execute(context.Background(), []string{"--json", "setup"}, io.Discard, &errOut, testOpenService)
	if err == nil || !strings.Contains(err.Error(), "does not support --json") {
		t.Fatalf("error=%v", err)
	}
	if !strings.Contains(errOut.String(), `"code":"`) {
		t.Fatalf("stderr=%q", errOut.String())
	}
}

func TestSelectSetupActionUsesTUISelection(t *testing.T) {
	selection := tui.Selection{
		Title: "Choose", Initial: 0,
		Options: []tui.Option{{Value: "install", Label: "Install"}, {Value: "skip", Label: "Skip"}},
	}
	got, err := selectSetupAction(context.Background(), setupSession{
		input: strings.NewReader("\x1b[B\r"), errOut: io.Discard,
	}, selection)
	if err != nil || got != setupSkip {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestSetupOffersBrowserOnlyAfterInstallingDaemon(t *testing.T) {
	tests := []struct {
		name         string
		installedNow bool
		url          string
		want         bool
	}{
		{name: "new daemon", installedNow: true, url: "http://127.0.0.1:7331", want: true},
		{name: "started daemon", installedNow: false, url: "http://127.0.0.1:7331", want: false},
		{name: "no address", installedNow: true, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldOfferSetupOpen(test.installedNow, runstate.State{URL: test.url}); got != test.want {
				t.Fatalf("offer=%t, want %t", got, test.want)
			}
		})
	}
}
