package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
)

type resultEnvelope struct {
	OK        bool
	Data      json.RawMessage
	Error     string
	ErrorCode string
	Hint      string
	keys      map[string]json.RawMessage
}

type commandOutput struct {
	stdout string
	stderr string
	exit   int
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

func runCLIWithInput(
	t *testing.T,
	binary, dbPath, input string,
	args ...string,
) (resultEnvelope, string, int) {
	t.Helper()
	result := executeCLI(t, binary, input, append([]string{"--db", dbPath, "--json"}, args...)...)
	if result.exit != 0 {
		return decodeFailure(t, []byte(result.stderr), result.stderr), result.stderr, result.exit
	}
	return decodeResult(t, []byte(result.stdout), result.stdout), result.stderr, result.exit
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
	result := executeCLI(t, binary, "", append(base, args...)...)
	if result.exit != 0 {
		return decodeFailure(t, []byte(result.stderr), result.stderr), result.stderr, result.exit
	}
	return decodeResult(t, []byte(result.stdout), result.stdout), result.stderr, result.exit
}

func executeCLI(t *testing.T, binary, input string, args ...string) commandOutput {
	t.Helper()
	return executeCLIWithEnv(t, binary, input, nil, args...)
}

func executeCLIWithEnv(t *testing.T, binary, input string, env []string, args ...string) commandOutput {
	t.Helper()
	command := exec.CommandContext(context.Background(), binary, args...)
	if len(env) > 0 {
		command.Env = append(os.Environ(), env...)
	}
	if input != "" {
		command.Stdin = strings.NewReader(input)
	}
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
	return commandOutput{stdout: stdout.String(), stderr: stderr.String(), exit: exit}
}

func decodeFailure(t *testing.T, body []byte, output string) resultEnvelope {
	t.Helper()
	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Hint    string `json:"hint"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(body, &response); decodeErr != nil {
		t.Fatalf("stderr is not pure JSON: %v\n%s", decodeErr, output)
	}
	keys := decodeObject(t, body, output)
	return resultEnvelope{
		Error: response.Error.Message, ErrorCode: response.Error.Code, Hint: response.Error.Hint, keys: keys,
	}
}

func decodeResult(t *testing.T, body []byte, output string) resultEnvelope {
	t.Helper()
	keys := decodeObject(t, body, output)
	assertCompactJSON(t, output)
	assertNoEnvelopeKeys(t, keys)
	return resultEnvelope{OK: true, Data: json.RawMessage(bytes.TrimSpace(body)), keys: keys}
}

func decodeObject(t *testing.T, body []byte, output string) map[string]json.RawMessage {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("stdout is not a JSON object: %v\n%s", err, output)
	}
	return value
}

func assertDirectObjectKeys(t *testing.T, value map[string]json.RawMessage, expected ...string) {
	t.Helper()
	assertObjectKeys(t, value, expected...)
	assertNoEnvelopeKeys(t, value)
}

func assertDirectObject(t *testing.T, value map[string]json.RawMessage, required ...string) {
	t.Helper()
	assertNoEnvelopeKeys(t, value)
	for _, key := range required {
		if _, ok := value[key]; !ok {
			t.Fatalf("response keys=%v, want key %q", mapKeys(value), key)
		}
	}
}

func assertNoEnvelopeKeys(t *testing.T, value map[string]json.RawMessage) {
	t.Helper()
	for _, key := range []string{"schema_version", "ok", "data"} {
		if _, ok := value[key]; ok {
			t.Fatalf("normal response contains envelope key %q: %v", key, mapKeys(value))
		}
	}
}

func assertObjectKeys(t *testing.T, value map[string]json.RawMessage, expected ...string) {
	t.Helper()
	want := make(map[string]struct{}, len(expected))
	for _, key := range expected {
		want[key] = struct{}{}
	}
	if len(value) != len(want) {
		t.Fatalf("response keys=%v, want %v", mapKeys(value), expected)
	}
	for key := range want {
		if _, ok := value[key]; !ok {
			t.Fatalf("response keys=%v, want %v", mapKeys(value), expected)
		}
	}
}

func assertCompactJSON(t *testing.T, output string) {
	t.Helper()
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("JSON output has no trailing newline: %q", output)
	}
	if strings.Contains(strings.TrimSuffix(output, "\n"), "\n") {
		t.Fatalf("JSON output is not compact: %q", output)
	}
}

func assertNormalSuccess(t *testing.T, result commandOutput, required ...string) map[string]json.RawMessage {
	t.Helper()
	if result.exit != 0 || result.stderr != "" {
		t.Fatalf("normal command failed: stdout=%q stderr=%q exit=%d", result.stdout, result.stderr, result.exit)
	}
	assertCompactJSON(t, result.stdout)
	value := decodeObject(t, []byte(result.stdout), result.stdout)
	assertDirectObject(t, value, required...)
	return value
}

func assertEnvelopeKeys(t *testing.T, envelope resultEnvelope, expected ...string) {
	t.Helper()
	want := make(map[string]struct{}, len(expected))
	for _, key := range expected {
		want[key] = struct{}{}
	}
	if len(envelope.keys) != len(want) {
		t.Fatalf("response keys=%v, want %v", mapKeys(envelope.keys), expected)
	}
	for key := range want {
		if _, ok := envelope.keys[key]; !ok {
			t.Fatalf("response keys=%v, want %v", mapKeys(envelope.keys), expected)
		}
	}
}

func mapKeys(value map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	return keys
}

func decodeDataObject(t *testing.T, envelope resultEnvelope) map[string]json.RawMessage {
	t.Helper()
	if envelope.Data == nil {
		t.Fatalf("response has no data: %+v", envelope)
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Data, &value); err != nil {
		t.Fatalf("data is not an object: %v\n%s", err, envelope.Data)
	}
	return value
}

func TestBlackBoxCRUDAndCycle(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "blackbox.db")
	feature, stderr, exit := runCLI(t, binary, dbPath, "feature", "create", "release", "Release")
	if exit != 0 || stderr != "" || !feature.OK {
		t.Fatalf("feature result=%+v stderr=%q exit=%d", feature, stderr, exit)
	}
	assertNoEnvelopeKeys(t, feature.keys)
	var featureData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(feature.Data, &featureData); err != nil {
		t.Fatal(err)
	}
	if featureData.ID != "F-1" {
		t.Fatalf("feature ID=%q, want F-1", featureData.ID)
	}
	byID, _, exit := runCLI(t, binary, dbPath, "feature", featureData.ID)
	if exit != 0 || !byID.OK {
		t.Fatalf("feature get by ID: %+v exit=%d", byID, exit)
	}
	nodeFeature, _, exit := runCLI(t, binary, dbPath, "show", featureData.ID)
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
	a, _, _ := runCLI(t, binary, dbPath, "task", "create", featureData.ID, "A")
	b, _, _ := runCLI(t, binary, dbPath, "task", "create", featureData.ID, "B")
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
	byTaskID, _, exit := runCLI(t, binary, dbPath, "task", at.ID)
	if exit != 0 || !byTaskID.OK {
		t.Fatalf("task get by ID: %+v exit=%d", byTaskID, exit)
	}
	nodeTask, _, exit := runCLI(t, binary, dbPath, "show", at.ID)
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
	if exit == 0 || value.Error == "" || !strings.Contains(value.Error, "cycle") {
		t.Fatalf("cycle result=%+v exit=%d", value, exit)
	}
	assertEnvelopeKeys(t, value, "error")
	for _, removal := range [][]string{
		{"dependency", "remove", at.ID, "missing-task"},
		{"pr", "detach", at.ID},
		{"document", "delete", "missing-document"},
	} {
		value, _, exit := runCLI(t, binary, dbPath, removal...)
		if exit == 0 || value.Error == "" || !strings.Contains(value.Error, "was not found") {
			t.Fatalf("%v result=%+v exit=%d", removal, value, exit)
		}
		assertEnvelopeKeys(t, value, "error")
	}
	valid, _, exit := runCLI(t, binary, dbPath, "validate")
	if exit != 0 || !valid.OK {
		t.Fatalf("validate result=%+v", valid)
	}
}

func TestBlackBoxTargetedSyncByID(t *testing.T) {
	binary := buildCLI(t)
	t.Setenv("PRX_CONFIG", filepath.Join(t.TempDir(), "missing-config.yaml"))
	dbPath := filepath.Join(t.TempDir(), "targeted-sync.db")
	feature, _, exit := runCLI(t, binary, dbPath, "feature", "create", "targeted", "Targeted")
	if exit != 0 || !feature.OK {
		t.Fatalf("feature create: %+v exit=%d", feature, exit)
	}
	var featureData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(feature.Data, &featureData); err != nil {
		t.Fatal(err)
	}
	first, _, exit := runCLI(t, binary, dbPath, "task", "create", featureData.ID, "First")
	if exit != 0 || !first.OK {
		t.Fatalf("first task create: %+v exit=%d", first, exit)
	}
	second, _, exit := runCLI(t, binary, dbPath, "task", "create", featureData.ID, "Second")
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
			taskID,
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
	if exit == 0 || missing.OK || missing.Error == "" || !strings.Contains(missing.Error, "was not found") {
		t.Fatalf("missing task sync: %+v exit=%d", missing, exit)
	}
}

func TestBlackBoxImplementationPlanCommands(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "plans.db")
	feature, _, exit := runCLI(t, binary, dbPath, "feature", "create", "plans", "Plans")
	if exit != 0 || !feature.OK {
		t.Fatalf("feature result=%+v exit=%d", feature, exit)
	}
	var featureData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(feature.Data, &featureData); err != nil {
		t.Fatal(err)
	}
	task, _, exit := runCLI(
		t,
		binary,
		dbPath,
		"task",
		"create",
		featureData.ID,
		"Plan task",
		"--kind",
		"manual",
	)
	if exit != 0 || !task.OK {
		t.Fatalf("task result=%+v exit=%d", task, exit)
	}
	var taskData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(task.Data, &taskData); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(t.TempDir(), "plan.md")
	const content = "# Plan\n\nImplement it.\n"
	if err := os.WriteFile(planPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	set, _, exit := runCLI(t, binary, dbPath, "plan", "set", taskData.ID, planPath)
	if exit != 0 || !set.OK {
		t.Fatalf("set result=%+v exit=%d", set, exit)
	}
	var planData struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(set.Data, &planData); err != nil {
		t.Fatal(err)
	}
	if planData.Content != content {
		t.Fatalf("set content=%q, want %q", planData.Content, content)
	}
	got, _, exit := runCLI(t, binary, dbPath, "plan", taskData.ID)
	if exit != 0 || !got.OK {
		t.Fatalf("get result=%+v exit=%d", got, exit)
	}
	if err := json.Unmarshal(got.Data, &planData); err != nil {
		t.Fatal(err)
	}
	if planData.Content != content {
		t.Fatalf("get content=%q, want %q", planData.Content, content)
	}
	taskSnapshot, _, exit := runCLI(t, binary, dbPath, "task", taskData.ID)
	if exit != 0 || !taskSnapshot.OK {
		t.Fatalf("task get result=%+v exit=%d", taskSnapshot, exit)
	}
	var taskState struct {
		HasPlan bool   `json:"has_implementation_plan"`
		Display string `json:"display_state"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(taskSnapshot.Data, &taskState); err != nil {
		t.Fatal(err)
	}
	if !taskState.HasPlan || taskState.Display != "designed" || taskState.Status != "auto" {
		t.Fatalf("task state=%+v", taskState)
	}
	deleted, _, exit := runCLI(t, binary, dbPath, "plan", "delete", taskData.ID)
	if exit != 0 || !deleted.OK {
		t.Fatalf("delete result=%+v exit=%d", deleted, exit)
	}
	var deletedData struct {
		ID string `json:"deleted"`
	}
	if err := json.Unmarshal(deleted.Data, &deletedData); err != nil {
		t.Fatal(err)
	}
	if deletedData.ID != taskData.ID {
		t.Fatalf("deleted=%q, want %q", deletedData.ID, taskData.ID)
	}
	stdinSet, _, exit := runCLIWithInput(
		t,
		binary,
		dbPath,
		"# Standard input plan\n",
		"plan",
		"set",
		taskData.ID,
		"-",
	)
	if exit != 0 || !stdinSet.OK || !bytes.Contains(stdinSet.Data, []byte("Standard input plan")) {
		t.Fatalf("stdin set result=%+v exit=%d", stdinSet, exit)
	}
	missingFile, _, exit := runCLI(
		t,
		binary,
		dbPath,
		"plan",
		"set",
		taskData.ID,
		filepath.Join(t.TempDir(), "missing.md"),
	)
	if exit == 0 || missingFile.ErrorCode != "invalid_implementation_plan" ||
		!strings.Contains(missingFile.Error, "could not read implementation plan") {
		t.Fatalf("missing file result=%+v exit=%d", missingFile, exit)
	}
	emptyStdin, _, exit := runCLIWithInput(t, binary, dbPath, "", "plan", "set", taskData.ID, "-")
	if exit == 0 || emptyStdin.ErrorCode != "invalid_implementation_plan" {
		t.Fatalf("empty stdin result=%+v exit=%d", emptyStdin, exit)
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
	created, _, exit := runCLI(t, binary, dbPath, "feature", "create", "live-write", "Live write")
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
				"missing-task",
			)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			if err := command.Run(); err == nil {
				t.Fatal("expected a non-zero exit for a missing task")
			}
			envelope := decodeFailure(t, stderr.Bytes(), stderr.String())
			if envelope.OK || envelope.Error == "" || !strings.Contains(envelope.Error, "was not found") {
				t.Fatalf("envelope=%+v", envelope)
			}
			assertEnvelopeKeys(t, envelope, "error")
			if stdout.Len() != 0 {
				t.Fatalf("stdout must stay empty in JSON mode: %q", stdout.String())
			}
		})
	}
}

