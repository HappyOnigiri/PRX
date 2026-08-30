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

func TestCommandTreeUsesCanonicalReadSyntax(t *testing.T) {
	root := NewRoot(io.Discard, io.Discard, testOpenService)
	for _, removed := range []string{"node", "implementation-plan"} {
		if command, _, err := root.Find([]string{removed}); err == nil && command != root {
			t.Fatalf("removed command %q is still registered as %q", removed, command.CommandPath())
		}
	}
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command != root && (command.Name() == "list" || command.Name() == "get") {
			t.Errorf("removed read verb remains registered: %s", command.CommandPath())
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(root)

	for alias, canonical := range map[string]string{"f": "feature", "t": "task", "dep": "dependency", "doc": "document"} {
		command, _, err := root.Find([]string{alias})
		if err != nil || command.Name() != canonical {
			t.Errorf("alias %q resolved to %v, err=%v", alias, command, err)
		}
	}
}

// The long description spells out the aliases that Cobra also lists on its own,
// so nothing detects a stale description once an alias changes. Keep the two in
// step until one of them owns the text.
func TestCommandDescriptionsMentionTheirAliases(t *testing.T) {
	root := NewRoot(io.Discard, io.Discard, testOpenService)
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if len(command.Aliases) == 0 {
			if strings.Contains(command.Long, "Alias:") {
				t.Errorf("command %q describes an alias it no longer declares", command.CommandPath())
			}
		} else if want := "Alias: " + strings.Join(command.Aliases, ", ") + "."; !strings.Contains(
			command.Long,
			want,
		) {
			t.Errorf("command %q long description lacks %q", command.CommandPath(), want)
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(root)
}

func TestPreScanOutputFlagsRespectsValuesAndBoundary(t *testing.T) {
	for _, test := range []struct {
		name      string
		args      []string
		wantJSON  bool
		wantHuman bool
	}{
		{name: "json after unknown command", args: []string{"unknown", "--json"}, wantJSON: true},
		{name: "JSON after unknown local flag", args: []string{"unknown", "--title", "--json"}, wantJSON: true},
		{name: "boolean values", args: []string{"--json=false", "--human=true"}, wantHuman: true},
		{name: "literal flag value", args: []string{"--db", "--json", "unknown"}},
		{name: "literal local flag value", args: []string{"feature", "create", "--title", "--json"}},
		{name: "double dash boundary", args: []string{"unknown", "--", "--json"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, state := newRootWithState(io.Discard, io.Discard, testOpenService)
			state.preScanOutputFlags(root, test.args)
			if state.json != test.wantJSON || state.human != test.wantHuman {
				t.Fatalf("flags=(json=%t human=%t), want (%t %t)",
					state.json, state.human, test.wantJSON, test.wantHuman)
			}
		})
	}
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
