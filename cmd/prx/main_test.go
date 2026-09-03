package main

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/cli"
	"github.com/HappyOnigiri/PRX/internal/config"
)

func TestDemoServiceIsIsolatedPersistsUntilCloseAndResets(t *testing.T) {
	realRoot := t.TempDir()
	realDB := filepath.Join(realRoot, "real.db")
	realConfig := filepath.Join(realRoot, "real.yaml")
	if err := os.WriteFile(realDB, []byte("real database sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(realConfig, []byte("real config sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := config.WithPath(context.Background(), realConfig)
	open := newOpenService(io.Discard)

	service, closer, err := open(ctx, cli.ServiceOptions{
		DatabasePath: realDB,
		FixturePath:  filepath.Join(realRoot, "missing-fixture.json"),
		Live:         true,
		Demo:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	temporaryRoot := closer.(*serviceCloser).temporaryRoot
	assertDemoCounts(t, service, 4, 120)
	if _, err := service.CreateFeature(ctx, "session-change", "Session change", "", ""); err != nil {
		t.Fatal(err)
	}
	assertDemoCounts(t, service, 5, 120)
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(temporaryRoot); !os.IsNotExist(err) {
		t.Fatalf("temporary demo root still exists: %v", err)
	}
	assertFileContent(t, realDB, "real database sentinel")
	assertFileContent(t, realConfig, "real config sentinel")

	service, closer, err = open(ctx, cli.ServiceOptions{Demo: true, Live: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := closer.Close(); err != nil {
			t.Error(err)
		}
	})
	assertDemoCounts(t, service, 4, 120)
}

func assertDemoCounts(t *testing.T, service cli.Service, features, tasks int) {
	t.Helper()
	snapshot, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Features) != features || len(snapshot.Tasks) != tasks {
		t.Fatalf("features=%d tasks=%d, want %d and %d", len(snapshot.Features), len(snapshot.Tasks), features, tasks)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

// Cobra skips its post-run hooks once a command returns an error, so a serve
// that fails after the demo is built has to release the temporary root itself.
func TestDemoServeFailureRemovesTemporaryRoot(t *testing.T) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	var temporaryRoot string
	open := func(
		ctx context.Context,
		options cli.ServiceOptions,
	) (cli.Service, io.Closer, error) {
		service, closer, err := newOpenService(io.Discard)(ctx, options)
		if err == nil {
			temporaryRoot = closer.(*serviceCloser).temporaryRoot
		}
		return service, closer, err
	}
	err = cli.Execute(
		context.Background(),
		[]string{"serve", "--demo", "--addr", listener.Addr().String()},
		io.Discard,
		io.Discard,
		open,
	)
	if err == nil {
		t.Fatal("serve on a busy address unexpectedly succeeded")
	}
	if temporaryRoot == "" {
		t.Fatal("the demo service was never opened, so this test proves nothing")
	}
	if _, statErr := os.Stat(temporaryRoot); !os.IsNotExist(statErr) {
		t.Errorf("temporary demo root still exists: %v", statErr)
	}
}