func TestBlackBoxReadAliasesAndFeatureSlugEscape(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "aliases.db")
	for _, slug := range []string{"checkout", "create"} {
		created, _, exit := runCLI(t, binary, dbPath, "feature", "create", slug, slug)
		if exit != 0 || !created.OK {
			t.Fatalf("create feature %q: %+v exit=%d", slug, created, exit)
		}
	}
	feature, _, exit := runCLI(t, binary, dbPath, "f", "checkout")
	if exit != 0 || !feature.OK || !bytes.Contains(feature.Data, []byte(`"slug":"checkout"`)) {
		t.Fatalf("feature alias: %+v exit=%d", feature, exit)
	}
	escaped, _, exit := runCLI(t, binary, dbPath, "show", "create")
	if exit != 0 || !escaped.OK || !bytes.Contains(escaped.Data, []byte(`"slug":"create"`)) {
		t.Fatalf("feature slug escape: %+v exit=%d", escaped, exit)
	}

	task, _, exit := runCLI(t, binary, dbPath, "task", "create", "checkout", "Build")
	if exit != 0 || !task.OK {
		t.Fatalf("create task: %+v exit=%d", task, exit)
	}
	var taskData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(task.Data, &taskData); err != nil {
		t.Fatal(err)
	}
	byAlias, _, exit := runCLI(t, binary, dbPath, "t", taskData.ID)
	if exit != 0 || !byAlias.OK {
		t.Fatalf("task alias: %+v exit=%d", byAlias, exit)
	}
	filtered, _, exit := runCLI(t, binary, dbPath, "task", "--feature", "checkout")
	if exit != 0 || !filtered.OK || !bytes.Contains(filtered.Data, []byte(taskData.ID)) {
		t.Fatalf("task filter: %+v exit=%d", filtered, exit)
	}
	invalid, _, exit := runCLI(t, binary, dbPath, "task", taskData.ID, "--feature", "checkout")
	if exit == 0 || invalid.ErrorCode != "usage_error" || !strings.Contains(invalid.Error, "cannot be used") {
		t.Fatalf("task ID with filter: %+v exit=%d", invalid, exit)
	}

	for alias, key := range map[string]string{"dep": "dependencies", "doc": "documents"} {
		value, _, aliasExit := runCLI(t, binary, dbPath, alias)
		if aliasExit != 0 || !value.OK {
			t.Fatalf("%s alias: %+v exit=%d", alias, value, aliasExit)
		}
		if _, ok := decodeDataObject(t, value)[key]; !ok {
			t.Fatalf("%s alias omitted %q", alias, key)
		}
	}
}

