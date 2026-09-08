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

// CLIResponseSchemaVersion は機械可読な CLI レスポンススキーマのバージョン。診断レポートと
// CLI 本体で一致させる必要があるが、レポートを組み立てるパッケージから CLI は import
// できないためここに置く。
const CLIResponseSchemaVersion = "2"

// 診断出力の上限はレポートの大きさと順序を確定させ、失敗中のリポジトリが多いデータベースでも
// 決定的な出力にする。すべてが必要な呼び出し側は代わりに `prx snapshot --json` を読む。
const (
	DebugMaxErrorGroups      = 5
	DebugMaxTasksPerGroup    = 3
	DebugMaxRepositoryRows   = 20
	DebugMaxAuthCacheRows    = 50
	DebugMaxErrorMessageRune = 300
)

// debugOverdueFactor は自動更新の遅延を報告する前に、設定された間隔に掛ける倍率。
// コマンド間で 1 回分の間隔切れは正常だが、何度も続くなら更新が動いていない。
const debugOverdueFactor = 3

// DebugEnvironmentNames は PRX のパス解決や認証を変える環境変数。
// 報告するのは設定の有無だけで、値は決して報告しない。
var DebugEnvironmentNames = []string{
	"PRX_DB",
	"PRX_CONFIG",
	"PRX_RUN_DIR",
	"GITHUB_TOKEN",
	"GH_TOKEN",
	"GH_HOST",
	"GH_ENTERPRISE_TOKEN",
}

// DebugProblemCode は診断レポートが検出した問題を識別する。値は公開契約であり、
// 安定していて、読み手が分岐に使ってよい。
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
	DebugProblemCodeDaemonPlistStale           DebugProblemCode = "daemon_plist_stale"
	DebugProblemCodeDaemonNotRunning           DebugProblemCode = "daemon_not_running"
	DebugProblemCodeDaemonBinaryOutdated       DebugProblemCode = "daemon_binary_outdated"
)

// debugProblemSummaries は出力されるレポート内で各問題を説明する。レポート本文は CLI と
// WebUI のクリップボードで共通なので英語のままとし、WebUI は画面表示のラベルだけを訳す。
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
	DebugProblemCodeDaemonPlistStale:           "the installed LaunchAgent does not match this PRX binary",
	DebugProblemCodeDaemonNotRunning:           "the LaunchAgent is installed but no server is running",
	DebugProblemCodeDaemonBinaryOutdated:       "the running server started from a different PRX binary",
}

// DebugProblem は検出した 1 件の問題と、その根拠、および詳細を出力するコマンドを表す。
type DebugProblem struct {
	Code        DebugProblemCode `json:"code"`
	Target      string           `json:"target,omitempty"`
	Evidence    string           `json:"evidence,omitempty"`
	NextCommand string           `json:"next_command,omitempty"`
}

// DebugBuild は実行中の PRX ビルドを表す。
type DebugBuild struct {
	Version     string `json:"version"`
	Development bool   `json:"development"`
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
}

// DebugRuntimeInput は配線層だけが知るプロセスの情報を運ぶ。
type DebugRuntimeInput struct {
	Mode          string
	Demo          bool
	GitHubFixture bool
	ListenAddress string
	StartedAt     *time.Time
}

// DebugRuntime はレポートを生成したプロセスを表す。
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

// DebugEnvironmentVariable は環境変数が設定済みかだけを表し、値は持たない。
type DebugEnvironmentVariable struct {
	Name string `json:"name"`
	Set  bool   `json:"set"`
}

// DebugPathsInput は解決済みのパスと、それがどう選ばれたかを運ぶ。
type DebugPathsInput struct {
	DatabasePath       string
	DatabasePathSource string
	ConfigPath         string
	ConfigPathSource   string
	Demo               bool
}

// DebugPaths は解決済みのパスと、その背後にある環境の状態を報告する。
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

