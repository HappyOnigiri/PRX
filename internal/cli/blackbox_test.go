package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
)

type resultEnvelope struct {
	SchemaVersion string          `json:"schema_version"`
	OK            bool            `json:"ok"`
	Data          json.RawMessage `json:"data"`
	Error         *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func buildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "prx")
	command := exec.CommandContext(context.Background(), "go", "build", "-o", binary, "../../cmd/prx")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}

func runCLI(t *testing.T, binary, dbPath string, args ...string) (resultEnvelope, string, int) {
	t.Helper()
	base := []string{"--db", dbPath, "--json"}
	command := exec.CommandContext(context.Background(), binary, append(base, args...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exit := 0
	if err != nil {
		var typed *exec.ExitError
		if errors.As(err, &typed) {
			exit = typed.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	var envelope resultEnvelope
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("stdout is not pure JSON: %v\n%s", decodeErr, stdout.String())
	}
	return envelope, stderr.String(), exit
}

func TestBlackBoxJSONCRUDAndCycle(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "blackbox.db")
	feature, stderr, exit := runCLI(t, binary, dbPath, "feature", "create", "--slug", "release", "--title", "Release")
	if exit != 0 || stderr != "" || !feature.OK || feature.SchemaVersion != "1" {
		t.Fatalf("feature result=%+v stderr=%q exit=%d", feature, stderr, exit)
	}
	var featureData struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(feature.Data, &featureData)
	a, _, _ := runCLI(t, binary, dbPath, "task", "create", "--feature", featureData.ID, "--title", "A")
	b, _, _ := runCLI(t, binary, dbPath, "task", "create", "--feature", featureData.ID, "--title", "B")
	var at, bt struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(a.Data, &at)
	_ = json.Unmarshal(b.Data, &bt)
	if value, _, exit := runCLI(t, binary, dbPath, "dependency", "add", at.ID, bt.ID); exit != 0 || !value.OK {
		t.Fatalf("add dependency: %+v", value)
	}
	value, _, exit := runCLI(t, binary, dbPath, "dependency", "add", bt.ID, at.ID)
	if exit == 0 || value.Error == nil || value.Error.Code != "cycle" {
		t.Fatalf("cycle result=%+v exit=%d", value, exit)
	}
	valid, _, exit := runCLI(t, binary, dbPath, "validate")
	if exit != 0 || !valid.OK {
		t.Fatalf("validate result=%+v", valid)
	}
}

func TestBlackBoxServerAndCLIShareDatabase(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "shared.db")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	server := exec.CommandContext(ctx, binary, "--db", dbPath, "--github-fixture", "demo", "serve", "--addr", address)
	var serverLog bytes.Buffer
	server.Stderr = &serverLog
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	for {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address, nil)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		response, requestErr := client.Do(request)
		if requestErr == nil {
			_ = response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			cancel()
			_ = server.Wait()
			t.Fatalf("server did not start: %v\n%s", requestErr, serverLog.String())
		}
		time.Sleep(25 * time.Millisecond)
	}
	created, _, exit := runCLI(t, binary, dbPath, "feature", "create", "--slug", "live-write", "--title", "Live write")
	if exit != 0 || !created.OK {
		cancel()
		_ = server.Wait()
		t.Fatalf("CLI write failed: %+v", created)
	}
	rpcClient := prxv1connect.NewPRXServiceClient(http.DefaultClient, "http://"+address)
	snapshot, err := rpcClient.GetSnapshot(context.Background(), connect.NewRequest(&prxv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Msg.Snapshot.Features) != 1 || snapshot.Msg.Snapshot.Features[0].Slug != "live-write" {
		t.Fatalf("server did not observe CLI write: %+v", snapshot.Msg.Snapshot.Features)
	}
	cancel()
	_ = server.Wait()
}
