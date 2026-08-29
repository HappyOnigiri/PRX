package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	return runCLIWithFixture(t, binary, dbPath, "", args...)
}

func runCLIWithFixture(
	t *testing.T,
	binary, dbPath, fixture string,
	args ...string,
) (resultEnvelope, string, int) {
	t.Helper()
	base := []string{"--db", dbPath, "--json"}
	if fixture != "" {
		base = append(base, "--github-fixture", fixture)
	}
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
	if err := json.Unmarshal(feature.Data, &featureData); err != nil {
		t.Fatal(err)
	}
	if featureData.ID != "F-1" {
		t.Fatalf("feature ID=%q, want F-1", featureData.ID)
	}
	byID, _, exit := runCLI(t, binary, dbPath, "feature", "get", featureData.ID)
	if exit != 0 || !byID.OK {
		t.Fatalf("feature get by ID: %+v exit=%d", byID, exit)
	}
	nodeFeature, _, exit := runCLI(t, binary, dbPath, "node", "get", featureData.ID)
	if exit != 0 || !nodeFeature.OK {
		t.Fatalf("node get feature: %+v exit=%d", nodeFeature, exit)
	}
	var nodeFeatureData struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(nodeFeature.Data, &nodeFeatureData); err != nil {
		t.Fatal(err)
	}
	if nodeFeatureData.ID != featureData.ID || nodeFeatureData.Slug != "release" {
		t.Fatalf("node feature=%+v", nodeFeatureData)
	}
	a, _, _ := runCLI(t, binary, dbPath, "task", "create", "--feature", featureData.ID, "--title", "A")
	b, _, _ := runCLI(t, binary, dbPath, "task", "create", "--feature", featureData.ID, "--title", "B")
	var at, bt struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(a.Data, &at); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b.Data, &bt); err != nil {
		t.Fatal(err)
	}
	if at.ID != "T-1" || bt.ID != "T-2" {
		t.Fatalf("task IDs=%q,%q, want T-1,T-2", at.ID, bt.ID)
	}
	byTaskID, _, exit := runCLI(t, binary, dbPath, "task", "get", at.ID)
	if exit != 0 || !byTaskID.OK {
		t.Fatalf("task get by ID: %+v exit=%d", byTaskID, exit)
	}
	nodeTask, _, exit := runCLI(t, binary, dbPath, "node", "get", at.ID)
	if exit != 0 || !nodeTask.OK {
		t.Fatalf("node get task: %+v exit=%d", nodeTask, exit)
	}
	var nodeTaskData struct {
		ID        string `json:"id"`
		FeatureID string `json:"feature_id"`
		Kind      string `json:"kind"`
	}
	if err := json.Unmarshal(nodeTask.Data, &nodeTaskData); err != nil {
		t.Fatal(err)
	}
	if nodeTaskData.ID != at.ID || nodeTaskData.FeatureID != featureData.ID || nodeTaskData.Kind != "pr" {
		t.Fatalf("node task=%+v", nodeTaskData)
	}
	if value, _, exit := runCLI(t, binary, dbPath, "dependency", "add", at.ID, bt.ID); exit != 0 || !value.OK {
		t.Fatalf("add dependency: %+v", value)
	}
	value, _, exit := runCLI(t, binary, dbPath, "dependency", "add", bt.ID, at.ID)
	if exit == 0 || value.Error == nil || value.Error.Code != "cycle" {
		t.Fatalf("cycle result=%+v exit=%d", value, exit)
	}
	for _, removal := range [][]string{
		{"dependency", "remove", at.ID, "missing-task"},
		{"pr", "detach", at.ID},
		{"document", "delete", "missing-document"},
	} {
		value, _, exit := runCLI(t, binary, dbPath, removal...)
		if exit == 0 || value.Error == nil || value.Error.Code != "not_found" {
			t.Fatalf("%v result=%+v exit=%d", removal, value, exit)
		}
	}
	valid, _, exit := runCLI(t, binary, dbPath, "validate")
	if exit != 0 || !valid.OK {
		t.Fatalf("validate result=%+v", valid)
	}
}

