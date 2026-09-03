package domain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FormatDebugReport renders the report the CLI prints and the WebUI copies to
// the clipboard. Both surfaces share this single rendering so a report pasted
// into an issue reads the same whichever one produced it.
func FormatDebugReport(report DebugReport) string {
	out := &debugText{}
	out.line("PRX diagnostic report")
	writeDebugProblems(out, report.Problems)
	writeDebugBuild(out, report.Build)
	writeDebugRuntime(out, report.Runtime)
	writeDebugPaths(out, report.Paths)
	writeDebugConfig(out, report.Config)
	writeDebugStorage(out, report.Storage)
	writeDebugData(out, report.Records)
	writeDebugGitHubSync(out, report.GitHubSync)
	return out.String()
}

func writeDebugProblems(out *debugText, problems []DebugProblem) {
	out.section("problems")
	if len(problems) == 0 {
		out.field("detected", "none")
		return
	}
	for _, problem := range problems {
		out.item(string(problem.Code) + ": " + debugProblemSummaries[problem.Code])
		out.subField("target", problem.Target)
		out.subField("evidence", problem.Evidence)
		out.subField("next", problem.NextCommand)
	}
}

func writeDebugBuild(out *debugText, build DebugBuild) {
	out.section("build")
	out.field("version", build.Version)
	out.field("development", debugYesNo(build.Development))
	out.field("go", build.GoVersion)
	out.field("platform", build.OS+"/"+build.Arch)
}

func writeDebugRuntime(out *debugText, value DebugRuntime) {
	out.section("runtime")
	out.field("mode", value.Mode)
	out.field("demo", debugYesNo(value.Demo))
	out.field("github_fixture", debugYesNo(value.GitHubFixture))
	out.field("generated_at", debugTime(&value.GeneratedAt))
	out.field("time_zone", value.TimeZone)
	if value.ListenAddress != "" {
		out.field("listen_address", value.ListenAddress)
	}
	if value.StartedAt != nil {
		out.field("started_at", debugTime(value.StartedAt))
		out.field("uptime_seconds", strconv.FormatInt(value.UptimeSeconds, 10))
	}
}

func writeDebugPaths(out *debugText, paths DebugPaths) {
	out.section("paths")
	out.field("database_path", paths.DatabasePath)
	out.field("database_path_source", paths.DatabasePathSource)
	out.field("database_file_exists", debugYesNo(paths.DatabaseFileExists))
	out.field("config_path", paths.ConfigPath)
	out.field("config_path_source", paths.ConfigPathSource)
	out.field("config_file_exists", debugYesNo(paths.ConfigFileExists))
	if paths.ConfigPermissions != "" {
		out.field("config_permissions", paths.ConfigPermissions)
	}
	out.field("environment", "")
	for _, variable := range paths.EnvironmentVariables {
		out.subKeyValue(variable.Name, debugSetUnset(variable.Set))
	}
}

func writeDebugConfig(out *debugText, config DebugConfig) {
	out.section("config")
	out.field("version", strconv.Itoa(config.Version))
	out.field("valid", debugYesNo(config.Valid))
	out.field("auto_sync_interval_seconds", strconv.FormatInt(config.AutoSyncIntervalSeconds, 10))
	writeDebugStringList(out, "errors", config.Errors)
	writeDebugStringList(out, "warnings", config.Warnings)
	out.field("hosts", debugEmptyWhenZero(len(config.Hosts)))
	for _, host := range config.Hosts {
		out.subItem(host.Host)
		out.subSubField("api_url", host.APIURL)
		out.subSubField("graphql_url", host.GraphQLURL)
	}
	out.field("auth_methods", debugEmptyWhenZero(len(config.AuthMethods)))
	for _, method := range config.AuthMethods {
		out.subItem(method.ID)
		out.subSubField("host", method.Host)
		out.subSubField("type", method.Type)
		out.subSubField("secret_configured", debugYesNo(method.SecretConfigured))
	}
}