// DebugConfigHost はレポートが提示する host の境界を表す。
type DebugConfigHost struct {
	Host       string `json:"host"`
	APIURL     string `json:"api_url"`
	GraphQLURL string `json:"graphql_url"`
}

// DebugConfigAuthMethod は秘密情報を含まない認証方式を表す。
type DebugConfigAuthMethod struct {
	ID               string `json:"id"`
	Host             string `json:"host"`
	Type             string `json:"type"`
	SecretConfigured bool   `json:"secret_configured"`
}

// DebugConfigPrompt は保存済みのエージェント prompt テンプレート 1 件を表す。本文はユーザー
// 作成で数キロバイトあるため、レポートは本文を載せず組み込みテキストと一致するかだけを示す。
type DebugConfigPrompt struct {
	Customized bool `json:"customized"`
	Bytes      int  `json:"bytes"`
}

// DebugConfigPrompts は保存済みテンプレートごとに 1 エントリを持つ。
type DebugConfigPrompts struct {
	Design         DebugConfigPrompt `json:"design"`
	Implementation DebugConfigPrompt `json:"implementation"`
	Batch          DebugConfigPrompt `json:"batch"`
}

// DebugConfigInput は domain の外で集めた設定情報を運ぶ。domain は設定パッケージを
// import できないため。
type DebugConfigInput struct {
	Version                 int
	AutoSyncIntervalSeconds int64
	Hosts                   []DebugConfigHost
	AuthMethods             []DebugConfigAuthMethod
	Prompts                 DebugConfigPrompts
	Warnings                []string
	LoadError               string
}

// DebugConfig は読み込んだ設定を、秘密情報を除いて報告する。
type DebugConfig struct {
	Version                 int                     `json:"version"`
	Valid                   bool                    `json:"valid"`
	Errors                  []string                `json:"errors"`
	Warnings                []string                `json:"warnings"`
	Hosts                   []DebugConfigHost       `json:"hosts"`
	AuthMethods             []DebugConfigAuthMethod `json:"auth_methods"`
	AutoSyncIntervalSeconds int64                   `json:"auto_sync_interval_seconds"`
	Prompts                 DebugConfigPrompts      `json:"prompts"`
}

// DebugDatabaseFile は SQLite データベースのディスク上の状態を報告する。
type DebugDatabaseFile struct {
	Applicable   bool   `json:"applicable"`
	SizeBytes    int64  `json:"size_bytes"`
	WALPresent   bool   `json:"wal_present"`
	WALSizeBytes int64  `json:"wal_size_bytes"`
	SHMPresent   bool   `json:"shm_present"`
	Writable     bool   `json:"writable"`
	WriteError   string `json:"write_error,omitempty"`
}

// DebugStorageInput は persistence が報告したストレージの情報を運ぶ。
type DebugStorageInput struct {
	AppliedSchemaVersion  int
	EmbeddedSchemaVersion int
	IntegrityErrors       []string
	DatabaseFile          DebugDatabaseFile
	Error                 string
}

// DebugStorage はスキーマの状態とデータベースの整合性を報告する。
type DebugStorage struct {
	AppliedSchemaVersion  int               `json:"applied_schema_version"`
	EmbeddedSchemaVersion int               `json:"embedded_schema_version"`
	IntegrityValid        bool              `json:"integrity_valid"`
	IntegrityErrors       []string          `json:"integrity_errors"`
	DatabaseFile          DebugDatabaseFile `json:"database_file"`
	CLISchemaVersion      string            `json:"cli_schema_version"`
	Error                 string            `json:"error,omitempty"`
}

// DebugCount は内訳の中の名前付きカウント 1 件を表す。
type DebugCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DebugData は保存済みレコード数とその内訳を報告する。
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
	PullRequestDisplayStates []DebugCount `json:"pull_request_display_states"`
	PullRequestHosts         []DebugCount `json:"pull_request_hosts"`
	DocumentKinds            []DebugCount `json:"document_kinds"`
	Error                    string       `json:"error,omitempty"`
}

