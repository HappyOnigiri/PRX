package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func reportTime() time.Time {
	return time.Date(2026, 9, 3, 4, 5, 6, 0, time.UTC)
}

func timePointer(value time.Time) *time.Time { return &value }

func TestSortedDebugProblemCodesCoverEverySummary(t *testing.T) {
	codes := SortedDebugProblemCodes()
	if len(codes) != len(debugProblemSummaries) {
		t.Fatalf("codes=%d summaries=%d", len(codes), len(debugProblemSummaries))
	}
	for index, code := range codes {
		if debugProblemSummaries[code] == "" {
			t.Errorf("problem %q has no summary", code)
		}
		if index > 0 && codes[index-1] >= code {
			t.Errorf("codes are not sorted at %d: %q then %q", index, codes[index-1], code)
		}
	}
}

func TestNewDebugBuildMarksDevelopmentBuilds(t *testing.T) {
	development := NewDebugBuild("1.2.3-dev")
	if !development.Development || development.Version != "1.2.3-dev" {
		t.Fatalf("build=%+v", development)
	}
	if development.GoVersion == "" || development.OS == "" || development.Arch == "" {
		t.Fatalf("build omitted toolchain facts: %+v", development)
	}
	if NewDebugBuild("1.2.3").Development {
		t.Fatal("a stamped release must not be reported as a development build")
	}
}

func TestNewDebugRuntimeReportsServerUptime(t *testing.T) {
	started := reportTime().Add(-90 * time.Second)
	value := NewDebugRuntime(DebugRuntimeInput{
		Mode:          "serve",
		Demo:          true,
		GitHubFixture: true,
		ListenAddress: "127.0.0.1:7331",
		StartedAt:     &started,
	}, reportTime())
	if value.UptimeSeconds != 90 || value.ListenAddress != "127.0.0.1:7331" {
		t.Fatalf("runtime=%+v", value)
	}
	if !value.GeneratedAt.Equal(reportTime()) || value.TimeZone == "" {
		t.Fatalf("runtime=%+v", value)
	}
	cli := NewDebugRuntime(DebugRuntimeInput{Mode: "cli"}, reportTime())
	if cli.StartedAt != nil || cli.UptimeSeconds != 0 {
		t.Fatalf("a CLI run has no uptime: %+v", cli)
	}
}

func TestNewDebugPathsReportsFilesAndEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PRX_DB", "somewhere.db")
	t.Setenv("GITHUB_TOKEN", "")
	databasePath := filepath.Join(home, "prx", "prx.db")
	configPath := filepath.Join(home, "prx", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(databasePath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := NewDebugPaths(DebugPathsInput{
		DatabasePath:       databasePath,
		DatabasePathSource: "env",
		ConfigPath:         configPath,
		ConfigPathSource:   "default",
	})
	if paths.DatabasePath != "~/prx/prx.db" || paths.ConfigPath != "~/prx/config.yaml" {
		t.Fatalf("paths were not shortened: %+v", paths)
	}
	if !paths.DatabaseFileExists || !paths.ConfigFileExists || paths.ConfigPermissions != "0644" {
		t.Fatalf("paths=%+v", paths)
	}
	variables := map[string]bool{}
	for _, variable := range paths.EnvironmentVariables {
		variables[variable.Name] = variable.Set
	}
	if len(variables) != len(DebugEnvironmentNames) || !variables["PRX_DB"] || variables["GITHUB_TOKEN"] {
		t.Fatalf("environment=%+v", paths.EnvironmentVariables)
	}
}

func TestNewDebugPathsReportsDemoInsteadOfTemporaryLocations(t *testing.T) {
	paths := NewDebugPaths(DebugPathsInput{
		DatabasePath:       "/var/folders/T/prx-demo-123/prx.db",
		DatabasePathSource: "demo",
		ConfigPath:         "/var/folders/T/prx-demo-123/config.yaml",
		ConfigPathSource:   "demo",
		Demo:               true,
	})
	if paths.DatabasePath != "demo" || paths.ConfigPath != "demo" {
		t.Fatalf("a demo run disclosed its temporary locations: %+v", paths)
	}
	if paths.DatabaseFileExists || paths.ConfigFileExists {
		t.Fatalf("paths=%+v", paths)
	}
}

func TestDebugPathShortenerOnlyShortensTheHomeDirectory(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	shortener := NewDebugPathShortener()
	for _, test := range []struct{ value, want string }{
		{value: "/home/user", want: "~"},
		{value: "/home/user/prx/prx.db", want: "~/prx/prx.db"},
		{value: "/home/user2/prx.db", want: "/home/user2/prx.db"},
		{value: "/etc/prx.db", want: "/etc/prx.db"},
		{value: "", want: ""},
	} {
		if got := shortener.Path(test.value); got != test.want {
			t.Errorf("Path(%q)=%q, want %q", test.value, got, test.want)
		}
	}
	message := "create database directory: mkdir /home/user/prx: permission denied"
	want := "create database directory: mkdir ~/prx: permission denied"
	if got := shortener.Text(message); got != want {
		t.Errorf("Text=%q, want %q", got, want)
	}
	if got := shortener.Text("config file \"/home/user\" is unreadable"); !strings.Contains(got, "\"~\"") {
		t.Errorf("Text did not shorten a trailing home directory: %q", got)
	}
	if got := (DebugPathShortener{}).Text("/home/user/x"); got != "/home/user/x" {
		t.Errorf("an unknown home directory must leave text alone: %q", got)
	}
}

func TestNewDebugConfigReportsLoadFailureAsAnError(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	value := NewDebugConfig(DebugConfigInput{
		LoadError: "config file \"/home/user/prx/config.yaml\" must have permissions 0600",
		Warnings:  []string{"unknown field \"extra\" on line 3 is ignored"},
	})
	if value.Valid || len(value.Errors) != 1 {
		t.Fatalf("config=%+v", value)
	}
	if !strings.Contains(value.Errors[0], "~/prx/config.yaml") {
		t.Fatalf("the load failure disclosed the home directory: %q", value.Errors[0])
	}
	if value.Hosts == nil || value.AuthMethods == nil {
		t.Fatalf("empty collections must not be nil: %+v", value)
	}

	loaded := NewDebugConfig(DebugConfigInput{
		Version:                 1,
		AutoSyncIntervalSeconds: 3600,
		Hosts:                   []DebugConfigHost{{Host: "github.com"}},
		AuthMethods:             []DebugConfigAuthMethod{{ID: "work", Host: "github.com"}},
	})
	if !loaded.Valid || len(loaded.Errors) != 0 {
		t.Fatalf("config=%+v", loaded)
	}
}

func TestNewDebugStorageDerivesIntegrityAndTruncatesLongErrors(t *testing.T) {
	valid := NewDebugStorage(DebugStorageInput{AppliedSchemaVersion: 8, EmbeddedSchemaVersion: 8})
	if !valid.IntegrityValid || valid.CLISchemaVersion != CLIResponseSchemaVersion {
		t.Fatalf("storage=%+v", valid)
	}
	invalid := NewDebugStorage(DebugStorageInput{
		IntegrityErrors: []string{strings.Repeat("e", DebugMaxErrorMessageRune+50)},
	})
	if invalid.IntegrityValid {
		t.Fatal("integrity errors must make the section invalid")
	}
	if got := []rune(invalid.IntegrityErrors[0]); len(got) != DebugMaxErrorMessageRune+1 {
		t.Fatalf("error was not truncated: %d runes", len(got))
	}
	failed := NewDebugStorage(DebugStorageInput{Error: "open sqlite: unable to open database file"})
	if failed.IntegrityValid {
		t.Fatal("an unreadable database cannot report valid integrity")
	}
}

func TestNewDebugDataCountsStoredRecords(t *testing.T) {
	data := NewDebugData(Snapshot{
		Features: []Feature{
			{DisplayStatus: FeatureStatusActive},
			{DisplayStatus: FeatureStatusCompleted},
			{DisplayStatus: FeatureStatusActive},
		},
		Tasks: []Task{
			{Kind: TaskKindPR, DisplayState: TaskDisplayStateOpen},
			{Kind: TaskKindManual, DisplayState: TaskDisplayStateNotStarted},
		},
		Dependencies: []Dependency{{}},
		PullRequests: []PullRequest{
			{Host: "github.com", DisplayState: PullRequestDisplayStateOpen},
			{Host: "ghe.example.com", DisplayState: PullRequestDisplayStateMerged},
		},
		Documents: []Document{{Kind: DocumentKindURL}},
	})
	if data.Features != 3 || data.Tasks != 2 || data.Dependencies != 1 || data.PullRequests != 2 ||
		data.Documents != 1 {
		t.Fatalf("data=%+v", data)
	}
	if len(data.FeatureStatuses) != 2 || data.FeatureStatuses[0] != (DebugCount{Name: "active", Count: 2}) {
		t.Fatalf("feature statuses=%+v", data.FeatureStatuses)
	}
	// Equal counts fall back to the name so the order never depends on the map.
	if data.PullRequestHosts[0].Name != "ghe.example.com" || data.PullRequestHosts[1].Name != "github.com" {
		t.Fatalf("hosts=%+v", data.PullRequestHosts)
	}
}

func TestNewDebugGitHubSyncGroupsFailuresAndReportsSchedule(t *testing.T) {
	attempt := reportTime().Add(-2 * time.Hour)
	completed := reportTime().Add(-3 * time.Hour)
	value := NewDebugGitHubSync(DebugGitHubSyncInput{
		Status: GitHubSyncStatus{
			IntervalSeconds: 3600,
			LastAttemptAt:   &attempt,
			LastUpdatedAt:   &completed,
		},
		PullRequests: []PullRequest{
			{
				TaskID:     "T-1",
				Host:       "github.com",
				Owner:      "acme",
				Repository: "web",
				SyncError:  "pull request acme/web#12 was not found",
				Stale:      true,
			},
			{
				TaskID:     "T-2",
				Host:       "github.com",
				Owner:      "acme",
				Repository: "web",
				SyncError:  "pull request acme/web#34 was not found",
			},
			{
				TaskID:     "T-3",
				Host:       "github.com",
				Owner:      "acme",
				Repository: "api",
				SyncError:  "credentials were rejected",
			},
			{TaskID: "T-4", Host: "github.com", Owner: "acme", Repository: "api"},
		},
		AuthCache: []DebugAuthCacheEntry{{Host: "github.com", Owner: "acme", Repository: "web", AuthMethodID: "work"}},
	}, reportTime())

	if value.FailedPullRequests != 3 || value.StalePullRequests != 1 {
		t.Fatalf("sync=%+v", value)
	}
	if !value.Due || value.NextRunAt == nil || !value.NextRunAt.Equal(attempt.Add(time.Hour)) {
		t.Fatalf("schedule=%+v", value)
	}
	if value.SecondsSinceLastUpdate != 3*60*60 {
		t.Fatalf("seconds since last update=%d", value.SecondsSinceLastUpdate)
	}
	if len(value.HostFailures) != 1 || value.HostFailures[0].Count != 3 {
		t.Fatalf("host failures=%+v", value.HostFailures)
	}
	if len(value.RepositoryFailures) != 2 || value.RepositoryFailures[0].Scope != "github.com/acme/web" {
		t.Fatalf("repository failures=%+v", value.RepositoryFailures)
	}
	// The two "not found" messages differ only in the pull request number, so
	// they must be reported as one group of two rather than two of one.
	if len(value.ErrorGroups) != 2 || value.ErrorGroups[0].Count != 2 {
		t.Fatalf("error groups=%+v", value.ErrorGroups)
	}
	if len(value.ErrorGroups[0].TaskIDs) != 2 || value.ErrorGroups[0].TotalTaskCount != 2 {
		t.Fatalf("group tasks=%+v", value.ErrorGroups[0])
	}
	if len(value.AuthCache) != 1 || value.OmittedAuthCacheEntries != 0 {
		t.Fatalf("auth cache=%+v", value.AuthCache)
	}
}

func TestNewDebugGitHubSyncAppliesDeterministicLimits(t *testing.T) {
	var pullRequests []PullRequest
	for index := range DebugMaxErrorGroups + 3 {
		for repetition := range index + 1 {
			pullRequests = append(pullRequests, PullRequest{
				TaskID:     fmt.Sprintf("T-%d-%d", index, repetition),
				Host:       "github.com",
				Owner:      "acme",
				Repository: fmt.Sprintf("repo-%02d", index),
				SyncError:  fmt.Sprintf("failure kind %s reached", string(rune('a'+index))),
			})
		}
	}
	var cache []DebugAuthCacheEntry
	for index := range DebugMaxAuthCacheRows + 2 {
		cache = append(
			cache,
			DebugAuthCacheEntry{Host: "github.com", Owner: "acme", Repository: fmt.Sprintf("r%03d", index)},
		)
	}

	first := NewDebugGitHubSync(DebugGitHubSyncInput{PullRequests: pullRequests, AuthCache: cache}, reportTime())
	second := NewDebugGitHubSync(DebugGitHubSyncInput{PullRequests: pullRequests, AuthCache: cache}, reportTime())
	if FormatDebugReport(DebugReport{GitHubSync: first}) != FormatDebugReport(DebugReport{GitHubSync: second}) {
		t.Fatal("two reports over the same data differ")
	}
	if len(first.ErrorGroups) != DebugMaxErrorGroups || first.OmittedErrorGroups != 3 {
		t.Fatalf("error groups=%d omitted=%d", len(first.ErrorGroups), first.OmittedErrorGroups)
	}
	// The most frequent group comes first, and each group lists a bounded sample.
	if first.ErrorGroups[0].Count != DebugMaxErrorGroups+3 {
		t.Fatalf("groups are not ordered by count: %+v", first.ErrorGroups[0])
	}
	if len(first.ErrorGroups[0].TaskIDs) != DebugMaxTasksPerGroup ||
		first.ErrorGroups[0].TotalTaskCount != DebugMaxErrorGroups+3 {
		t.Fatalf("group sample=%+v", first.ErrorGroups[0])
	}
	if len(first.AuthCache) != DebugMaxAuthCacheRows || first.OmittedAuthCacheEntries != 2 {
		t.Fatalf("auth cache=%d omitted=%d", len(first.AuthCache), first.OmittedAuthCacheEntries)
	}
}

func TestNewDebugGitHubSyncTreatsAnUnattemptedRunAsDue(t *testing.T) {
	value := NewDebugGitHubSync(DebugGitHubSyncInput{Status: GitHubSyncStatus{IntervalSeconds: 3600}}, reportTime())
	if !value.Due || value.NextRunAt != nil {
		t.Fatalf("sync=%+v", value)
	}
}

func TestDetectDebugProblems(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	for _, test := range []struct {
		name   string
		report DebugReport
		want   DebugProblemCode
		next   string
	}{
		{
			name: "storage unavailable",
			report: DebugReport{
				Paths:   DebugPaths{DatabasePath: "~/prx.db"},
				Config:  DebugConfig{Valid: true},
				Storage: DebugStorage{Error: "open sqlite: unable to open database file"},
			},
			want: DebugProblemCodeStorageUnavailable,
			next: "prx debug --json",
		},
		{
			name: "schema ahead of binary",
			report: DebugReport{
				Config:  DebugConfig{Valid: true},
				Storage: DebugStorage{AppliedSchemaVersion: 9, EmbeddedSchemaVersion: 8},
			},
			want: DebugProblemCodeSchemaVersionAheadOfBinary,
		},
		{
			name: "database not writable",
			report: DebugReport{
				Paths:  DebugPaths{DatabasePath: "~/prx.db"},
				Config: DebugConfig{Valid: true},
				Storage: DebugStorage{
					DatabaseFile: DebugDatabaseFile{Applicable: true, WriteError: "permission denied"},
				},
			},
			want: DebugProblemCodeDatabaseNotWritable,
			next: "ls -l ~/prx.db",
		},
		{
			name: "integrity errors",
			report: DebugReport{
				Config:  DebugConfig{Valid: true},
				Storage: DebugStorage{IntegrityErrors: []string{"task T-1 references a missing feature"}},
			},
			want: DebugProblemCodeDatabaseIntegrityErrors,
			next: "prx validate",
		},
		{
			name:   "config unreadable",
			report: DebugReport{Config: DebugConfig{Errors: []string{"decode config: bad yaml"}}},
			want:   DebugProblemCodeConfigUnreadable,
			next:   "prx config validate",
		},
		{
			name: "config permissions too open",
			report: DebugReport{
				Paths:  DebugPaths{ConfigPath: "~/prx/config.yaml", ConfigPermissions: "0644"},
				Config: DebugConfig{Valid: true},
			},
			want: DebugProblemCodeConfigPermissionsTooOpen,
			next: "chmod 600 ~/prx/config.yaml",
		},
		{
			name: "config unknown fields",
			report: DebugReport{
				Config: DebugConfig{Valid: true, Warnings: []string{"unknown field \"extra\""}},
			},
			want: DebugProblemCodeConfigUnknownFields,
		},
		{
			name: "host without credentials",
			report: DebugReport{
				Config: DebugConfig{
					Valid:       true,
					AuthMethods: []DebugConfigAuthMethod{{ID: "work", Host: "github.com"}},
				},
				Records: DebugData{
					PullRequests:     1,
					PullRequestHosts: []DebugCount{{Name: "ghe.example.com", Count: 1}},
				},
				GitHubSync: DebugGitHubSync{Status: GitHubSyncStatus{LastUpdatedAt: timePointer(reportTime())}},
			},
			want: DebugProblemCodeNoAuthMethodForHost,
			next: "prx config auth",
		},
		{
			name: "run error",
			report: DebugReport{
				Config:     DebugConfig{Valid: true},
				GitHubSync: DebugGitHubSync{Status: GitHubSyncStatus{Error: "credentials were rejected"}},
			},
			want: DebugProblemCodeGitHubSyncRunError,
			next: "prx sync status --json",
		},
		{
			name: "never completed",
			report: DebugReport{
				Config:  DebugConfig{Valid: true},
				Records: DebugData{PullRequests: 2},
			},
			want: DebugProblemCodeGitHubSyncNeverCompleted,
			next: "prx sync",
		},
		{
			name: "overdue",
			report: DebugReport{
				Config:  DebugConfig{Valid: true},
				Records: DebugData{PullRequests: 2},
				GitHubSync: DebugGitHubSync{
					Status: GitHubSyncStatus{
						IntervalSeconds: 3600,
						LastUpdatedAt:   timePointer(reportTime().Add(-24 * time.Hour)),
					},
					SecondsSinceLastUpdate: 24 * 60 * 60,
				},
			},
			want: DebugProblemCodeGitHubSyncOverdue,
			next: "prx sync",
		},
		{
			name: "stale pull requests",
			report: DebugReport{
				Config:  DebugConfig{Valid: true},
				Records: DebugData{PullRequests: 4},
				GitHubSync: DebugGitHubSync{
					StalePullRequests: 2,
					Status:            GitHubSyncStatus{LastUpdatedAt: timePointer(reportTime())},
				},
			},
			want: DebugProblemCodePullRequestsStale,
			next: "prx stale",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			problems := DetectDebugProblems(test.report, reportTime())
			var found *DebugProblem
			for index, problem := range problems {
				if problem.Code == test.want {
					found = &problems[index]
				}
			}
			if found == nil {
				t.Fatalf("problem %q was not detected: %+v", test.want, problems)
			}
			if found.Evidence == "" {
				t.Errorf("problem %q carries no evidence", test.want)
			}
			if test.next != "" && found.NextCommand != test.next {
				t.Errorf("next command=%q, want %q", found.NextCommand, test.next)
			}
		})
	}
}