func writeDebugStorage(out *debugText, storage DebugStorage) {
	out.section("storage")
	if storage.Error != "" {
		out.field("error", storage.Error)
	}
	out.field("applied_schema_version", strconv.Itoa(storage.AppliedSchemaVersion))
	out.field("embedded_schema_version", strconv.Itoa(storage.EmbeddedSchemaVersion))
	out.field("cli_schema_version", storage.CLISchemaVersion)
	out.field("integrity_valid", debugYesNo(storage.IntegrityValid))
	writeDebugStringList(out, "integrity_errors", storage.IntegrityErrors)
	file := storage.DatabaseFile
	if !file.Applicable {
		out.field("database_file", "not applicable")
		return
	}
	out.field("database_file", "")
	out.subKeyValue("size_bytes", strconv.FormatInt(file.SizeBytes, 10))
	out.subKeyValue("wal_present", debugYesNo(file.WALPresent))
	out.subKeyValue("wal_size_bytes", strconv.FormatInt(file.WALSizeBytes, 10))
	out.subKeyValue("shm_present", debugYesNo(file.SHMPresent))
	out.subKeyValue("writable", debugYesNo(file.Writable))
	if file.WriteError != "" {
		out.subKeyValue("write_error", file.WriteError)
	}
}

func writeDebugData(out *debugText, data DebugData) {
	out.section("records")
	if data.Error != "" {
		out.field("error", data.Error)
	}
	out.field("projects", strconv.Itoa(data.Projects))
	out.field("features", strconv.Itoa(data.Features))
	out.field("tasks", strconv.Itoa(data.Tasks))
	out.field("dependencies", strconv.Itoa(data.Dependencies))
	out.field("pull_requests", strconv.Itoa(data.PullRequests))
	out.field("documents", strconv.Itoa(data.Documents))
	writeDebugCounts(out, "project_states", data.ProjectStates)
	writeDebugCounts(out, "feature_statuses", data.FeatureStatuses)
	writeDebugCounts(out, "task_display_states", data.TaskDisplayStates)
	writeDebugCounts(out, "task_kinds", data.TaskKinds)
	writeDebugCounts(out, "pull_request_display_states", data.PullRequestDisplayStates)
	writeDebugCounts(out, "pull_request_hosts", data.PullRequestHosts)
	writeDebugCounts(out, "document_kinds", data.DocumentKinds)
}

func writeDebugGitHubSync(out *debugText, sync DebugGitHubSync) {
	out.section("github_sync")
	if sync.Error != "" {
		out.field("error", sync.Error)
	}
	out.field("interval_seconds", strconv.FormatInt(sync.Status.IntervalSeconds, 10))
	out.field("last_attempt_at", debugTime(sync.Status.LastAttemptAt))
	out.field("last_updated_at", debugTime(sync.Status.LastUpdatedAt))
	out.field("seconds_since_last_update", strconv.FormatInt(sync.SecondsSinceLastUpdate, 10))
	out.field("next_run_at", debugTime(sync.NextRunAt))
	out.field("due", debugYesNo(sync.Due))
	out.field("succeeded", strconv.Itoa(sync.Status.Succeeded))
	out.field("failed", strconv.Itoa(sync.Status.Failed))
	out.field("run_error", debugValue(sync.Status.Error))
	out.field("stale_pull_requests", strconv.Itoa(sync.StalePullRequests))
	out.field("failed_pull_requests", strconv.Itoa(sync.FailedPullRequests))
	writeDebugFailures(out, "host_failures", sync.HostFailures, 0)
	writeDebugFailures(out, "repository_failures", sync.RepositoryFailures, sync.OmittedRepositoryFailures)
	writeDebugErrorGroups(out, sync)
	writeDebugAuthCache(out, sync)
}