// DebugSyncFailure は host または repository 単位の同期失敗数を表す。
type DebugSyncFailure struct {
	Scope string `json:"scope"`
	Count int    `json:"count"`
}

// DebugErrorGroup は代表的な同期エラー 1 件と、それに該当する task を表す。
type DebugErrorGroup struct {
	Message        string   `json:"message"`
	Count          int      `json:"count"`
	TaskIDs        []string `json:"task_ids"`
	TotalTaskCount int      `json:"total_task_count"`
}

// DebugAuthCacheEntry は repository ごとに最後に成功した認証情報を記録する。
type DebugAuthCacheEntry struct {
	Host            string    `json:"host"`
	Owner           string    `json:"owner"`
	Repository      string    `json:"repository"`
	AuthMethodID    string    `json:"auth_method_id"`
	LastSucceededAt time.Time `json:"last_succeeded_at"`
}

// DebugGitHubSyncInput は他所で集めた同期の情報を運ぶ。
type DebugGitHubSyncInput struct {
	Status       GitHubSyncStatus
	PullRequests []PullRequest
	AuthCache    []DebugAuthCacheEntry
	Error        string
}

// DebugGitHubSync は同期の状態と、その背後にある失敗を報告する。
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

// DebugDaemonInput は配線層だけが知る常駐の状態を運ぶ。domain は launchd も稼働記録も
// import できないため。
type DebugDaemonInput struct {
	Supported     bool
	Installed     bool
	PlistStatus   string
	PlistPath     string
	LogPath       string
	Running       bool
	Address       string
	PID           int
	Version       string
	BinaryMatches bool
	Demo          bool
	Error         string
}

// DebugDaemon は LaunchAgent と稼働中サーバーを報告する。これは別プロセスの記述なので、
// このレポートを生成したプロセスを表す DebugRuntime とは分けている。混ぜると
// runtime の mode の意味が壊れる。
type DebugDaemon struct {
	Supported     bool   `json:"supported"`
	Installed     bool   `json:"installed"`
	PlistStatus   string `json:"plist_status"`
	PlistPath     string `json:"plist_path"`
	LogPath       string `json:"log_path"`
	Running       bool   `json:"running"`
	Address       string `json:"address"`
	PID           int    `json:"pid"`
	Version       string `json:"version"`
	BinaryMatches bool   `json:"binary_matches"`
	Error         string `json:"error,omitempty"`
}

// NewDebugDaemon は常駐セクションを導出する。パスはホームを ~ に短縮し、demo では
// 使わないファイルを指さないよう demo と報告する。
func NewDebugDaemon(input DebugDaemonInput) DebugDaemon {
	shortener := NewDebugPathShortener()
	result := DebugDaemon{
		Supported:     input.Supported,
		Installed:     input.Installed,
		PlistStatus:   input.PlistStatus,
		PlistPath:     shortener.Path(input.PlistPath),
		LogPath:       shortener.Path(input.LogPath),
		Running:       input.Running,
		Address:       input.Address,
		PID:           input.PID,
		Version:       input.Version,
		BinaryMatches: input.BinaryMatches,
		Error:         input.Error,
	}
	if input.Demo {
		result.PlistPath = "demo"
		result.LogPath = "demo"
	}
	if result.PlistStatus == "" {
		result.PlistStatus = string(DebugPlistStatusUnknown)
	}
	return result
}

// DebugPlistStatusUnknown は plist を読めなかったことを表す。値は launchd アダプタと
// 共有する公開契約で、`current` と `stale` も同じ語彙を使う。
const DebugPlistStatusUnknown = "unknown"

// DebugPlistStatusStale は plist が現在の PRX と一致しないことを表す。
const DebugPlistStatusStale = "stale"

