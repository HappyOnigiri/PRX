package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestExecuteWritesJSONErrorToStderrWhenStdoutFails(t *testing.T) {
	var errOut bytes.Buffer
	err := Execute(context.Background(), []string{"--json", "unknown"}, failingWriter{}, &errOut, testOpenService)
	if err == nil {
		t.Fatal("expected command error")
	}
	if !strings.Contains(errOut.String(), `"code":"usage_error"`) ||
		!strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("stderr=%q", errOut.String())
	}
}

func testOpenService(context.Context, string, string, bool) (Service, io.Closer, error) {
	return nil, nil, nil
}

func TestCommandsHaveDocumentation(t *testing.T) {
	root := NewRoot(io.Discard, io.Discard, testOpenService)
	var visit func(*cobra.Command, bool)
	visit = func(command *cobra.Command, excluded bool) {
		excluded = excluded || command.Hidden || command.Name() == "help" || command.Name() == "completion"
		if !excluded {
			if strings.TrimSpace(command.Short) == "" {
				t.Errorf("command %q has no Short description", command.CommandPath())
			}
			if (command.Run != nil || command.RunE != nil) && strings.TrimSpace(command.Example) == "" {
				t.Errorf("command %q has no Example", command.CommandPath())
			}
		}
		for _, child := range command.Commands() {
			visit(child, excluded)
		}
	}
	visit(root, false)
}

type recordingCloser struct {
	closed bool
}

func (c *recordingCloser) Close() error {
	c.closed = true
	return nil
}

func TestRootOpensAndClosesCommandResources(t *testing.T) {
	closer := &recordingCloser{}
	var gotDBPath, gotFixture string
	var gotLive bool
	root, state := newRootWithState(
		io.Discard,
		io.Discard,
		func(_ context.Context, dbPath, fixturePath string, live bool) (Service, io.Closer, error) {
			gotDBPath = dbPath
			gotFixture = fixturePath
			gotLive = live
			return nil, closer, nil
		},
	)
	state.dbPath = "test.db"
	state.fixture = "demo"

	var serve *cobra.Command
	for _, command := range root.Commands() {
		if command.Name() == "serve" {
			serve = command
			break
		}
	}
	if serve == nil {
		t.Fatal("serve command was not registered")
	}
	if err := root.PersistentPreRunE(serve, nil); err != nil {
		t.Fatal(err)
	}
	if gotDBPath != "test.db" || gotFixture != "demo" || !gotLive {
		t.Fatalf("open args = (%q, %q, %t)", gotDBPath, gotFixture, gotLive)
	}
	root.PersistentPostRun(serve, nil)
	if !closer.closed {
		t.Fatal("expected command resource to be closed")
	}
}

func TestRootHelpDoesNotOpenService(t *testing.T) {
	opened := false
	root := NewRoot(io.Discard, io.Discard, func(context.Context, string, string, bool) (Service, io.Closer, error) {
		opened = true
		return nil, nil, nil
	})
	root.SetArgs([]string{"help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if opened {
		t.Fatal("help should not open the database")
	}
}

func TestRootVersionDoesNotOpenService(t *testing.T) {
	var out bytes.Buffer
	opened := false
	err := Execute(context.Background(), []string{"--version"}, &out, io.Discard,
		func(context.Context, string, string, bool) (Service, io.Closer, error) {
			opened = true
			return nil, nil, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if opened {
		t.Fatal("version should not open the database")
	}
	root := NewRoot(io.Discard, io.Discard, testOpenService)
	if got, want := out.String(), "prx version "+root.Version+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
