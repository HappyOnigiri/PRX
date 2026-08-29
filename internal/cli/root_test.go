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
	err := Execute(context.Background(), []string{"--json", "unknown"}, failingWriter{}, &errOut)
	if err == nil {
		t.Fatal("expected command error")
	}
	if !strings.Contains(errOut.String(), "error: unknown command") {
		t.Fatalf("stderr=%q", errOut.String())
	}
}

func TestCommandsHaveDocumentation(t *testing.T) {
	root := NewRoot(io.Discard, io.Discard)
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