func TestBlackBoxTargetedSyncByID(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "targeted-sync.db")
	feature, _, exit := runCLI(t, binary, dbPath, "feature", "create", "--slug", "targeted", "--title", "Targeted")
	if exit != 0 || !feature.OK {
		t.Fatalf("feature create: %+v exit=%d", feature, exit)
	}
	var featureData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(feature.Data, &featureData); err != nil {
		t.Fatal(err)
	}
	first, _, exit := runCLI(t, binary, dbPath, "task", "create", "--feature", featureData.ID, "--title", "First")
	if exit != 0 || !first.OK {
		t.Fatalf("first task create: %+v exit=%d", first, exit)
	}
	second, _, exit := runCLI(t, binary, dbPath, "task", "create", "--feature", featureData.ID, "--title", "Second")
	if exit != 0 || !second.OK {
		t.Fatalf("second task create: %+v exit=%d", second, exit)
	}
	var firstTask, secondTask struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(first.Data, &firstTask); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second.Data, &secondTask); err != nil {
		t.Fatal(err)
	}
	for index, taskID := range []string{firstTask.ID, secondTask.ID} {
		attached, _, attachExit := runCLI(
			t,
			binary,
			dbPath,
			"pr",
			"attach",
			"--task",
			taskID,
			"--url",
			fmt.Sprintf("https://github.com/acme/api/pull/%d", index+1),
		)
		if attachExit != 0 || !attached.OK {
			t.Fatalf("attach %s: %+v exit=%d", taskID, attached, attachExit)
		}
	}
	synced, stderr, exit := runCLIWithFixture(t, binary, dbPath, "demo", "sync", "--task", firstTask.ID)
	if exit != 0 || stderr != "" || !synced.OK {
		t.Fatalf("sync by task ID: %+v stderr=%q exit=%d", synced, stderr, exit)
	}
	var counts struct {
		Succeeded int `json:"succeeded"`
		Failed    int `json:"failed"`
	}
	if err := json.Unmarshal(synced.Data, &counts); err != nil {
		t.Fatal(err)
	}
	if counts.Succeeded != 1 || counts.Failed != 0 {
		t.Fatalf("sync counts=%+v, want one success", counts)
	}
	snapshot, _, exit := runCLI(t, binary, dbPath, "snapshot")
	if exit != 0 || !snapshot.OK {
		t.Fatalf("snapshot: %+v exit=%d", snapshot, exit)
	}
	var snapshotData struct {
		PullRequests []struct {
			TaskID string `json:"task_id"`
			Stale  bool   `json:"stale"`
		} `json:"pull_requests"`
	}
	if err := json.Unmarshal(snapshot.Data, &snapshotData); err != nil {
		t.Fatal(err)
	}
	for _, pullRequest := range snapshotData.PullRequests {
		switch pullRequest.TaskID {
		case firstTask.ID:
			if pullRequest.Stale {
				t.Fatalf("target pull request is stale: %+v", pullRequest)
			}
		case secondTask.ID:
			if !pullRequest.Stale {
				t.Fatalf("non-target pull request was refreshed: %+v", pullRequest)
			}
		}
	}
	featureSync, _, exit := runCLIWithFixture(t, binary, dbPath, "demo", "sync", "--feature", featureData.ID)
	if exit != 0 || !featureSync.OK {
		t.Fatalf("sync by feature ID: %+v exit=%d", featureSync, exit)
	}
	missing, _, exit := runCLIWithFixture(t, binary, dbPath, "demo", "sync", "--task", "missing-task")
	if exit == 0 || missing.OK || missing.Error == nil || missing.Error.Code != "not_found" {
		t.Fatalf("missing task sync: %+v exit=%d", missing, exit)
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
	for _, guarded := range []struct {
		name   string
		host   string
		origin string
	}{
		{name: "rebound host", host: "attacker.example"},
		{name: "cross origin", host: address, origin: "https://attacker.example"},
	} {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address, nil)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		request.Host = guarded.host
		if guarded.origin != "" {
			request.Header.Set("Origin", guarded.origin)
		}
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr != nil {
			t.Fatalf("%s: %v", guarded.name, requestErr)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			cancel()
			_ = server.Wait()
			t.Fatalf("%s: status = %d, want %d", guarded.name, response.StatusCode, http.StatusForbidden)
		}
	}
	rpcClient := prxv1connect.NewPRXServiceClient(http.DefaultClient, "http://"+address)
	snapshot, err := rpcClient.GetSnapshot(context.Background(), connect.NewRequest(&prxv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	features := snapshot.Msg.GetSnapshot().GetFeatures()
	if len(features) != 1 || features[0].GetSlug() != "live-write" {
		t.Fatalf("server did not observe CLI write: %+v", features)
	}
	cancel()
	_ = server.Wait()
}

func TestBlackBoxJSONFlagFormsAgree(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "flagforms.db")
	for _, flag := range []string{"--json", "--json=true"} {
		t.Run(flag, func(t *testing.T) {
			command := exec.CommandContext(
				context.Background(),
				binary,
				"--db",
				dbPath,
				flag,
				"task",
				"get",
				"missing-task",
			)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			if err := command.Run(); err == nil {
				t.Fatal("expected a non-zero exit for a missing task")
			}
			var envelope resultEnvelope
			if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
				t.Fatalf("stdout is not pure JSON: %v\n%s", err, stdout.String())
			}
			if envelope.OK || envelope.Error == nil || envelope.Error.Code != "not_found" {
				t.Fatalf("envelope=%+v", envelope)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr must stay empty in JSON mode: %q", stderr.String())
			}
		})
	}
}