// DebugReport は診断レポート全体で、検出した問題を先頭に置く。
type DebugReport struct {
	Problems   []DebugProblem  `json:"problems"`
	Build      DebugBuild      `json:"build"`
	Runtime    DebugRuntime    `json:"runtime"`
	Daemon     DebugDaemon     `json:"daemon"`
	Paths      DebugPaths      `json:"paths"`
	Config     DebugConfig     `json:"config"`
	Storage    DebugStorage    `json:"storage"`
	Records    DebugData       `json:"records"`
	GitHubSync DebugGitHubSync `json:"github_sync"`
}

// NewDebugBuild は実行中のビルドを表す値を作る。domain はバージョンを埋め込む
// パッケージに依存しないため、バージョンは引数で受け取る。
func NewDebugBuild(version string) DebugBuild {
	return DebugBuild{
		Version:     version,
		Development: strings.HasSuffix(version, "-dev"),
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
	}
}

// NewDebugRuntime はレポートの生成時刻と、server の場合は待ち受けを開始してからの
// 経過時間を記録する。
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

// NewDebugPaths は報告対象のパスを解決する。demo 実行では、渡された一時ディレクトリでは
// なく demo という語をそのまま報告する。
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

// DebugEnvironmentVariables は PRX に関係する環境変数のうち設定済みのものを報告する。
// 環境の状態を説明する場所は診断レポートだけとする。
func DebugEnvironmentVariables() []DebugEnvironmentVariable {
	result := make([]DebugEnvironmentVariable, 0, len(DebugEnvironmentNames))
	for _, name := range DebugEnvironmentNames {
		result = append(result, DebugEnvironmentVariable{Name: name, Set: os.Getenv(name) != ""})
	}
	return result
}

// NewDebugConfig は設定セクションを導出する。server も、データベースを開けなかった
// CLI 実行も、この 1 か所の導出を通る。
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
		Prompts:                 input.Prompts,
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

// NewDebugStorage は persistence が報告した内容からストレージセクションを導出する。
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

// NewDebugData は保存済みレコードを数える。件数はアプリケーションの他の部分が読むのと同じ
// snapshot に由来するため、報告される件数は呼び出し側が一覧できる内容と常に一致する。
func NewDebugData(snapshot Snapshot) DebugData {
	projects := newDebugTally()
	features := newDebugTally()
	taskStates := newDebugTally()
	pullRequestStates := newDebugTally()
	pullRequestHosts := newDebugTally()
	documentKinds := newDebugTally()
	// archived な project は配下の feature をすべて読み取り専用にするため、書き込みが
	// 拒否された理由を説明するには archived の件数が要る。
	for _, project := range snapshot.Projects {
		projects.add(debugProjectState(project.Archived))
	}
	for _, feature := range snapshot.Features {
		features.add(string(feature.DisplayStatus))
	}
	for _, task := range snapshot.Tasks {
		taskStates.add(string(task.DisplayState))
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
		PullRequestDisplayStates: pullRequestStates.counts(),
		PullRequestHosts:         pullRequestHosts.counts(),
		DocumentKinds:            documentKinds.counts(),
	}
}

// NewDebugGitHubSync は同期セクションを導出する。壊れた repository 1 つと壊れた環境全体を
// 読み手が区別できるよう、件数を制限した失敗の内訳も含める。
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

// DetectDebugProblems は読み手が最初に見るべき点を報告する。各問題は検出の元になった値を
// 持つため、レポート全体を読み直さずに確認できる。
func DetectDebugProblems(report DebugReport, now time.Time) []DebugProblem {
	problems := make([]DebugProblem, 0)
	problems = append(problems, detectDebugStorageProblems(report)...)
	problems = append(problems, detectDebugConfigProblems(report)...)
	problems = append(problems, detectDebugSyncProblems(report, now)...)
	problems = append(problems, detectDebugDaemonProblems(report)...)
	return problems
}