func TestBlackBoxHelpAndErrorShareCanonicalHint(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "help.db")
	help := executeCLI(t, binary, "", "--db", dbPath, "feature", "--help", "--json")
	if help.exit != 0 || help.stderr != "" {
		t.Fatalf("JSON help failed: %+v", help)
	}
	assertCompactJSON(t, help.stdout)
	var helpValue struct {
		Hint string `json:"hint"`
	}
	if err := json.Unmarshal([]byte(help.stdout), &helpValue); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Usage:\n  prx feature", "Examples:\nprx feature", "Global Flags:"} {
		if !strings.Contains(helpValue.Hint, text) {
			t.Fatalf("help omitted %q: %q", text, helpValue.Hint)
		}
	}

	failure := executeCLI(t, binary, "", "--db", dbPath, "feature", "missing", "--json")
	if failure.exit == 0 || failure.stdout != "" {
		t.Fatalf("JSON failure result=%+v", failure)
	}
	decoded := decodeFailure(t, []byte(failure.stderr), failure.stderr)
	if decoded.Hint != helpValue.Hint {
		t.Fatalf("error hint differs from normal help\nhelp=%q\nhint=%q", helpValue.Hint, decoded.Hint)
	}

	text := executeCLI(t, binary, "", "--db", dbPath, "feature", "missing")
	if text.exit == 0 || text.stdout != "" || text.stderr != "Error: "+decoded.Error+"\n\n"+helpValue.Hint {
		t.Fatalf("text error did not reuse help: %+v", text)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("domain error should have opened storage: %v", err)
	}
}

func TestBlackBoxJSONHelpFormsDoNotOpenStorage(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "missing.db")
	for _, args := range [][]string{
		{"--json"},
		{"--json", "help", "task"},
		{"help", "config", "host", "--json"},
		{"task", "--json", "--help"},
	} {
		result := executeCLI(t, binary, "", append([]string{"--db", dbPath}, args...)...)
		if result.exit != 0 || result.stderr != "" {
			t.Fatalf("%v: %+v", args, result)
		}
		value := decodeObject(t, []byte(result.stdout), result.stdout)
		assertObjectKeys(t, value, "hint")
		assertCompactJSON(t, result.stdout)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("help opened storage: %v", err)
	}
}

func TestBlackBoxResolvedCommandErrorsIncludeCompleteHelp(t *testing.T) {
	binary := buildCLI(t)
	for _, test := range []struct {
		name      string
		args      []string
		usage     string
		example   string
		wantError string
	}{
		{
			name: "unknown command with trailing JSON flag", args: []string{"unknown", "--json"},
			usage: "Usage:\n  prx [command]", wantError: "unknown command",
		},
		{
			name: "deep unknown command", args: []string{"config", "host", "unknown", "--json"},
			usage: "Usage:\n  prx config host", example: "Examples:\nprx config host", wantError: "unknown command",
		},
		{
			name:      "missing positional arguments",
			args:      []string{"feature", "create", "--json"},
			usage:     "Usage:\n  prx feature create SLUG TITLE",
			example:   "Examples:\nprx feature create",
			wantError: "accepts 2 arg(s)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := executeCLI(t, binary, "", test.args...)
			if result.exit == 0 || result.stdout != "" {
				t.Fatalf("result=%+v", result)
			}
			failure := decodeFailure(t, []byte(result.stderr), result.stderr)
			if !strings.Contains(failure.Error, test.wantError) || !strings.Contains(failure.Hint, test.usage) ||
				(test.example != "" && !strings.Contains(failure.Hint, test.example)) ||
				!strings.Contains(failure.Hint, "Flags:") {
				t.Fatalf("failure=%+v", failure)
			}
			assertCompactJSON(t, result.stderr)
		})
	}
}

func TestBlackBoxShowReportsMissingTargetsWithCurrentVocabulary(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "show.db")
	for _, identifier := range []string{"F-99", "checkout", "T-99"} {
		result := executeCLI(t, binary, "", "--db", dbPath, "--json", "show", identifier)
		if result.exit == 0 || result.stdout != "" {
			t.Fatalf("%s: result=%+v", identifier, result)
		}
		failure := decodeFailure(t, []byte(result.stderr), result.stderr)
		if strings.Contains(failure.Error, "node ") {
			t.Fatalf("%s: error names the removed node command: %q", identifier, failure.Error)
		}
		if !strings.Contains(failure.Error, "was not found") {
			t.Fatalf("%s: error=%q", identifier, failure.Error)
		}
	}
}

func TestBlackBoxRemovedHumanFlagIsRejectedWithRootHelp(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "conflict.db")
	help := executeCLI(t, binary, "", "--db", dbPath, "--json", "--help")
	if help.exit != 0 || help.stderr != "" {
		t.Fatalf("root help failed: %+v", help)
	}
	var helpValue struct {
		Hint string `json:"hint"`
	}
	if err := json.Unmarshal([]byte(help.stdout), &helpValue); err != nil {
		t.Fatal(err)
	}

	removed := executeCLI(t, binary, "", "--db", dbPath, "--json", "--human")
	if removed.exit == 0 || removed.stdout != "" {
		t.Fatalf("removed flag result=%+v", removed)
	}
	failure := decodeFailure(t, []byte(removed.stderr), removed.stderr)
	if failure.ErrorCode != "usage_error" || !strings.Contains(failure.Error, "unknown flag: --human") {
		t.Fatalf("removed flag error=%+v", failure)
	}
	if failure.Hint != helpValue.Hint {
		t.Fatalf("removed flag hint differs from root help\nhelp=%q\nhint=%q", helpValue.Hint, failure.Hint)
	}
}

