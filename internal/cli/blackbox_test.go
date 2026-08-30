package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func TestBlackBoxImplementationPlanCommands(t *testing.T) {
	binary := buildCLI(t)
	dbPath := filepath.Join(t.TempDir(), "plans.db")
	feature, _, exit := runCLI(t, binary, dbPath, "feature", "create", "--slug", "plans", "--title", "Plans")
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
		"--feature",
		featureData.ID,
		"--title",
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
	set, _, exit := runCLI(t, binary, dbPath, "implementation-plan", "set", taskData.ID, "--file", planPath)
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
	got, _, exit := runCLI(t, binary, dbPath, "implementation-plan", "get", taskData.ID)
	if exit != 0 || !got.OK {
		t.Fatalf("get result=%+v exit=%d", got, exit)
	}
	if err := json.Unmarshal(got.Data, &planData); err != nil {
		t.Fatal(err)
	}
	if planData.Content != content {
		t.Fatalf("get content=%q, want %q", planData.Content, content)
	}
	taskSnapshot, _, exit := runCLI(t, binary, dbPath, "task", "get", taskData.ID)
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
	deleted, _, exit := runCLI(t, binary, dbPath, "implementation-plan", "delete", taskData.ID)
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
	invalid, _, exit := runCLI(
		t,
		binary,
		dbPath,
		"implementation-plan",
		"set",
		taskData.ID,
		"--file",
		planPath,
		"--stdin",
	)
	if exit == 0 || invalid.Error == nil || invalid.Error.Code != "invalid_implementation_plan" {
		t.Fatalf("invalid input result=%+v exit=%d", invalid, exit)
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

func TestBlackBoxConfigUsesSecureSecretFreeOutput(t *testing.T) {
	binary := buildCLI(t)
	root := t.TempDir()
	configPath := filepath.Join(root, "prx", "config.yaml")
	dbPath := filepath.Join(root, "unused.db")
	show, stderr, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config", "show")
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
		"--host",
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
		"--id",
		"ghe-inline",
		"--host",
		"ghe.example.com",
		"--type",
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
	listed, stderr, exit := runConfigCLI(t, binary, dbPath, configPath, "", "config", "auth", "list")
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
	if exit == 0 || stderr != "" || removed.Error == nil || removed.Error.Code != "references_exist" {
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
	if _, stderr, exit := runConfigCLI(
		t,
		binary,
		dbPath,
		configPath,
		"",
		"config",
		"auth",
		"reorder",
		"ghe-inline",
	); exit != 0 ||
		stderr != "" {
		t.Fatalf("auth reorder stderr=%q exit=%d", stderr, exit)
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
	command := exec.CommandContext(context.Background(), binary, append(commandArgs, args...)...)
	command.Stdin = strings.NewReader(input)
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
