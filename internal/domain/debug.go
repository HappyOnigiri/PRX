package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// CLIResponseSchemaVersion is the version of the machine-readable CLI response
// schema. The diagnostic report and the CLI itself must agree on it, and the CLI
// cannot be imported from the packages that assemble the report.
const CLIResponseSchemaVersion = "2"

// Diagnostic output limits keep the report bounded and its ordering total, so a
// database with many failing repositories still produces deterministic output.
// A caller that needs everything reads `prx snapshot --json` instead.
const (
	DebugMaxErrorGroups      = 5
	DebugMaxTasksPerGroup    = 3
	DebugMaxRepositoryRows   = 20
	DebugMaxAuthCacheRows    = 50
	DebugMaxErrorMessageRune = 300
)

// debugOverdueFactor multiplies the configured interval before a missed
// automatic refresh is reported. One expired interval is normal between two
// commands; several in a row means nothing is refreshing.
const debugOverdueFactor = 3

// DebugEnvironmentNames are the variables that change how PRX resolves paths or
// authenticates. Only their presence is ever reported, never their values.
var DebugEnvironmentNames = []string{
	"PRX_DB",
	"PRX_CONFIG",
	"GITHUB_TOKEN",
	"GH_TOKEN",
	"GH_HOST",
	"GH_ENTERPRISE_TOKEN",
}

// DebugProblemCode identifies a problem the diagnostic report detected. The
// values are a public contract: they are stable, and a reader may branch on them.
type DebugProblemCode string

const (
	DebugProblemCodeStorageUnavailable         DebugProblemCode = "storage_unavailable"
	DebugProblemCodeSchemaVersionAheadOfBinary DebugProblemCode = "schema_version_ahead_of_binary"
	DebugProblemCodeDatabaseNotWritable        DebugProblemCode = "database_not_writable"
	DebugProblemCodeDatabaseIntegrityErrors    DebugProblemCode = "database_integrity_errors"
	DebugProblemCodeConfigUnreadable           DebugProblemCode = "config_unreadable"
	DebugProblemCodeConfigPermissionsTooOpen   DebugProblemCode = "config_permissions_too_open"
	DebugProblemCodeConfigUnknownFields        DebugProblemCode = "config_unknown_fields"
	DebugProblemCodeNoAuthMethodForHost        DebugProblemCode = "no_auth_method_for_host"
	DebugProblemCodeGitHubSyncRunError         DebugProblemCode = "github_sync_run_error"
	DebugProblemCodeGitHubSyncOverdue          DebugProblemCode = "github_sync_overdue"
	DebugProblemCodeGitHubSyncNeverCompleted   DebugProblemCode = "github_sync_never_completed"
	DebugProblemCodePullRequestsStale          DebugProblemCode = "pull_requests_stale"
)

// debugProblemSummaries explain each problem in the rendered report. The report
// text is shared by the CLI and the WebUI clipboard, so it stays English while
// the WebUI translates its own on-screen labels.
var debugProblemSummaries = map[DebugProblemCode]string{
	DebugProblemCodeStorageUnavailable:         "the database could not be opened, so most sections are unavailable",
	DebugProblemCodeSchemaVersionAheadOfBinary: "the database was migrated by a newer PRX than this binary",
	DebugProblemCodeDatabaseNotWritable:        "the database file cannot be written, so every mutation will fail",
	DebugProblemCodeDatabaseIntegrityErrors:    "stored dependency data failed validation",
	DebugProblemCodeConfigUnreadable:           "the configuration could not be loaded, so GitHub cannot authenticate",
	DebugProblemCodeConfigPermissionsTooOpen:   "the configuration file is readable by other accounts",
	DebugProblemCodeConfigUnknownFields:        "the configuration contains fields PRX ignores and will drop",
	DebugProblemCodeNoAuthMethodForHost:        "pull requests exist on a host that has no credential method",
	DebugProblemCodeGitHubSyncRunError:         "the latest recorded synchronization run failed",
	DebugProblemCodeGitHubSyncOverdue:          "the automatic interval expired well before this report",
	DebugProblemCodeGitHubSyncNeverCompleted:   "pull requests exist but no synchronization run has ever completed",
	DebugProblemCodePullRequestsStale:          "at least one pull request is holding stale state",
}