func TestBlackBoxUnknownHelpTopicUsesTheErrorPath(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "help.db")
	result := executeCLI(t, binary, "", "--db", dbPath, "--json", "help", "nosuchtopic")
	if result.exit == 0 || result.stdout != "" {
		t.Fatalf("result=%+v", result)
	}
	failure := decodeFailure(t, []byte(result.stderr), result.stderr)
	if !strings.Contains(failure.Error, "unknown command") || !strings.Contains(failure.Hint, "Usage:\n  prx help") {
		t.Fatalf("failure=%+v", failure)
	}
	assertCompactJSON(t, result.stderr)
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("unknown help topic opened storage: %v", err)
	}
}

func TestBlackBoxErrorHintDoesNotVaryWithAmbientEnvironment(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "hint.db")
	hints := make([]string, 0, 2)
	for _, ambient := range []string{"/first/ambient", "/second/ambient"} {
		result := executeCLIWithEnv(t, binary, "", []string{
			"PRX_DB=" + filepath.Join(ambient, "prx.db"),
			"PRX_CONFIG=" + filepath.Join(ambient, "prx.yaml"),
		}, "--db", dbPath, "--json", "feature", "missing")
		if result.exit == 0 || result.stdout != "" {
			t.Fatalf("result=%+v", result)
		}
		failure := decodeFailure(t, []byte(result.stderr), result.stderr)
		if strings.Contains(failure.Hint, ambient) {
			t.Fatalf("hint leaked the ambient path %q: %q", ambient, failure.Hint)
		}
		hints = append(hints, failure.Hint)
	}
	if hints[0] != hints[1] {
		t.Fatalf("hint varied with the environment\nfirst=%q\nsecond=%q", hints[0], hints[1])
	}
}

func TestBlackBoxEnvironmentSuppliesStoragePathWithoutFlag(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "env.db")
	result := executeCLIWithEnv(t, binary, "", []string{"PRX_DB=" + dbPath},
		"--json", "feature", "create", "from-environment", "From environment")
	if result.exit != 0 || result.stderr != "" {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("PRX_DB was not used as the storage path: %v", err)
	}
}

func TestBlackBoxCollectionResponsesUseNamedKeys(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "collections.db")
	cases := []struct {
		name string
		args []string
		key  string
	}{
		{name: "features", args: []string{"feature"}, key: "features"},
		{name: "tasks", args: []string{"task"}, key: "tasks"},
		{name: "dependencies", args: []string{"dependency"}, key: "dependencies"},
		{name: "pull requests", args: []string{"pr"}, key: "pull_requests"},
		{name: "documents", args: []string{"document"}, key: "documents"},
		{name: "ready tasks", args: []string{"ready"}, key: "ready_tasks"},
		{name: "review waiting tasks", args: []string{"reviews"}, key: "review_waiting_tasks"},
		{name: "conflict tasks", args: []string{"conflicts"}, key: "conflict_tasks"},
		{name: "stale tasks", args: []string{"stale"}, key: "stale_tasks"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			value, stderr, exit := runCLI(t, binary, dbPath, test.args...)
			if exit != 0 || stderr != "" || !value.OK {
				t.Fatalf("result=%+v stderr=%q exit=%d", value, stderr, exit)
			}
			data := decodeDataObject(t, value)
			if len(data) != 1 {
				t.Fatalf("data keys=%v, want only %q", mapKeys(data), test.key)
			}
			if _, ok := data[test.key]; !ok {
				t.Fatalf("data keys=%v, want %q", mapKeys(data), test.key)
			}
			assertJSONArray(t, data[test.key], test.key)
			assertNoEnvelopeKeys(t, value.keys)
		})
	}

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	for _, test := range []struct {
		name string
		args []string
		key  string
	}{
		{name: "hosts", args: []string{"config", "host"}, key: "hosts"},
		{name: "auth methods", args: []string{"config", "auth"}, key: "auth_methods"},
	} {
		t.Run("config "+test.name, func(t *testing.T) {
			value, stderr, exit := runConfigCLI(
				t,
				binary,
				filepath.Join(t.TempDir(), "unused.db"),
				configPath,
				"",
				test.args...,
			)
			if exit != 0 || stderr != "" || !value.OK {
				t.Fatalf("result=%+v stderr=%q exit=%d", value, stderr, exit)
			}
			data := decodeDataObject(t, value)
			if len(data) != 1 {
				t.Fatalf("data keys=%v, want only %q", mapKeys(data), test.key)
			}
			if _, ok := data[test.key]; !ok {
				t.Fatalf("data keys=%v, want %q", mapKeys(data), test.key)
			}
			assertJSONArray(t, data[test.key], test.key)
			assertNoEnvelopeKeys(t, value.keys)
		})
	}
}

func assertJSONArray(t *testing.T, value json.RawMessage, name string) {
	t.Helper()
	var items []json.RawMessage
	if err := json.Unmarshal(value, &items); err != nil {
		t.Fatalf("%s is not an array: %v (%s)", name, err, value)
	}
	if items == nil {
		t.Fatalf("%s is null, want []", name)
	}
}

func TestBlackBoxSuccessOutputModes(t *testing.T) {
	binary := buildCLI(t)
	cases := []struct {
		name     string
		args     []string
		textWant string
		jsonWant string
	}{
		{
			name:     "config auth list",
			args:     []string{"config", "auth"},
			textWant: "No authentication methods configured.\n",
			jsonWant: `{"auth_methods":[]}
`,
		},
		{
			name:     "feature list",
			args:     []string{"feature"},
			textWant: "No features found.\n",
			jsonWant: `{"features":[]}
`,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "output.db")
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			base := []string{"--db", dbPath, "--config", configPath}

			textOutput := executeCLI(t, binary, "", append(base, test.args...)...)
			if textOutput.exit != 0 || textOutput.stderr != "" || textOutput.stdout != test.textWant {
				t.Fatalf(
					"text output=%q stderr=%q exit=%d, want %q",
					textOutput.stdout,
					textOutput.stderr,
					textOutput.exit,
					test.textWant,
				)
			}

			jsonOutput := executeCLI(t, binary, "", append(append(base, "--json"), test.args...)...)
			if jsonOutput.exit != 0 || jsonOutput.stderr != "" || jsonOutput.stdout != test.jsonWant {
				t.Fatalf(
					"JSON output=%q stderr=%q exit=%d, want %q",
					jsonOutput.stdout,
					jsonOutput.stderr,
					jsonOutput.exit,
					test.jsonWant,
				)
			}
		})
	}
}