func TestDetectDebugProblemsReportsNothingForAHealthyInstallation(t *testing.T) {
	report := DebugReport{
		Paths:  DebugPaths{ConfigPermissions: "0600"},
		Config: DebugConfig{Valid: true, AuthMethods: []DebugConfigAuthMethod{{ID: "work", Host: "github.com"}}},
		Storage: DebugStorage{
			AppliedSchemaVersion:  8,
			EmbeddedSchemaVersion: 8,
			IntegrityValid:        true,
			DatabaseFile:          DebugDatabaseFile{Applicable: true, Writable: true},
		},
		Records: DebugData{
			PullRequests:     2,
			PullRequestHosts: []DebugCount{{Name: "github.com", Count: 2}},
		},
		GitHubSync: DebugGitHubSync{Status: GitHubSyncStatus{
			IntervalSeconds: 3600,
			LastUpdatedAt:   timePointer(reportTime().Add(-30 * time.Minute)),
		}},
	}
	if problems := DetectDebugProblems(report, reportTime()); len(problems) != 0 {
		t.Fatalf("problems=%+v", problems)
	}
}

// A storage failure hides the values every other storage check reads, so it is
// reported alone instead of with the derived findings it would invalidate.
func TestDetectDebugProblemsReportsStorageFailureAlone(t *testing.T) {
	report := DebugReport{
		Config: DebugConfig{Valid: true},
		Storage: DebugStorage{
			Error:                 "open sqlite: unable to open database file",
			AppliedSchemaVersion:  9,
			EmbeddedSchemaVersion: 8,
			IntegrityErrors:       []string{"unreadable"},
		},
	}
	problems := DetectDebugProblems(report, reportTime())
	if len(problems) != 1 || problems[0].Code != DebugProblemCodeStorageUnavailable {
		t.Fatalf("problems=%+v", problems)
	}
}