func writeDebugErrorGroups(out *debugText, sync DebugGitHubSync) {
	out.field("error_groups", debugOmitted(len(sync.ErrorGroups), sync.OmittedErrorGroups))
	for _, group := range sync.ErrorGroups {
		out.subItem(group.Message)
		out.subSubField("count", strconv.Itoa(group.Count))
		out.subSubField("task_ids", strings.Join(group.TaskIDs, ", "))
		out.subSubField("total_task_count", strconv.Itoa(group.TotalTaskCount))
	}
}

func writeDebugAuthCache(out *debugText, sync DebugGitHubSync) {
	out.field("auth_cache", debugOmitted(len(sync.AuthCache), sync.OmittedAuthCacheEntries))
	for _, entry := range sync.AuthCache {
		out.subKeyValue(
			fmt.Sprintf("%s/%s/%s", entry.Host, entry.Owner, entry.Repository),
			entry.AuthMethodID+" at "+debugTime(&entry.LastSucceededAt),
		)
	}
}

func writeDebugCounts(out *debugText, key string, counts []DebugCount) {
	out.field(key, debugEmptyWhenZero(len(counts)))
	for _, count := range counts {
		out.subKeyValue(count.Name, strconv.Itoa(count.Count))
	}
}

func writeDebugFailures(out *debugText, key string, failures []DebugSyncFailure, omitted int) {
	out.field(key, debugOmitted(len(failures), omitted))
	for _, failure := range failures {
		out.subKeyValue(failure.Scope, strconv.Itoa(failure.Count))
	}
}

func writeDebugStringList(out *debugText, key string, values []string) {
	out.field(key, debugEmptyWhenZero(len(values)))
	for _, value := range values {
		out.subItem(value)
	}
}

// debugText builds the report body. Section headers sit at the left margin and
// every value below one is indented, so a reader can tell where a section ends
// even when a value wraps in a terminal.
type debugText struct {
	builder strings.Builder
}

func (t *debugText) line(value string) {
	t.builder.WriteString(value)
	t.builder.WriteString("\n")
}

func (t *debugText) section(name string) {
	t.builder.WriteString("\n")
	t.line(name + ":")
}

func (t *debugText) field(key, value string) {
	if value == "" {
		t.line("  " + key + ":")
		return
	}
	t.line("  " + key + ": " + value)
}

func (t *debugText) item(value string) { t.line("  - " + value) }

func (t *debugText) subField(key, value string) {
	if value == "" {
		return
	}
	t.line("    " + key + ": " + value)
}

func (t *debugText) subItem(value string) { t.line("    - " + value) }

func (t *debugText) subKeyValue(key, value string) { t.line("    " + key + ": " + value) }

func (t *debugText) subSubField(key, value string) {
	if value == "" {
		return
	}
	t.line("      " + key + ": " + value)
}

func (t *debugText) String() string { return t.builder.String() }

func debugYesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func debugSetUnset(value bool) string {
	if value {
		return "set"
	}
	return "unset"
}

func debugValue(value string) string {
	if value == "" {
		return "none"
	}
	return value
}

func debugTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "never"
	}
	return value.UTC().Format(time.RFC3339)
}

func debugEmptyWhenZero(count int) string {
	if count == 0 {
		return "none"
	}
	return ""
}

// debugOmitted reports how many rows an output limit dropped, so a truncated
// breakdown never reads as a complete one.
func debugOmitted(shown, omitted int) string {
	if omitted > 0 {
		return fmt.Sprintf("%d shown, %d omitted", shown, omitted)
	}
	return debugEmptyWhenZero(shown)
}

// SortedDebugProblemCodes lists every problem code in a stable order. Tests and
// enum translation tables use it to prove no member was forgotten.
func SortedDebugProblemCodes() []DebugProblemCode {
	result := make([]DebugProblemCode, 0, len(debugProblemSummaries))
	for code := range debugProblemSummaries {
		result = append(result, code)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