func TestBlackBoxDefaultTextOutputCoversResourcesAndSummaries(t *testing.T) {
	binary := buildCLI(t)
	root := t.TempDir()
	dbPath := filepath.Join(root, "text.db")
	configPath := filepath.Join(root, "config.yaml")
	base := []string{"--db", dbPath, "--config", configPath}
	run := func(args ...string) commandOutput {
		result := executeCLI(t, binary, "", append(base, args...)...)
		if result.exit != 0 || result.stderr != "" || result.stdout == "" {
			t.Fatalf("%v: stdout=%q stderr=%q exit=%d", args, result.stdout, result.stderr, result.exit)
		}
		if json.Valid([]byte(result.stdout)) {
			t.Fatalf("%v emitted JSON by default: %s", args, result.stdout)
		}
		return result
	}

	createdFeature := run("feature", "create", "checkout", "Checkout rollout")
	if !strings.Contains(createdFeature.stdout, "Created feature checkout (F-1).") {
		t.Fatalf("feature create output=%q", createdFeature.stdout)
	}
	featureList := run("feature")
	for _, value := range []string{"ID", "SLUG", "STATUS", "TASKS", "Checkout rollout"} {
		if !strings.Contains(featureList.stdout, value) {
			t.Fatalf("feature list omitted %q: %s", value, featureList.stdout)
		}
	}
	featureGet := run("feature", "checkout")
	for _, value := range []string{"ID:", "Slug:", "Description:", "Created:", "Updated:"} {
		if !strings.Contains(featureGet.stdout, value) {
			t.Fatalf("feature detail omitted %q: %s", value, featureGet.stdout)
		}
	}

	taskA := run("task", "create", "checkout", "Payment API")
	taskB := run("task", "create", "checkout", "Checkout UI")
	if !strings.Contains(taskA.stdout, "Created task T-1.") || !strings.Contains(taskB.stdout, "Created task T-2.") {
		t.Fatalf("task create outputs=%q %q", taskA.stdout, taskB.stdout)
	}
	taskList := run("task", "--feature", "checkout")
	for _, value := range []string{"STATUS", "READY", "KIND", "ASSIGNEE", "Payment API"} {
		if !strings.Contains(taskList.stdout, value) {
			t.Fatalf("task list omitted %q: %s", value, taskList.stdout)
		}
	}
	dependency := run("dependency", "add", "T-1", "T-2")
	if dependency.stdout != "Added dependency T-1 -> T-2.\n" {
		t.Fatalf("dependency output=%q", dependency.stdout)
	}
	ready := run("ready")
	for _, value := range []string{"ID", "STATUS", "READY", "T-1", "Payment API"} {
		if !strings.Contains(ready.stdout, value) {
			t.Fatalf("ready queue omitted %q: %s", value, ready.stdout)
		}
	}
	graph := run("graph", "checkout")
	for _, value := range []string{"Feature", "Tasks", "Dependencies", "BLOCKER", "BLOCKED"} {
		if !strings.Contains(graph.stdout, value) {
			t.Fatalf("graph omitted %q: %s", value, graph.stdout)
		}
	}

	planPath := filepath.Join(root, "plan.md")
	if err := os.WriteFile(planPath, []byte("# Implementation plan\n\n1. Build it.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("plan", "set", "T-1", planPath)
	plan := run("plan", "T-1")
	for _, value := range []string{"Task: T-1", "Updated:", "# Implementation plan", "1. Build it."} {
		if !strings.Contains(plan.stdout, value) {
			t.Fatalf("plan omitted %q: %s", value, plan.stdout)
		}
	}
	snapshot := run("snapshot")
	snapshotSections := []string{
		"Snapshot:", "Queues:", "Features", "Tasks", "Dependencies", "Pull requests", "Documents",
	}
	for _, value := range snapshotSections {
		if !strings.Contains(snapshot.stdout, value) {
			t.Fatalf("snapshot omitted %q: %s", value, snapshot.stdout)
		}
	}
	configShow := run("config")
	for _, value := range []string{"Config version:", "Hosts", "Authentication methods"} {
		if !strings.Contains(configShow.stdout, value) {
			t.Fatalf("config show omitted %q: %s", value, configShow.stdout)
		}
	}
	syncStatus := run("sync", "status")
	for _, value := range []string{
		"Interval:", "Last attempt:", "Last updated:", "Succeeded:", "Failed:", "Error:",
	} {
		if !strings.Contains(syncStatus.stdout, value) {
			t.Fatalf("sync status omitted %q: %s", value, syncStatus.stdout)
		}
	}
	syncRead := run("config", "sync")
	if syncRead.stdout != "Automatic sync interval: 3600 seconds.\n" {
		t.Fatalf("config sync output=%q", syncRead.stdout)
	}
	syncUpdate := run("config", "sync", "update", "900")
	if syncUpdate.stdout != "Updated automatic sync interval to 900 seconds.\n" {
		t.Fatalf("config sync update output=%q", syncUpdate.stdout)
	}
}

func TestBlackBoxOutputFlagsAndErrorModes(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "flags.db")

	text := executeCLI(t, binary, "", "--json=false", "schema-version")
	if text.exit != 0 || text.stderr != "" || text.stdout != "Schema version: 2\n" {
		t.Fatalf("false JSON flag did not select text output: %+v", text)
	}

	textError := executeCLI(t, binary, "", "--db", dbPath, "task", "missing")
	if textError.exit == 0 || textError.stdout != "" || !strings.HasPrefix(textError.stderr, "Error: ") {
		t.Fatalf("text error result=%+v", textError)
	}

	usage := executeCLI(t, binary, "", "--json", "feature", "create")
	if usage.exit == 0 || usage.stdout != "" {
		t.Fatalf("usage error result=%+v", usage)
	}
	usageFailure := decodeFailure(t, []byte(usage.stderr), usage.stderr)
	if usageFailure.ErrorCode != "usage_error" {
		t.Fatalf("usage error code=%q body=%s", usageFailure.ErrorCode, usage.stderr)
	}
}

func TestBlackBoxRequiredOperandsRejectOldFlagsAndWrongCounts(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "operands.db")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	removedFlags := []struct {
		flag string
		args []string
	}{
		{flag: "--slug", args: []string{"feature", "create", "checkout", "Checkout", "--slug", "legacy"}},
		{flag: "--title", args: []string{"feature", "create", "checkout", "Checkout", "--title", "Legacy"}},
		{flag: "--feature", args: []string{"task", "create", "checkout", "Task", "--feature", "legacy"}},
		{flag: "--title", args: []string{"task", "create", "checkout", "Task", "--title", "Legacy"}},
		{flag: "--task", args: []string{"pr", "attach", "T-1", "https://example.com/pull/1", "--task", "T-1"}},
		{flag: "--url", args: []string{"pr", "attach", "T-1", "https://x.test/1", "--url", "https://x.test/2"}},
		{flag: "--value", args: []string{"document", "add", "T-1", "https://example.com", "--value", "legacy"}},
		{flag: "--file", args: []string{"plan", "set", "T-1", "plan.md", "--file", "legacy.md"}},
		{flag: "--stdin", args: []string{"plan", "set", "T-1", "plan.md", "--stdin"}},
		{flag: "--interval-seconds", args: []string{"config", "sync", "update", "600", "--interval-seconds", "600"}},
		{flag: "--host", args: []string{"config", "host", "add", "ghe.example.com", "--host", "legacy.example.com"}},
		{flag: "--id", args: []string{"config", "auth", "add", "work", "github.com", "gh_cli", "--id", "legacy"}},
		{flag: "--type", args: []string{"config", "auth", "add", "work", "github.com", "gh_cli", "--type", "inline"}},
		{flag: "--host", args: []string{"config", "auth", "add", "work", "github.com", "gh_cli", "--host", "x.test"}},
	}
	for _, test := range removedFlags {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			args := []string{"--db", dbPath, "--config", configPath, "--json"}
			result := executeCLI(t, binary, "", append(args, test.args...)...)
			if result.exit == 0 || result.stdout != "" {
				t.Fatalf("result=%+v", result)
			}
			failure := decodeFailure(t, []byte(result.stderr), result.stderr)
			if failure.ErrorCode != "usage_error" || !strings.Contains(failure.Error, "unknown flag: "+test.flag) {
				t.Fatalf("failure=%+v", failure)
			}
		})
	}

	wrongCounts := [][]string{
		{"feature", "create", "checkout"},
		{"task", "create", "checkout", "Task", "extra"},
		{"pr", "attach", "T-1"},
		{"document", "add", "T-1", "value", "extra"},
		{"plan", "set", "T-1"},
		{"config", "sync", "update", "600", "extra"},
		{"config", "host", "add"},
		{"config", "auth", "add", "work", "github.com"},
	}
	for _, commandArgs := range wrongCounts {
		t.Run(strings.Join(commandArgs, " "), func(t *testing.T) {
			args := []string{"--db", dbPath, "--config", configPath, "--json"}
			result := executeCLI(t, binary, "", append(args, commandArgs...)...)
			if result.exit == 0 || result.stdout != "" {
				t.Fatalf("result=%+v", result)
			}
			failure := decodeFailure(t, []byte(result.stderr), result.stderr)
			if failure.ErrorCode != "usage_error" || !strings.Contains(failure.Error, "arg(s)") {
				t.Fatalf("failure=%+v", failure)
			}
		})
	}
}