// DebugProblem is one detected problem with the evidence behind it and the
// command whose output explains it in full.
type DebugProblem struct {
	Code        DebugProblemCode `json:"code"`
	Target      string           `json:"target,omitempty"`
	Evidence    string           `json:"evidence,omitempty"`
	NextCommand string           `json:"next_command,omitempty"`
}

// DebugBuild describes the running PRX build.
type DebugBuild struct {
	Version     string `json:"version"`
	Development bool   `json:"development"`
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
}

// DebugRuntimeInput carries the process facts only the wiring layer knows.
type DebugRuntimeInput struct {
	Mode          string
	Demo          bool
	GitHubFixture bool
	ListenAddress string
	StartedAt     *time.Time
}

// DebugRuntime describes the process that produced the report.
type DebugRuntime struct {
	Mode          string     `json:"mode"`
	Demo          bool       `json:"demo"`
	GitHubFixture bool       `json:"github_fixture"`
	GeneratedAt   time.Time  `json:"generated_at"`
	TimeZone      string     `json:"time_zone"`
	ListenAddress string     `json:"listen_address,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	UptimeSeconds int64      `json:"uptime_seconds,omitempty"`
}

// DebugEnvironmentVariable reports whether one variable is set, never its value.
type DebugEnvironmentVariable struct {
	Name string `json:"name"`
	Set  bool   `json:"set"`
}

// DebugPathsInput carries the resolved locations and how they were selected.
type DebugPathsInput struct {
	DatabasePath       string
	DatabasePathSource string
	ConfigPath         string
	ConfigPathSource   string
	Demo               bool
}

// DebugPaths reports resolved locations and the ambient environment behind them.
type DebugPaths struct {
	DatabasePath         string                     `json:"database_path"`
	DatabasePathSource   string                     `json:"database_path_source"`
	DatabaseFileExists   bool                       `json:"database_file_exists"`
	ConfigPath           string                     `json:"config_path"`
	ConfigPathSource     string                     `json:"config_path_source"`
	ConfigFileExists     bool                       `json:"config_file_exists"`
	ConfigPermissions    string                     `json:"config_permissions,omitempty"`
	EnvironmentVariables []DebugEnvironmentVariable `json:"environment_variables"`
}

// DebugConfigHost is a host boundary as the report presents it.
type DebugConfigHost struct {
	Host       string `json:"host"`
	APIURL     string `json:"api_url"`
	GraphQLURL string `json:"graphql_url"`
}

// DebugConfigAuthMethod is a credential method without any secret material.
type DebugConfigAuthMethod struct {
	ID               string `json:"id"`
	Host             string `json:"host"`
	Type             string `json:"type"`
	SecretConfigured bool   `json:"secret_configured"`
}

// DebugConfigInput carries the configuration facts collected outside the domain,
// which cannot import the configuration package.
type DebugConfigInput struct {
	Version                 int
	AutoSyncIntervalSeconds int64
	Hosts                   []DebugConfigHost
	AuthMethods             []DebugConfigAuthMethod
	Warnings                []string
	LoadError               string
}

// DebugConfig reports the loaded configuration without secret material.
type DebugConfig struct {
	Version                 int                     `json:"version"`
	Valid                   bool                    `json:"valid"`
	Errors                  []string                `json:"errors"`
	Warnings                []string                `json:"warnings"`
	Hosts                   []DebugConfigHost       `json:"hosts"`
	AuthMethods             []DebugConfigAuthMethod `json:"auth_methods"`
	AutoSyncIntervalSeconds int64                   `json:"auto_sync_interval_seconds"`
}

// DebugDatabaseFile reports the on-disk state of the SQLite database.
type DebugDatabaseFile struct {
	Applicable   bool   `json:"applicable"`
	SizeBytes    int64  `json:"size_bytes"`
	WALPresent   bool   `json:"wal_present"`
	WALSizeBytes int64  `json:"wal_size_bytes"`
	SHMPresent   bool   `json:"shm_present"`
	Writable     bool   `json:"writable"`
	WriteError   string `json:"write_error,omitempty"`
}

// DebugStorageInput carries the storage facts persistence reported.
type DebugStorageInput struct {
	AppliedSchemaVersion  int
	EmbeddedSchemaVersion int
	IntegrityErrors       []string
	DatabaseFile          DebugDatabaseFile
	Error                 string
}

// DebugStorage reports schema state and database integrity.
type DebugStorage struct {
	AppliedSchemaVersion  int               `json:"applied_schema_version"`
	EmbeddedSchemaVersion int               `json:"embedded_schema_version"`
	IntegrityValid        bool              `json:"integrity_valid"`
	IntegrityErrors       []string          `json:"integrity_errors"`
	DatabaseFile          DebugDatabaseFile `json:"database_file"`
	CLISchemaVersion      string            `json:"cli_schema_version"`
	Error                 string            `json:"error,omitempty"`
}

// DebugCount is one named count in a breakdown.
type DebugCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DebugData reports stored record counts and their breakdowns.
type DebugData struct {
	Projects                 int          `json:"projects"`
	Features                 int          `json:"features"`
	Tasks                    int          `json:"tasks"`
	Dependencies             int          `json:"dependencies"`
	PullRequests             int          `json:"pull_requests"`
	Documents                int          `json:"documents"`
	ProjectStates            []DebugCount `json:"project_states"`
	FeatureStatuses          []DebugCount `json:"feature_statuses"`
	TaskDisplayStates        []DebugCount `json:"task_display_states"`
	TaskKinds                []DebugCount `json:"task_kinds"`
	PullRequestDisplayStates []DebugCount `json:"pull_request_display_states"`
	PullRequestHosts         []DebugCount `json:"pull_request_hosts"`
	DocumentKinds            []DebugCount `json:"document_kinds"`
	Error                    string       `json:"error,omitempty"`
}

// DebugSyncFailure counts synchronization failures within one host or repository.
type DebugSyncFailure struct {
	Scope string `json:"scope"`
	Count int    `json:"count"`
}

// DebugErrorGroup is one representative synchronization error and its tasks.
type DebugErrorGroup struct {
	Message        string   `json:"message"`
	Count          int      `json:"count"`
	TaskIDs        []string `json:"task_ids"`
	TotalTaskCount int      `json:"total_task_count"`
}

// DebugAuthCacheEntry records which credential last succeeded for a repository.
type DebugAuthCacheEntry struct {
	Host            string    `json:"host"`
	Owner           string    `json:"owner"`
	Repository      string    `json:"repository"`
	AuthMethodID    string    `json:"auth_method_id"`
	LastSucceededAt time.Time `json:"last_succeeded_at"`
}

// DebugGitHubSyncInput carries the synchronization facts collected elsewhere.
type DebugGitHubSyncInput struct {
	Status       GitHubSyncStatus
	PullRequests []PullRequest
	AuthCache    []DebugAuthCacheEntry
	Error        string
}

// DebugGitHubSync reports synchronization state and the failures behind it.
type DebugGitHubSync struct {
	Status                    GitHubSyncStatus      `json:"status"`
	NextRunAt                 *time.Time            `json:"next_run_at,omitempty"`
	Due                       bool                  `json:"due"`
	SecondsSinceLastUpdate    int64                 `json:"seconds_since_last_update,omitempty"`
	StalePullRequests         int                   `json:"stale_pull_requests"`
	FailedPullRequests        int                   `json:"failed_pull_requests"`
	HostFailures              []DebugSyncFailure    `json:"host_failures"`
	RepositoryFailures        []DebugSyncFailure    `json:"repository_failures"`
	OmittedRepositoryFailures int                   `json:"omitted_repository_failures"`
	ErrorGroups               []DebugErrorGroup     `json:"error_groups"`
	OmittedErrorGroups        int                   `json:"omitted_error_groups"`
	AuthCache                 []DebugAuthCacheEntry `json:"auth_cache"`
	OmittedAuthCacheEntries   int                   `json:"omitted_auth_cache_entries"`
	Error                     string                `json:"error,omitempty"`
}

// DebugReport is the whole diagnostic report, with the detected problems first.
type DebugReport struct {
	Problems   []DebugProblem  `json:"problems"`
	Build      DebugBuild      `json:"build"`
	Runtime    DebugRuntime    `json:"runtime"`
	Paths      DebugPaths      `json:"paths"`
	Config     DebugConfig     `json:"config"`
	Storage    DebugStorage    `json:"storage"`
	Records    DebugData       `json:"records"`
	GitHubSync DebugGitHubSync `json:"github_sync"`
}

// NewDebugBuild describes the running build. The version is passed in because
// the domain does not depend on the package that embeds it.
func NewDebugBuild(version string) DebugBuild {
	return DebugBuild{
		Version:     version,
		Development: strings.HasSuffix(version, "-dev"),
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
	}
}

// NewDebugRuntime records when the report was produced and, for a server, how
// long it has been listening.
func NewDebugRuntime(input DebugRuntimeInput, now time.Time) DebugRuntime {
	zone, _ := now.Local().Zone()
	result := DebugRuntime{
		Mode:          input.Mode,
		Demo:          input.Demo,
		GitHubFixture: input.GitHubFixture,
		GeneratedAt:   now.UTC(),
		TimeZone:      zone,
		ListenAddress: input.ListenAddress,
		StartedAt:     input.StartedAt,
	}
	if input.StartedAt != nil {
		result.UptimeSeconds = int64(now.Sub(*input.StartedAt).Seconds())
	}
	return result
}

// NewDebugPaths resolves the reported locations. A demo run reports the literal
// word demo instead of the temporary directory it was given.
func NewDebugPaths(input DebugPathsInput) DebugPaths {
	shortener := NewDebugPathShortener()
	result := DebugPaths{
		DatabasePath:         shortener.Path(input.DatabasePath),
		DatabasePathSource:   input.DatabasePathSource,
		ConfigPath:           shortener.Path(input.ConfigPath),
		ConfigPathSource:     input.ConfigPathSource,
		EnvironmentVariables: DebugEnvironmentVariables(),
	}
	if input.Demo {
		result.DatabasePath = "demo"
		result.ConfigPath = "demo"
		return result
	}
	if info, err := os.Stat(input.DatabasePath); err == nil && info.Mode().IsRegular() {
		result.DatabaseFileExists = true
	}
	if info, err := os.Stat(input.ConfigPath); err == nil && info.Mode().IsRegular() {
		result.ConfigFileExists = true
		result.ConfigPermissions = fmt.Sprintf("%04o", info.Mode().Perm())
	}
	return result
}

// DebugEnvironmentVariables reports which PRX-relevant variables are set. The
// diagnostic report is the one place that describes the ambient environment.
func DebugEnvironmentVariables() []DebugEnvironmentVariable {
	result := make([]DebugEnvironmentVariable, 0, len(DebugEnvironmentNames))
	for _, name := range DebugEnvironmentNames {
		result = append(result, DebugEnvironmentVariable{Name: name, Set: os.Getenv(name) != ""})
	}
	return result
}

// NewDebugConfig derives the configuration section. Both the server and a CLI
// run that could not open the database reach this single derivation.
func NewDebugConfig(input DebugConfigInput) DebugConfig {
	shortener := NewDebugPathShortener()
	result := DebugConfig{
		Version:                 input.Version,
		Valid:                   input.LoadError == "",
		Errors:                  []string{},
		Warnings:                debugStrings(shortener, input.Warnings),
		Hosts:                   input.Hosts,
		AuthMethods:             input.AuthMethods,
		AutoSyncIntervalSeconds: input.AutoSyncIntervalSeconds,
	}
	if input.LoadError != "" {
		result.Errors = append(result.Errors, shortener.Text(truncateDebugMessage(input.LoadError)))
	}
	if result.Hosts == nil {
		result.Hosts = []DebugConfigHost{}
	}
	if result.AuthMethods == nil {
		result.AuthMethods = []DebugConfigAuthMethod{}
	}
	return result
}

// NewDebugStorage derives the storage section from what persistence reported.
func NewDebugStorage(input DebugStorageInput) DebugStorage {
	shortener := NewDebugPathShortener()
	return DebugStorage{
		AppliedSchemaVersion:  input.AppliedSchemaVersion,
		EmbeddedSchemaVersion: input.EmbeddedSchemaVersion,
		IntegrityValid:        input.Error == "" && len(input.IntegrityErrors) == 0,
		IntegrityErrors:       debugStrings(shortener, input.IntegrityErrors),
		DatabaseFile:          input.DatabaseFile,
		CLISchemaVersion:      CLIResponseSchemaVersion,
		Error:                 shortener.Text(truncateDebugMessage(input.Error)),
	}
}

// NewDebugData counts the stored records. The counts come from the snapshot the
// rest of the application reads, so a reported count always matches what a
// caller can list.
func NewDebugData(snapshot Snapshot) DebugData {
	projects := newDebugTally()
	features := newDebugTally()
	taskStates := newDebugTally()
	taskKinds := newDebugTally()
	pullRequestStates := newDebugTally()
	pullRequestHosts := newDebugTally()
	documentKinds := newDebugTally()
	// An archived project makes every feature under it read-only, so the reader
	// needs the archived count to explain a refused write.
	for _, project := range snapshot.Projects {
		projects.add(debugProjectState(project.Archived))
	}
	for _, feature := range snapshot.Features {
		features.add(string(feature.DisplayStatus))
	}
	for _, task := range snapshot.Tasks {
		taskStates.add(string(task.DisplayState))
		taskKinds.add(string(task.Kind))
	}
	for _, pullRequest := range snapshot.PullRequests {
		pullRequestStates.add(string(pullRequest.DisplayState))
		pullRequestHosts.add(pullRequest.Host)
	}
	for _, document := range snapshot.Documents {
		documentKinds.add(string(document.Kind))
	}
	return DebugData{
		Projects:                 len(snapshot.Projects),
		Features:                 len(snapshot.Features),
		Tasks:                    len(snapshot.Tasks),
		Dependencies:             len(snapshot.Dependencies),
		PullRequests:             len(snapshot.PullRequests),
		Documents:                len(snapshot.Documents),
		ProjectStates:            projects.counts(),
		FeatureStatuses:          features.counts(),
		TaskDisplayStates:        taskStates.counts(),
		TaskKinds:                taskKinds.counts(),
		PullRequestDisplayStates: pullRequestStates.counts(),
		PullRequestHosts:         pullRequestHosts.counts(),
		DocumentKinds:            documentKinds.counts(),
	}
}

// NewDebugGitHubSync derives the synchronization section, including the bounded
// failure breakdowns a reader needs to tell one broken repository from a broken
// installation.
func NewDebugGitHubSync(input DebugGitHubSyncInput, now time.Time) DebugGitHubSync {
	shortener := NewDebugPathShortener()
	result := DebugGitHubSync{
		Status:             input.Status,
		HostFailures:       []DebugSyncFailure{},
		RepositoryFailures: []DebugSyncFailure{},
		ErrorGroups:        []DebugErrorGroup{},
		AuthCache:          []DebugAuthCacheEntry{},
		Error:              shortener.Text(truncateDebugMessage(input.Error)),
	}
	if input.Status.LastAttemptAt != nil && input.Status.IntervalSeconds > 0 {
		next := input.Status.LastAttemptAt.Add(time.Duration(input.Status.IntervalSeconds) * time.Second)
		result.NextRunAt = &next
		result.Due = !now.Before(next)
	} else {
		result.Due = true
	}
	if input.Status.LastUpdatedAt != nil {
		result.SecondsSinceLastUpdate = int64(now.Sub(*input.Status.LastUpdatedAt).Seconds())
	}
	hosts := newDebugTally()
	repositories := newDebugTally()
	groups := newDebugErrorGroups()
	for _, pullRequest := range input.PullRequests {
		if pullRequest.Stale {
			result.StalePullRequests++
		}
		if pullRequest.SyncError == "" {
			continue
		}
		result.FailedPullRequests++
		hosts.add(pullRequest.Host)
		repositories.add(fmt.Sprintf("%s/%s/%s", pullRequest.Host, pullRequest.Owner, pullRequest.Repository))
		groups.add(shortener.Text(pullRequest.SyncError), pullRequest.TaskID)
	}
	result.HostFailures = debugFailures(hosts.counts())
	repositoryRows := debugFailures(repositories.counts())
	result.RepositoryFailures, result.OmittedRepositoryFailures = limitDebugRows(repositoryRows, DebugMaxRepositoryRows)
	result.ErrorGroups, result.OmittedErrorGroups = limitDebugRows(groups.result(), DebugMaxErrorGroups)
	result.AuthCache, result.OmittedAuthCacheEntries = limitDebugRows(input.AuthCache, DebugMaxAuthCacheRows)
	return result
}

// DetectDebugProblems reports what a reader should look at first. Every problem
// carries the value that triggered it so the finding can be checked without
// re-reading the whole report.
func DetectDebugProblems(report DebugReport, now time.Time) []DebugProblem {
	problems := make([]DebugProblem, 0)
	problems = append(problems, detectDebugStorageProblems(report)...)
	problems = append(problems, detectDebugConfigProblems(report)...)
	problems = append(problems, detectDebugSyncProblems(report, now)...)
	return problems
}

func detectDebugStorageProblems(report DebugReport) []DebugProblem {
	problems := make([]DebugProblem, 0, 4)
	if report.Storage.Error != "" {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeStorageUnavailable,
			Target:      report.Paths.DatabasePath,
			Evidence:    report.Storage.Error,
			NextCommand: "prx debug --json",
		})
		return problems
	}
	if report.Storage.AppliedSchemaVersion > report.Storage.EmbeddedSchemaVersion {
		problems = append(problems, DebugProblem{
			Code:   DebugProblemCodeSchemaVersionAheadOfBinary,
			Target: report.Paths.DatabasePath,
			Evidence: fmt.Sprintf(
				"applied %d, this binary carries %d",
				report.Storage.AppliedSchemaVersion,
				report.Storage.EmbeddedSchemaVersion,
			),
			NextCommand: "install the PRX version that wrote this database",
		})
	}
	if file := report.Storage.DatabaseFile; file.Applicable && !file.Writable {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeDatabaseNotWritable,
			Target:      report.Paths.DatabasePath,
			Evidence:    file.WriteError,
			NextCommand: "ls -l " + report.Paths.DatabasePath,
		})
	}
	if len(report.Storage.IntegrityErrors) > 0 {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeDatabaseIntegrityErrors,
			Target:      report.Paths.DatabasePath,
			Evidence:    fmt.Sprintf("%d integrity errors", len(report.Storage.IntegrityErrors)),
			NextCommand: "prx validate",
		})
	}
	if report.Records.PullRequests > 0 && report.GitHubSync.StalePullRequests > 0 {
		problems = append(problems, DebugProblem{
			Code: DebugProblemCodePullRequestsStale,
			Evidence: fmt.Sprintf(
				"%d of %d pull requests are stale",
				report.GitHubSync.StalePullRequests,
				report.Records.PullRequests,
			),
			NextCommand: "prx stale",
		})
	}
	return problems
}

func detectDebugConfigProblems(report DebugReport) []DebugProblem {
	problems := make([]DebugProblem, 0, 3)
	if !report.Config.Valid {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeConfigUnreadable,
			Target:      report.Paths.ConfigPath,
			Evidence:    strings.Join(report.Config.Errors, "; "),
			NextCommand: "prx config validate",
		})
	}
	if mode := report.Paths.ConfigPermissions; mode != "" && !debugPermissionsArePrivate(mode) {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeConfigPermissionsTooOpen,
			Target:      report.Paths.ConfigPath,
			Evidence:    "mode " + mode,
			NextCommand: "chmod 600 " + report.Paths.ConfigPath,
		})
	}
	if len(report.Config.Warnings) > 0 {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeConfigUnknownFields,
			Target:      report.Paths.ConfigPath,
			Evidence:    fmt.Sprintf("%d warnings", len(report.Config.Warnings)),
			NextCommand: "prx config validate",
		})
	}
	for _, host := range debugHostsWithoutCredentials(report) {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeNoAuthMethodForHost,
			Target:      host,
			Evidence:    "pull requests exist on this host and no auth method is scoped to it",
			NextCommand: "prx config auth",
		})
	}
	return problems
}

func detectDebugSyncProblems(report DebugReport, now time.Time) []DebugProblem {
	problems := make([]DebugProblem, 0, 3)
	sync := report.GitHubSync
	if sync.Status.Error != "" {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeGitHubSyncRunError,
			Evidence:    sync.Status.Error,
			NextCommand: "prx sync status --json",
		})
	}
	if report.Records.PullRequests > 0 && sync.Status.LastUpdatedAt == nil {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeGitHubSyncNeverCompleted,
			Evidence:    fmt.Sprintf("%d pull requests and no completed run", report.Records.PullRequests),
			NextCommand: "prx sync",
		})
	}
	overdueAfter := sync.Status.IntervalSeconds * debugOverdueFactor
	if report.Records.PullRequests > 0 && sync.Status.LastUpdatedAt != nil && overdueAfter > 0 &&
		now.Sub(*sync.Status.LastUpdatedAt) > time.Duration(overdueAfter)*time.Second {
		problems = append(problems, DebugProblem{
			Code: DebugProblemCodeGitHubSyncOverdue,
			Evidence: fmt.Sprintf(
				"last completed %ds ago with a %ds interval",
				sync.SecondsSinceLastUpdate,
				sync.Status.IntervalSeconds,
			),
			NextCommand: "prx sync",
		})
	}
	return problems
}

// debugHostsWithoutCredentials reports hosts that carry pull requests and have
// no credential method. An omitted method list keeps GitHub.com's compatibility
// defaults, so an empty configuration is not by itself a problem.
func debugHostsWithoutCredentials(report DebugReport) []string {
	if !report.Config.Valid || len(report.Config.AuthMethods) == 0 {
		return nil
	}
	configured := make(map[string]bool, len(report.Config.AuthMethods))
	for _, method := range report.Config.AuthMethods {
		configured[method.Host] = true
	}
	result := make([]string, 0)
	for _, host := range report.Records.PullRequestHosts {
		if host.Name != "" && !configured[host.Name] {
			result = append(result, host.Name)
		}
	}
	sort.Strings(result)
	return result
}

// debugPermissionsArePrivate mirrors the configuration loader's own rule, which
// rejects any mode that grants access to the group or to others.
func debugPermissionsArePrivate(mode string) bool {
	var parsed int64
	if _, err := fmt.Sscanf(mode, "%o", &parsed); err != nil {
		return true
	}
	return parsed&0o077 == 0
}

// DebugPathShortener replaces the home directory with ~ so a report can be
// pasted into an issue without disclosing the account name.
type DebugPathShortener struct {
	home string
}

// NewDebugPathShortener reads the home directory once. A system without one
// leaves every path untouched rather than guessing a prefix.
func NewDebugPathShortener() DebugPathShortener {
	home, err := os.UserHomeDir()
	if err != nil {
		return DebugPathShortener{}
	}
	return DebugPathShortener{home: home}
}

// Path shortens one path. It requires an exact match or a separator after the
// home directory so a sibling such as /home/user2 is left alone.
func (s DebugPathShortener) Path(value string) string {
	if s.home == "" || value == "" {
		return value
	}
	if value == s.home {
		return "~"
	}
	separator := string(filepath.Separator)
	if strings.HasPrefix(value, s.home+separator) {
		return "~" + strings.TrimPrefix(value, s.home)
	}
	return value
}

// Text shortens the paths embedded in a message. Failures such as a directory
// that could not be created carry the path inside their text, so shortening the
// structured fields alone would still disclose it.
func (s DebugPathShortener) Text(value string) string {
	if s.home == "" || value == "" {
		return value
	}
	var builder strings.Builder
	for index := 0; index < len(value); {
		if strings.HasPrefix(value[index:], s.home) && !continuesPathSegment(value[index+len(s.home):]) {
			builder.WriteString("~")
			index += len(s.home)
			continue
		}
		builder.WriteByte(value[index])
		index++
	}
	return builder.String()
}

// continuesPathSegment reports whether the text following a home-directory match
// extends the name, as in /home/user2. That is a different directory and must
// keep its own name.
func continuesPathSegment(rest string) bool {
	if rest == "" {
		return false
	}
	character := rest[0]
	return character == '-' || character == '_' || character == '.' ||
		(character >= '0' && character <= '9') ||
		(character >= 'a' && character <= 'z') ||
		(character >= 'A' && character <= 'Z')
}

func debugStrings(shortener DebugPathShortener, values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, shortener.Text(truncateDebugMessage(value)))
	}
	return result
}

// truncateDebugMessage bounds one message so a single pathological error cannot
// dominate the report.
func truncateDebugMessage(value string) string {
	runes := []rune(value)
	if len(runes) <= DebugMaxErrorMessageRune {
		return value
	}
	return string(runes[:DebugMaxErrorMessageRune]) + "…"
}

// debugTally counts values without letting map iteration order reach the output.
type debugTally struct {
	counted map[string]int
}

func newDebugTally() *debugTally {
	return &debugTally{counted: map[string]int{}}
}

func (t *debugTally) add(name string) { t.counted[name]++ }

// debugProjectState names the only state a project has, because a project is
// outside the two-layer status rule features and tasks share.
func debugProjectState(archived bool) string {
	if archived {
		return "archived"
	}
	return "active"
}

// counts orders by descending count and then by name, so equal counts still
// produce one deterministic order.
func (t *debugTally) counts() []DebugCount {
	result := make([]DebugCount, 0, len(t.counted))
	for name, count := range t.counted {
		result = append(result, DebugCount{Name: name, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func debugFailures(counts []DebugCount) []DebugSyncFailure {
	result := make([]DebugSyncFailure, 0, len(counts))
	for _, count := range counts {
		result = append(result, DebugSyncFailure{Scope: count.Name, Count: count.Count})
	}
	return result
}

func limitDebugRows[T any](values []T, limit int) ([]T, int) {
	if len(values) <= limit {
		return values, 0
	}
	return values[:limit], len(values) - limit
}

// debugErrorNoise matches the parts of a synchronization error that differ per
// pull request. Without removing them every message is unique and grouping
// reports one representative per failure instead of one per shape.
var debugErrorNoise = regexp.MustCompile(`https?://\S+|"[^"]*"|'[^']*'|#?\d+`)

type debugErrorGroups struct {
	order  []string
	groups map[string]*DebugErrorGroup
}

func newDebugErrorGroups() *debugErrorGroups {
	return &debugErrorGroups{groups: map[string]*DebugErrorGroup{}}
}

func (g *debugErrorGroups) add(message, taskID string) {
	key := debugErrorNoise.ReplaceAllString(message, "*")
	group, ok := g.groups[key]
	if !ok {
		group = &DebugErrorGroup{Message: truncateDebugMessage(message), TaskIDs: []string{}}
		g.groups[key] = group
		g.order = append(g.order, key)
	}
	group.Count++
	group.TotalTaskCount++
	if len(group.TaskIDs) < DebugMaxTasksPerGroup {
		group.TaskIDs = append(group.TaskIDs, taskID)
	}
}

// result orders by descending count and then by message so the limit drops the
// same groups on every run.
func (g *debugErrorGroups) result() []DebugErrorGroup {
	result := make([]DebugErrorGroup, 0, len(g.order))
	for _, key := range g.order {
		result = append(result, *g.groups[key])
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Message < result[j].Message
	})
	return result
}