func TestFormatDebugReportRendersEverySection(t *testing.T) {
	report := DebugReport{
		Problems: []DebugProblem{{
			Code:        DebugProblemCodeDatabaseIntegrityErrors,
			Target:      "~/prx/prx.db",
			Evidence:    "1 integrity errors",
			NextCommand: "prx validate",
		}},
		Build: DebugBuild{Version: "1.2.3-dev", Development: true, GoVersion: "go1.25.1", OS: "darwin", Arch: "arm64"},
		Runtime: DebugRuntime{
			Mode:          "serve",
			GeneratedAt:   reportTime(),
			TimeZone:      "UTC",
			ListenAddress: "127.0.0.1:7331",
			StartedAt:     timePointer(reportTime().Add(-time.Minute)),
			UptimeSeconds: 60,
		},
		Paths: DebugPaths{
			DatabasePath:       "~/prx/prx.db",
			DatabasePathSource: "default",
			DatabaseFileExists: true,
			ConfigPath:         "~/prx/config.yaml",
			ConfigPathSource:   "default",
			ConfigFileExists:   true,
			ConfigPermissions:  "0600",
			EnvironmentVariables: []DebugEnvironmentVariable{
				{Name: "PRX_DB", Set: false},
				{Name: "GITHUB_TOKEN", Set: true},
			},
		},
		Config: DebugConfig{
			Version:  1,
			Valid:    true,
			Errors:   []string{},
			Warnings: []string{},
			Hosts: []DebugConfigHost{
				{Host: "github.com", APIURL: "https://api.github.com/", GraphQLURL: "https://api.github.com/graphql"},
			},
			AuthMethods: []DebugConfigAuthMethod{
				{ID: "work", Host: "github.com", Type: "keychain", SecretConfigured: true},
			},
			AutoSyncIntervalSeconds: 3600,
		},
		Storage: DebugStorage{
			AppliedSchemaVersion:  8,
			EmbeddedSchemaVersion: 8,
			IntegrityValid:        false,
			IntegrityErrors:       []string{"task T-1 references a missing feature"},
			DatabaseFile: DebugDatabaseFile{
				Applicable:   true,
				SizeBytes:    24576,
				WALPresent:   true,
				WALSizeBytes: 32768,
				Writable:     true,
			},
			CLISchemaVersion: "2",
		},
		Records: DebugData{
			Features:        1,
			Tasks:           1,
			PullRequests:    1,
			FeatureStatuses: []DebugCount{{Name: "active", Count: 1}},
		},
		GitHubSync: DebugGitHubSync{
			Status: GitHubSyncStatus{
				IntervalSeconds: 3600,
				LastAttemptAt:   timePointer(reportTime().Add(-time.Hour)),
				Succeeded:       1,
			},
			Due:                true,
			HostFailures:       []DebugSyncFailure{{Scope: "github.com", Count: 1}},
			RepositoryFailures: []DebugSyncFailure{{Scope: "github.com/acme/web", Count: 1}},
			FailedPullRequests: 1,
			ErrorGroups: []DebugErrorGroup{
				{Message: "pull request was not found", Count: 1, TaskIDs: []string{"T-1"}, TotalTaskCount: 1},
			},
			AuthCache: []DebugAuthCacheEntry{
				{
					Host:            "github.com",
					Owner:           "acme",
					Repository:      "web",
					AuthMethodID:    "work",
					LastSucceededAt: reportTime(),
				},
			},
		},
	}

	want := `PRX diagnostic report

problems:
  - database_integrity_errors: stored dependency data failed validation
    target: ~/prx/prx.db
    evidence: 1 integrity errors
    next: prx validate

build:
  version: 1.2.3-dev
  development: yes
  go: go1.25.1
  platform: darwin/arm64

runtime:
  mode: serve
  demo: no
  github_fixture: no
  generated_at: 2026-09-03T04:05:06Z
  time_zone: UTC
  listen_address: 127.0.0.1:7331
  started_at: 2026-09-03T04:04:06Z
  uptime_seconds: 60

paths:
  database_path: ~/prx/prx.db
  database_path_source: default
  database_file_exists: yes
  config_path: ~/prx/config.yaml
  config_path_source: default
  config_file_exists: yes
  config_permissions: 0600
  environment:
    PRX_DB: unset
    GITHUB_TOKEN: set

config:
  version: 1
  valid: yes
  auto_sync_interval_seconds: 3600
  errors: none
  warnings: none
  hosts:
    - github.com
      api_url: https://api.github.com/
      graphql_url: https://api.github.com/graphql
  auth_methods:
    - work
      host: github.com
      type: keychain
      secret_configured: yes

storage:
  applied_schema_version: 8
  embedded_schema_version: 8
  cli_schema_version: 2
  integrity_valid: no
  integrity_errors:
    - task T-1 references a missing feature
  database_file:
    size_bytes: 24576
    wal_present: yes
    wal_size_bytes: 32768
    shm_present: no
    writable: yes

records:
  features: 1
  tasks: 1
  dependencies: 0
  pull_requests: 1
  documents: 0
  feature_statuses:
    active: 1
  task_display_states: none
  task_kinds: none
  pull_request_display_states: none
  pull_request_hosts: none
  document_kinds: none

github_sync:
  interval_seconds: 3600
  last_attempt_at: 2026-09-03T03:05:06Z
  last_updated_at: never
  seconds_since_last_update: 0
  next_run_at: never
  due: yes
  succeeded: 1
  failed: 0
  run_error: none
  stale_pull_requests: 0
  failed_pull_requests: 1
  host_failures:
    github.com: 1
  repository_failures:
    github.com/acme/web: 1
  error_groups:
    - pull request was not found
      count: 1
      task_ids: T-1
      total_task_count: 1
  auth_cache:
    github.com/acme/web: work at 2026-09-03T04:05:06Z
`
	if got := FormatDebugReport(report); got != want {
		t.Fatalf("report text mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestFormatDebugReportReportsAbsentSections(t *testing.T) {
	text := FormatDebugReport(DebugReport{
		Storage: DebugStorage{Error: "open sqlite: unable to open database file"},
		Records: DebugData{Error: "storage is unavailable"},
		GitHubSync: DebugGitHubSync{
			Error:                     "storage is unavailable",
			OmittedRepositoryFailures: 4,
			RepositoryFailures:        []DebugSyncFailure{{Scope: "github.com/acme/web", Count: 1}},
		},
	})
	for _, want := range []string{
		"problems:\n  detected: none",
		"database_file: not applicable",
		"storage:\n  error: open sqlite: unable to open database file",
		"records:\n  error: storage is unavailable",
		"repository_failures: 1 shown, 4 omitted",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("report omitted %q:\n%s", want, text)
		}
	}
}