func TestBlackBoxDoubleDashPreservesLiteralPositionalValue(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "literal.db")
	created := executeCLI(t, binary, "", "--db", dbPath, "feature", "create", "literal", "--", "--json")
	if created.exit != 0 || created.stderr != "" || !strings.Contains(created.stdout, "Created feature literal") ||
		json.Valid([]byte(created.stdout)) {
		t.Fatalf("create result=%+v", created)
	}
	shown, _, exit := runCLI(t, binary, dbPath, "feature", "literal")
	if exit != 0 || !shown.OK || !bytes.Contains(shown.Data, []byte(`"title":"--json"`)) {
		t.Fatalf("shown=%+v exit=%d", shown, exit)
	}
	task := executeCLI(t, binary, "", "--db", dbPath, "task", "create", "literal", "--", "--assignee")
	if task.exit != 0 || task.stderr != "" || !strings.Contains(task.stdout, "Created task T-1") {
		t.Fatalf("task result=%+v", task)
	}
	document := executeCLI(
		t,
		binary,
		"",
		"--db",
		dbPath,
		"document",
		"add",
		"T-1",
		"--kind",
		"markdown_path",
		"--",
		"--json",
	)
	if document.exit != 0 || document.stderr != "" || !strings.Contains(document.stdout, "Added document") {
		t.Fatalf("document result=%+v", document)
	}
	invalidURL := executeCLI(t, binary, "", "--db", dbPath, "pr", "attach", "T-1", "--", "--url")
	if invalidURL.exit == 0 || invalidURL.stdout != "" || !strings.Contains(invalidURL.stderr, "expected an https") ||
		strings.Contains(invalidURL.stderr, "unknown flag") {
		t.Fatalf("pull request result=%+v", invalidURL)
	}
}

func TestBlackBoxDocumentAddResolvesPublicParentForms(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "documents.db")
	feature, _, exit := runCLI(t, binary, dbPath, "feature", "create", "checkout", "Checkout")
	if exit != 0 || !feature.OK {
		t.Fatalf("feature=%+v exit=%d", feature, exit)
	}
	var featureData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(feature.Data, &featureData); err != nil {
		t.Fatal(err)
	}
	task, _, exit := runCLI(t, binary, dbPath, "task", "create", featureData.ID, "Task")
	if exit != 0 || !task.OK {
		t.Fatalf("task=%+v exit=%d", task, exit)
	}
	var taskData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(task.Data, &taskData); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name       string
		parent     string
		wantField  string
		wantAbsent string
	}{
		{name: "feature ID", parent: featureData.ID, wantField: "feature_id", wantAbsent: "task_id"},
		{name: "feature slug", parent: "checkout", wantField: "feature_id", wantAbsent: "task_id"},
		{name: "task ID", parent: taskData.ID, wantField: "task_id", wantAbsent: "feature_id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, _, addExit := runCLI(
				t,
				binary,
				dbPath,
				"document",
				"add",
				test.parent,
				"https://example.com/"+strings.ReplaceAll(test.name, " ", "-"),
			)
			if addExit != 0 || !value.OK {
				t.Fatalf("value=%+v exit=%d", value, addExit)
			}
			data := decodeDataObject(t, value)
			if _, ok := data[test.wantField]; !ok {
				t.Fatalf("data=%v, want %q", mapKeys(data), test.wantField)
			}
			if _, ok := data[test.wantAbsent]; ok {
				t.Fatalf("data=%v, did not want %q", mapKeys(data), test.wantAbsent)
			}
		})
	}

	for _, missingCase := range []struct {
		parent      string
		wantMessage string
	}{
		{parent: "missing-feature", wantMessage: "feature or task"},
		{parent: "T-999", wantMessage: "task"},
		{parent: strings.ToLower(taskData.ID), wantMessage: "feature or task"},
	} {
		missing, _, missingExit := runCLI(
			t,
			binary,
			dbPath,
			"document",
			"add",
			missingCase.parent,
			"https://example.com/missing",
		)
		if missingExit == 0 || missing.ErrorCode != "not_found" {
			t.Fatalf("parent=%q result=%+v exit=%d", missingCase.parent, missing, missingExit)
		}
		if !strings.Contains(missing.Error, missingCase.wantMessage) {
			t.Fatalf("parent=%q message=%q, want %q", missingCase.parent, missing.Error, missingCase.wantMessage)
		}
	}
}

func TestBlackBoxConfigSyncReadsAndUpdatesOneInterval(t *testing.T) {
	binary := buildCLI(t)
	root := t.TempDir()
	dbPath := filepath.Join(root, "unused.db")
	configPath := filepath.Join(root, "config.yaml")
	readInterval := func() int64 {
		value, stderr, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config", "sync")
		if exit != 0 || stderr != "" || !value.OK {
			t.Fatalf("sync read=%+v stderr=%q exit=%d", value, stderr, exit)
		}
		data := decodeDataObject(t, value)
		assertObjectKeys(t, data, "interval_seconds")
		var interval int64
		if err := json.Unmarshal(data["interval_seconds"], &interval); err != nil {
			t.Fatal(err)
		}
		return interval
	}
	if interval := readInterval(); interval != 3600 {
		t.Fatalf("default interval=%d, want 3600", interval)
	}
	updated, stderr, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config", "sync", "update", "900")
	if exit != 0 || stderr != "" || !updated.OK {
		t.Fatalf("sync update=%+v stderr=%q exit=%d", updated, stderr, exit)
	}
	if interval := readInterval(); interval != 900 {
		t.Fatalf("updated interval=%d, want 900", interval)
	}

	tooLow, _, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config", "sync", "update", "599")
	if exit == 0 || tooLow.ErrorCode != "invalid_config" {
		t.Fatalf("too low=%+v exit=%d", tooLow, exit)
	}
	nonNumeric, _, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config", "sync", "update", "later")
	if exit == 0 || nonNumeric.ErrorCode != "usage_error" {
		t.Fatalf("non-numeric=%+v exit=%d", nonNumeric, exit)
	}
	if interval := readInterval(); interval != 900 {
		t.Fatalf("failed updates changed interval to %d", interval)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("config sync opened database: %v", err)
	}
}