// detectDebugDaemonProblems は導入済みの LaunchAgent についてだけ報告する。未導入は
// 問題ではない。`prx serve` を手で使う運用も正当である。
func detectDebugDaemonProblems(report DebugReport) []DebugProblem {
	problems := make([]DebugProblem, 0, 3)
	daemon := report.Daemon
	if !daemon.Supported || !daemon.Installed {
		return problems
	}
	if daemon.PlistStatus == DebugPlistStatusStale {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeDaemonPlistStale,
			Target:      daemon.PlistPath,
			Evidence:    "the LaunchAgent was written by a different PRX binary or path",
			NextCommand: "prx daemon install",
		})
	}
	if !daemon.Running {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeDaemonNotRunning,
			Target:      daemon.PlistPath,
			Evidence:    "no process holds the run state lock",
			NextCommand: "prx daemon start",
		})
		return problems
	}
	if !daemon.BinaryMatches {
		problems = append(problems, DebugProblem{
			Code:        DebugProblemCodeDaemonBinaryOutdated,
			Target:      daemon.Address,
			Evidence:    fmt.Sprintf("the server reports version %s", daemon.Version),
			NextCommand: "prx daemon restart",
		})
	}
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

// debugHostsWithoutCredentials は pull request を持つのに認証方式がない host を報告する。
// 方式の一覧を省略した場合は GitHub.com の互換既定が効くため、設定が空であること自体は
// 問題ではない。
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

// debugPermissionsArePrivate は設定ローダーと同じ規則に従い、group や other に
// アクセスを与えるモードをすべて拒否する。
func debugPermissionsArePrivate(mode string) bool {
	var parsed int64
	if _, err := fmt.Sscanf(mode, "%o", &parsed); err != nil {
		return true
	}
	return parsed&0o077 == 0
}

// DebugPathShortener はホームディレクトリを ~ に置き換え、アカウント名を明かさずに
// レポートを issue へ貼れるようにする。
type DebugPathShortener struct {
	home string
}

// NewDebugPathShortener はホームディレクトリを一度だけ読む。ホームがない環境では
// prefix を推測せず、すべてのパスをそのままにする。
func NewDebugPathShortener() DebugPathShortener {
	home, err := os.UserHomeDir()
	if err != nil {
		return DebugPathShortener{}
	}
	return DebugPathShortener{home: home}
}

// Path はパス 1 件を短縮する。完全一致か、ホームディレクトリの直後が区切り文字である
// ことを要求するため、/home/user2 のような別ディレクトリはそのまま残る。
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

// Text はメッセージ中に埋め込まれたパスを短縮する。ディレクトリを作成できなかった等の失敗は
// 本文中にパスを含むため、構造化フィールドだけを短縮しても漏れてしまう。
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

// continuesPathSegment は、ホームディレクトリに一致した直後の文字が /home/user2 のように
// 名前を続けているかを返す。それは別のディレクトリなので、名前を保たなければならない。
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

// truncateDebugMessage はメッセージ 1 件の長さを制限し、極端に長いエラーがレポートを
// 占有しないようにする。
func truncateDebugMessage(value string) string {
	runes := []rune(value)
	if len(runes) <= DebugMaxErrorMessageRune {
		return value
	}
	return string(runes[:DebugMaxErrorMessageRune]) + "…"
}

// debugTally は値を集計し、map の反復順が出力に漏れないようにする。
type debugTally struct {
	counted map[string]int
}

func newDebugTally() *debugTally {
	return &debugTally{counted: map[string]int{}}
}

func (t *debugTally) add(name string) { t.counted[name]++ }

// debugProjectState は project が持つ唯一の状態を返す。project は feature や task が
// 共有する 2 層 status の規則の外にあるため。
func debugProjectState(archived bool) string {
	if archived {
		return "archived"
	}
	return "active"
}

// counts は件数の降順、次に名前順で並べる。件数が同じでも順序は 1 つに定まる。
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

// debugErrorNoise は同期エラーのうち pull request ごとに異なる部分に一致する。これを
// 取り除かないと全メッセージが一意になり、グループ化しても形ごとではなく失敗ごとに
// 代表が 1 件出てしまう。
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

// result は件数の降順、次にメッセージ順で並べる。これにより上限で切り落とすグループが
// 実行ごとに変わらない。
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
