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

func TestExecuteFallsBackWhenJSONErrorCannotBeWritten(t *testing.T) {
	var errOut bytes.Buffer
	err := Execute(context.Background(), []string{"--json", "unknown"}, failingWriter{}, &errOut, testOpenService)
	if err == nil {
		t.Fatal("expected command error")
	}
	if !strings.Contains(errOut.String(), "error: unknown command") {
		t.Fatalf("stderr=%q", errOut.String())
	}
}

func testOpenService(context.Context, string, string, bool) (Service, io.Closer, error) {
	return nil, nil, nil
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
	root, state := newRootWithState(io.Discard, io.Discard, func(_ context.Context, dbPath, fixturePath string, live bool) (Service, io.Closer, error) {
		gotDBPath = dbPath
		gotFixture = fixturePath
		gotLive = live
		return nil, closer, nil
	})
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