func TestBlackBoxJSONResponsesCoverEveryResponseCommand(t *testing.T) {
	binary := buildCLI(t)
	root := t.TempDir()
	dbPath := filepath.Join(root, "normal.db")
	dbConfigPath := filepath.Join(root, "normal-config.yaml")
	configPath := filepath.Join(root, "config.yaml")

	runDB := func(args ...string) map[string]json.RawMessage {
		commandArgs := []string{"--db", dbPath, "--config", dbConfigPath, "--json"}
		result := executeCLI(t, binary, "", append(commandArgs, args...)...)
		return assertNormalSuccess(t, result)
	}
	runConfig := func(args ...string) map[string]json.RawMessage {
		commandArgs := []string{
			"--db", filepath.Join(root, "config-unused.db"), "--config", configPath, "--json",
		}
		result := executeCLI(t, binary, "", append(commandArgs, args...)...)
		return assertNormalSuccess(t, result)
	}

	configList := executeCLI(
		t,
		binary,
		"",
		"--db",
		filepath.Join(root, "config-list-unused.db"),
		"--config",
		configPath,
		"--json",
		"config",
		"auth",
	)
	if configList.exit != 0 || configList.stderr != "" {
		t.Fatalf(
			"config auth list failed: stdout=%q stderr=%q exit=%d",
			configList.stdout,
			configList.stderr,
			configList.exit,
		)
	}
	if want := `{"auth_methods":[]}
`; configList.stdout != want {
		t.Fatalf("config auth list output=%q, want %q", configList.stdout, want)
	}
	assertCompactJSON(t, configList.stdout)
	assertDirectObjectKeys(t, decodeObject(t, []byte(configList.stdout), configList.stdout), "auth_methods")

	configShow := runConfig("config")
	assertDirectObject(t, configShow, "version", "github")
	assertDirectObject(t, runConfig("config", "path"), "path")
	assertDirectObject(t, runConfig("config", "validate"), "valid")
	assertDirectObject(t, runConfig("config", "host", "add", "ghe.example.com"), "host")
	assertDirectObjectKeys(t, runConfig("config", "host"), "hosts")
	assertDirectObject(
		t,
		runConfig("config", "host", "update", "ghe.example.com", "--api-url", "https://ghe.example.com/api/v3/"),
		"host",
	)
	assertDirectObject(t, runConfig(
		"config",
		"auth",
		"add",
		"work-gh",
		"ghe.example.com",
		"environment",
		"--variable",
		"GH_ENTERPRISE_TOKEN",
	), "id")
	assertDirectObjectKeys(t, runConfig("config", "auth"), "auth_methods")
	assertDirectObjectKeys(
		t,
		runConfig("config", "sync"),
		"interval_seconds",
	)
	assertDirectObjectKeys(
		t,
		runConfig("config", "sync", "update", "600"),
		"interval_seconds",
	)
	assertDirectObject(t, runConfig("config", "auth", "update", "work-gh", "--user", "HappyOnigiri"), "id")
	assertDirectObjectKeys(t, runConfig("config", "auth", "reorder", "work-gh"), "auth_methods")
	assertDirectObjectKeys(t, runConfig("config", "auth", "remove", "work-gh"), "removed")
	assertDirectObjectKeys(t, runConfig("config", "host", "remove", "ghe.example.com"), "removed")

	feature := runDB("feature", "create", "checkout", "Checkout")
	assertDirectObject(t, feature, "id", "slug")
	var featureID string
	if err := json.Unmarshal(feature["id"], &featureID); err != nil {
		t.Fatal(err)
	}
	assertDirectObjectKeys(t, runDB("feature"), "features")
	assertDirectObject(t, runDB("feature", featureID), "id", "slug")
	assertDirectObject(t, runDB("feature", "update", featureID, "--title", "Updated checkout"), "id", "slug")
	assertDirectObject(t, runDB("feature", "archive", featureID), "id", "slug")
	assertDirectObject(t, runDB("feature", "unarchive", featureID), "id", "slug")

	taskA := runDB("task", "create", featureID, "A")
	taskB := runDB("task", "create", featureID, "B")
	assertDirectObject(t, taskA, "id", "feature_id")
	assertDirectObject(t, taskB, "id", "feature_id")
	var taskAID, taskBID string
	if err := json.Unmarshal(taskA["id"], &taskAID); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(taskB["id"], &taskBID); err != nil {
		t.Fatal(err)
	}
	assertDirectObjectKeys(t, runDB("task"), "tasks")
	assertDirectObject(t, runDB("task", taskAID), "id", "feature_id")
	assertDirectObject(t, runDB("task", "update", taskAID, "--title", "Updated A"), "id", "feature_id")

	assertDirectObject(t, runDB("dependency", "add", taskAID, taskBID), "blocker_task_id", "blocked_task_id")
	assertDirectObjectKeys(t, runDB("dependency"), "dependencies")

	assertDirectObject(t, runDB(
		"pr",
		"attach",
		taskAID,
		"https://github.com/acme/payments/pull/42",
	), "task_id", "url")
	assertDirectObjectKeys(t, runDB("pr"), "pull_requests")

	document := runDB(
		"document",
		"add",
		taskAID,
		"https://example.com/checkout",
		"--kind",
		"url",
	)
	assertDirectObject(t, document, "id", "task_id")
	var documentID string
	if err := json.Unmarshal(document["id"], &documentID); err != nil {
		t.Fatal(err)
	}
	assertDirectObjectKeys(t, runDB("document"), "documents")

	planPath := filepath.Join(root, "plan.md")
	if err := os.WriteFile(planPath, []byte("# Checkout plan\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertDirectObject(t, runDB("plan", "set", taskAID, planPath), "task_id", "content")
	assertDirectObject(t, runDB("plan", taskAID), "task_id", "content")

	assertDirectObject(t, runDB("snapshot"), "features", "tasks", "dependencies", "pull_requests", "documents")
	assertDirectObjectKeys(t, runDB("graph", featureID), "feature", "tasks", "dependencies")
	assertDirectObjectKeys(t, runDB("ready"), "ready_tasks")
	assertDirectObjectKeys(t, runDB("reviews"), "review_waiting_tasks")
	assertDirectObjectKeys(t, runDB("conflicts"), "conflict_tasks")
	assertDirectObjectKeys(t, runDB("stale"), "stale_tasks")
	assertDirectObjectKeys(t, runDB("--github-fixture", "demo", "sync"), "failed", "succeeded")
	assertDirectObjectKeys(
		t,
		runDB("sync", "status"),
		"interval_seconds",
		"last_attempt_at",
		"last_updated_at",
		"succeeded",
		"failed",
	)
	assertDirectObjectKeys(t, runDB("validate"), "valid")

	assertDirectObjectKeys(t, runDB("document", "delete", documentID), "deleted")
	assertDirectObjectKeys(t, runDB("plan", "delete", taskAID), "deleted")
	assertDirectObjectKeys(t, runDB("pr", "detach", taskAID), "detached")
	assertDirectObjectKeys(t, runDB("dependency", "remove", taskAID, taskBID), "removed")
	assertDirectObjectKeys(t, runDB("task", "delete", taskBID), "deleted")
	assertDirectObjectKeys(t, runDB("task", "delete", taskAID, "--cascade"), "deleted")

	deletableFeature := runDB("feature", "create", "deletable", "Deletable")
	var deletableFeatureID string
	if err := json.Unmarshal(deletableFeature["id"], &deletableFeatureID); err != nil {
		t.Fatal(err)
	}
	assertDirectObjectKeys(t, runDB("feature", "delete", deletableFeatureID, "--cascade"), "deleted")

	seedDB := filepath.Join(root, "seed.db")
	seedConfig := filepath.Join(root, "seed-config.yaml")
	seed := executeCLI(
		t,
		binary,
		"",
		"--db",
		seedDB,
		"--config",
		seedConfig,
		"--json",
		"seed",
		"--features",
		"1",
		"--tasks",
		"1",
	)
	assertDirectObject(t, assertNormalSuccess(t, seed, "features", "tasks"), "features", "tasks")
}

func TestBlackBoxSchemaVersionDoesNotOpenStorage(t *testing.T) {
	binary := buildCLI(t)
	root := t.TempDir()
	dbPath := filepath.Join(root, "missing.db")
	configPath := filepath.Join(root, "missing", "config.yaml")
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "default text", args: []string{"schema-version"}, want: "Schema version: 2\n"},
		{name: "json before command", args: []string{"--json", "schema-version"}, want: `{"schema_version":"2"}
`},
		{name: "json after command", args: []string{"schema-version", "--json"}, want: `{"schema_version":"2"}
`},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.CommandContext(
				context.Background(),
				binary,
				"--db",
				dbPath,
				"--config",
				configPath,
			)
			command.Args = append(command.Args, test.args...)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			if err := command.Run(); err != nil {
				t.Fatalf("schema-version failed: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
			}
			if stdout.String() != test.want {
				t.Fatalf("schema-version output=%q, want %q", stdout.String(), test.want)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr=%q", stderr.String())
			}
		})
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("schema-version opened database: err=%v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("schema-version opened config: err=%v", err)
	}
}

