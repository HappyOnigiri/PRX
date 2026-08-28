package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
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