func TestBlackBoxExplicitJSONErrorsUseJSONOnStderr(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "errors.db")
	command := exec.CommandContext(context.Background(), binary, "--db", dbPath, "--json", "task", "missing-task")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	var typed *exec.ExitError
	if !errors.As(err, &typed) || typed.ExitCode() != 1 {
		t.Fatalf("error=%v, want exit code 1", err)
	}
	value := decodeFailure(t, stderr.Bytes(), stderr.String())
	if value.OK || value.ErrorCode != "not_found" || value.Error == "" || value.Data != nil {
		t.Fatalf("error response=%+v", value)
	}
	assertEnvelopeKeys(t, value, "error")
	if stdout.Len() != 0 {
		t.Fatalf("stdout=%q", stdout.String())
	}
	assertCompactJSON(t, stderr.String())
	if strings.Contains(stderr.String(), "\n  ") {
		t.Fatalf("error response is not compact: %s", stderr.String())
	}
}

func TestBlackBoxConfigUsesSecureSecretFreeOutput(t *testing.T) {
	binary := buildCLI(t)
	root := t.TempDir()
	configPath := filepath.Join(root, "prx", "config.yaml")
	dbPath := filepath.Join(root, "unused.db")
	show, stderr, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config")
	if exit != 0 || stderr != "" || !show.OK {
		t.Fatalf("config show=%+v stderr=%q exit=%d", show, stderr, exit)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("config command opened database: err=%v", err)
	}
	if _, stderr, exit := runConfigCLI(
		t,
		binary,
		dbPath,
		configPath,
		"",
		"config",
		"host",
		"add",
		"ghe.example.com",
	); exit != 0 ||
		stderr != "" {
		t.Fatalf("host add stderr=%q exit=%d", stderr, exit)
	}
	added, stderr, exit := runConfigCLI(
		t,
		binary,
		dbPath,
		configPath,
		"github_pat_cli_secret\n",
		"config",
		"auth",
		"add",
		"ghe-inline",
		"ghe.example.com",
		"inline",
		"--token-stdin",
	)
	if exit != 0 || stderr != "" || !added.OK {
		t.Fatalf("auth add=%+v stderr=%q exit=%d", added, stderr, exit)
	}
	if strings.Contains(string(added.Data), "github_pat_cli_secret") {
		t.Fatalf("auth add response exposed token: %s", added.Data)
	}
	body, err := os.ReadFile(configPath)
	if err != nil || !strings.Contains(string(body), "github_pat_cli_secret") {
		t.Fatalf("config did not persist inline token: %s err=%v", body, err)
	}
	listed, stderr, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config", "auth")
	if exit != 0 || stderr != "" || strings.Contains(string(listed.Data), "github_pat_cli_secret") {
		t.Fatalf("auth list=%s stderr=%q exit=%d", listed.Data, stderr, exit)
	}
	removed, stderr, exit := runConfigCLI(
		t,
		binary,
		dbPath,
		configPath,
		"",
		"config",
		"host",
		"remove",
		"ghe.example.com",
	)
	if exit == 0 || stderr == "" || removed.Error == "" || !strings.Contains(removed.Error, "used by auth method") {
		t.Fatalf("referenced host removal=%+v stderr=%q exit=%d", removed, stderr, exit)
	}
	updated, stderr, exit := runConfigCLI(
		t,
		binary,
		dbPath,
		configPath,
		"",
		"config",
		"auth",
		"update",
		"ghe-inline",
		"--type",
		"environment",
		"--variable",
		"GH_ENTERPRISE_TOKEN",
	)
	if exit != 0 || stderr != "" || !updated.OK {
		t.Fatalf("auth update=%+v stderr=%q exit=%d", updated, stderr, exit)
	}
	body, err = os.ReadFile(configPath)
	if err != nil || strings.Contains(string(body), "github_pat_cli_secret") {
		t.Fatalf("changing away from inline retained token: %s err=%v", body, err)
	}
	reordered, stderr, exit := runConfigCLI(
		t,
		binary,
		dbPath,
		configPath,
		"",
		"config",
		"auth",
		"reorder",
		"ghe-inline",
	)
	if exit != 0 || stderr != "" {
		t.Fatalf("auth reorder stderr=%q exit=%d", stderr, exit)
	}
	if data := decodeDataObject(t, reordered); len(data) != 1 {
		t.Fatalf("auth reorder data keys=%v", mapKeys(data))
	} else if _, ok := data["auth_methods"]; !ok {
		t.Fatalf("auth reorder data keys=%v, want auth_methods", mapKeys(data))
	}
	if _, stderr, exit := runConfigCLI(
		t,
		binary,
		dbPath,
		configPath,
		"",
		"config",
		"auth",
		"remove",
		"ghe-inline",
	); exit != 0 ||
		stderr != "" {
		t.Fatalf("auth remove stderr=%q exit=%d", stderr, exit)
	}
	if _, stderr, exit := runConfigCLI(
		t,
		binary,
		dbPath,
		configPath,
		"",
		"config",
		"host",
		"remove",
		"ghe.example.com",
	); exit != 0 ||
		stderr != "" {
		t.Fatalf("host remove stderr=%q exit=%d", stderr, exit)
	}
	valid, stderr, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config", "validate")
	if exit != 0 || stderr != "" || !valid.OK {
		t.Fatalf("config validate=%+v stderr=%q exit=%d", valid, stderr, exit)
	}
}

func runConfigCLI(
	t *testing.T,
	binary, dbPath, configPath, input string,
	args ...string,
) (resultEnvelope, string, int) {
	t.Helper()
	commandArgs := []string{"--db", dbPath, "--config", configPath, "--json"}
	result := executeCLI(t, binary, input, append(commandArgs, args...)...)
	if result.exit != 0 {
		return decodeFailure(t, []byte(result.stderr), result.stderr), result.stderr, result.exit
	}
	return decodeResult(t, []byte(result.stdout), result.stdout), result.stderr, result.exit
}
