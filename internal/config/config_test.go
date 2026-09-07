package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/prompt"
)

func TestNormalizeDefaultsAndRejectsUnsafeValues(t *testing.T) {
	value, err := (Config{
		Version: CurrentVersion,
		GitHub:  GitHubConfig{Hosts: []Host{{Host: "GHE.Example.com:8443"}}},
	}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	host := value.GitHub.Hosts[0]
	if host.Host != "ghe.example.com:8443" || host.WebURL != "https://ghe.example.com:8443" ||
		host.APIURL != "https://ghe.example.com:8443/api/v3/" ||
		host.UploadURL != "https://ghe.example.com:8443/api/uploads/" ||
		host.GraphQLURL != "https://ghe.example.com:8443/api/graphql" ||
		value.GitHub.AutoSyncIntervalSeconds != DefaultAutoSyncIntervalSeconds {
		t.Fatalf("normalized host=%+v", host)
	}

	for _, test := range []struct {
		name  string
		value Config
	}{
		{name: "unsupported version", value: Config{Version: 2}},
		{
			name: "duplicate host",
			value: Config{
				Version: 1,
				GitHub: GitHubConfig{Hosts: []Host{
					{Host: "ghe.example.com"},
					{Host: "GHE.EXAMPLE.COM"},
				}},
			},
		},
		{
			name: "http web URL",
			value: Config{
				Version: 1,
				GitHub: GitHubConfig{Hosts: []Host{{
					Host: "ghe.example.com", WebURL: "http://ghe.example.com",
				}}},
			},
		},
		{
			name: "cross-host enterprise API URL",
			value: Config{
				Version: 1,
				GitHub: GitHubConfig{Hosts: []Host{{
					Host: "ghe.example.com", APIURL: "https://api.example.com/api/v3/",
				}}},
			},
		},
		{
			name: "automatic sync interval below minimum",
			value: Config{
				Version: 1,
				GitHub:  GitHubConfig{AutoSyncIntervalSeconds: 599},
			},
		},
		{
			name: "unknown auth host",
			value: Config{
				Version: 1,
				GitHub: GitHubConfig{AuthMethods: []AuthMethod{{
					ID: "token", Host: "ghe.example.com", Type: AuthMethodTypeInline, Token: "secret",
				}}},
			},
		},
		{
			name: "token on environment",
			value: Config{
				Version: 1,
				GitHub: GitHubConfig{AuthMethods: []AuthMethod{{
					ID: "token", Host: "github.com", Type: AuthMethodTypeEnvironment,
					Variable: "TOKEN", Token: "secret",
				}}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.value.Validate(); ErrorCodeOf(err) != ErrorCodeInvalid {
				t.Fatalf("error=%v code=%s", err, ErrorCodeOf(err))
			}
		})
	}
	maximum := Default()
	if err := maximum.SetAutoSyncInterval(int64(^uint64(0) >> 1)); err != nil {
		t.Fatalf("maximum interval was rejected: %v", err)
	}
	// Normalize は省略された間隔を既定値として扱うため、0 は黙って 3600 に
	// なるのではなくここで拒否されなければならない。
	for _, seconds := range []int64{0, -1, MinimumAutoSyncIntervalSeconds - 1} {
		below := Default()
		if err := below.SetAutoSyncInterval(seconds); ErrorCodeOf(err) != ErrorCodeInvalid {
			t.Fatalf("SetAutoSyncInterval(%d) error=%v code=%s", seconds, err, ErrorCodeOf(err))
		}
	}

	for _, raw := range []string{
		"https://ghe.example.com",
		"ghe.example.com/path",
		"ghe.example.com:bad",
		"ghe.example.com:0",
		"ghe.example.com:65536",
	} {
		if _, err := NormalizeHost(raw); err == nil {
			t.Fatalf("NormalizeHost(%q) succeeded", raw)
		}
	}
}

func TestConfigStoreRoundTripMasksSecretsAndPreservesImplicitMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if value, err := store.Load(); err != nil || value.Version != CurrentVersion || len(value.GitHub.Hosts) != 1 ||
		value.GitHub.AutoSyncIntervalSeconds != DefaultAutoSyncIntervalSeconds {
		t.Fatalf("missing config value=%+v err=%v", value, err)
	}
	if err := store.Save(Default()); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "auth_methods") {
		t.Fatalf("implicit auth mode was serialized: %s", body)
	}

	explicitEmpty := Default()
	explicitEmpty.GitHub.AuthMethods = []AuthMethod{}
	if err := store.Save(explicitEmpty); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "auth_methods: []") {
		t.Fatalf("explicit empty auth mode was not serialized: %s", body)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GitHub.AuthMethods == nil || len(loaded.GitHub.AuthMethods) != 0 {
		t.Fatalf("explicit empty auth mode was lost: %#v", loaded.GitHub.AuthMethods)
	}

	ghe := Host{Host: "ghe.example.com"}
	value := Config{
		Version: CurrentVersion,
		GitHub: GitHubConfig{
			Hosts: []Host{DefaultHost(), ghe},
			AuthMethods: []AuthMethod{
				{ID: "inline", Host: "ghe.example.com", Type: AuthMethodTypeInline, Token: "github_pat_supersecret"},
				{ID: "environment", Host: "github.com", Type: AuthMethodTypeEnvironment, Variable: "GH_TOKEN"},
			},
		},
	}
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode=%o, want 600", info.Mode().Perm())
	}
	if directory, err := os.Stat(filepath.Dir(path)); err != nil || directory.Mode().Perm() != 0o700 {
		t.Fatalf("config directory mode=%v err=%v", directory.Mode().Perm(), err)
	}
	loaded, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GitHub.AuthMethods[0].Token != "github_pat_supersecret" {
		t.Fatalf("inline token did not round-trip: %+v", loaded.GitHub.AuthMethods[0])
	}
	public, err := store.Public()
	if err != nil {
		t.Fatal(err)
	}
	publicJSON, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(publicJSON), "github_pat_supersecret") ||
		public.GitHub.AuthMethods[0].SecretHint != "gith…cret" ||
		!public.GitHub.AuthMethods[0].SecretConfigured {
		t.Fatalf("public config exposed or lost secret metadata: %s / %+v", publicJSON, public.GitHub.AuthMethods[0])
	}
}

func TestConfigStoreWarnsAboutUnknownFieldsAndRejectsInsecureFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	// 新しい PRX が書いたファイルも読み込め続ける必要がある。このビルドが
	// 知らないフィールドは拒否せず報告する。
	newer := "version: 1\nunknown: true\ngithub:\n  hosts:\n" +
		"    - host: ghe.example.com\n      future_url: https://ghe.example.com/future\n"
	if err := os.WriteFile(path, []byte(newer), 0o600); err != nil {
		t.Fatal(err)
	}
	warnings, err := store.Validate()
	if err != nil {
		t.Fatalf("unknown fields failed the load: %v", err)
	}
	if len(warnings) != 2 || !strings.Contains(warnings[0], `"unknown"`) ||
		!strings.Contains(warnings[1], `"future_url"`) {
		t.Fatalf("unknown field warnings=%q", warnings)
	}
	loaded, warnings, err := store.LoadWithWarnings()
	if err != nil || len(warnings) != 2 {
		t.Fatalf("load warnings=%q err=%v", warnings, err)
	}
	if len(loaded.GitHub.Hosts) != 1 || loaded.GitHub.Hosts[0].Host != "ghe.example.com" {
		t.Fatalf("known fields were lost: %+v", loaded.GitHub.Hosts)
	}

	for _, test := range []struct {
		name string
		body string
	}{
		{name: "type error", body: "version: nope\n"},
		{name: "unknown field with a type error", body: "version: nope\nunknown: true\n"},
		{name: "duplicate key", body: "version: 1\nversion: 1\n"},
		{name: "syntax error", body: "version: 1\n  github:\n"},
		{name: "second document", body: "version: 1\n---\nversion: 1\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(test.body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(); ErrorCodeOf(err) != ErrorCodeInvalid {
				t.Fatalf("error=%v code=%s", err, ErrorCodeOf(err))
			}
		})
	}

	if err := os.WriteFile(path, []byte("version: 1\ngithub:\n  hosts: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); ErrorCodeOf(err) != ErrorCodeInvalid {
		t.Fatalf("insecure mode error=%v code=%s", err, ErrorCodeOf(err))
	}
}

// TestConfigStoreUpdateDropsUnknownFields は、古いビルドによる書き込みが表現
// できなかった内容を落とすことを記録する。読み込み時の警告が告げるとおり。
func TestConfigStoreUpdateDropsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("version: 1\nunknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(*Config) error { return nil }); err != nil {
		t.Fatal(err)
	}
	warnings, err := store.Validate()
	if err != nil || len(warnings) != 0 {
		t.Fatalf("warnings after write=%q err=%v", warnings, err)
	}
}

func TestConfigCRUDAndPathPrecedence(t *testing.T) {
	value := Default()
	host := Host{Host: "ghe.example.com"}
	if err := value.AddHost(host); err != nil {
		t.Fatal(err)
	}
	method := AuthMethod{ID: "first", Host: "ghe.example.com", Type: AuthMethodTypeInline, Token: "secret"}
	if err := value.AddAuthMethod(method); err != nil {
		t.Fatal(err)
	}
	updated := method
	updated.ID = "renamed"
	updated.Token = "new-secret"
	if err := value.UpdateAuthMethod("first", updated); err != nil {
		t.Fatal(err)
	}
	if err := value.AddAuthMethod(AuthMethod{ID: "second", Host: "github.com", Type: AuthMethodTypeGHCLI}); err != nil {
		t.Fatal(err)
	}
	if err := value.ReorderAuthMethods([]string{"second", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if value.GitHub.AuthMethods[0].ID != "second" {
		t.Fatalf("reordered methods=%+v", value.GitHub.AuthMethods)
	}
	if err := value.RemoveHost("ghe.example.com"); ErrorCodeOf(err) != ErrorCodeReferences {
		t.Fatalf("referenced host removal error=%v code=%s", err, ErrorCodeOf(err))
	}
	if err := value.UpdateHost("ghe.example.com", Host{Host: "ghe-renamed.example.com"}); err != nil {
		t.Fatal(err)
	}
	if value.GitHub.AuthMethods[1].Host != "ghe-renamed.example.com" {
		t.Fatalf("renamed host auth=%+v", value.GitHub.AuthMethods)
	}
	if err := value.RemoveAuthMethod("renamed"); err != nil {
		t.Fatal(err)
	}
	if err := value.RemoveHost("ghe-renamed.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := value.RemoveHost("github.com"); ErrorCodeOf(err) != ErrorCodeInvalid {
		t.Fatalf("default host removal error=%v code=%s", err, ErrorCodeOf(err))
	}
	if err := value.RemoveAuthMethod("missing"); ErrorCodeOf(err) != ErrorCodeNotFound {
		t.Fatalf("missing auth removal error=%v code=%s", err, ErrorCodeOf(err))
	}

	envPath := filepath.Join(t.TempDir(), "env.yaml")
	t.Setenv("PRX_CONFIG", envPath)
	fromEnv, err := NewStore("")
	if err != nil || fromEnv.Path() != envPath {
		t.Fatalf("env path=%q err=%v", fromEnv.Path(), err)
	}
	overridePath := filepath.Join(t.TempDir(), "override.yaml")
	fromOverride, err := NewStore(overridePath)
	if err != nil || fromOverride.Path() != overridePath {
		t.Fatalf("override path=%q err=%v", fromOverride.Path(), err)
	}
	if got := PathFromContext(WithPath(context.Background(), overridePath)); got != overridePath {
		t.Fatalf("context path=%q", got)
	}
}

func TestPromptTemplatesLoadDefaultAndSurviveAWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	// prompts がなかった頃のファイルも読み込め、CLI がプロンプトなしになる
	// のではなく組み込みテンプレートが穴を埋める。
	legacy := "version: 1\ngithub:\n  hosts:\n    - host: github.com\n"
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Prompts != prompt.DefaultTemplates() {
		t.Fatalf("prompts=%+v, want the built-in templates", loaded.Prompts)
	}

	custom := prompt.Templates{
		Design:         "Design {{task_id}}\nsecond line\n",
		Implementation: "Implement {{task_id}} of {{feature_id}}",
		Batch:          "Implement {{task_list}} of {{feature_id}}",
	}
	if _, err := store.Update(func(settings *Config) error { return settings.SetPrompts(custom) }); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Prompts != custom {
		t.Fatalf("prompts=%+v, want %+v", reloaded.Prompts, custom)
	}
	// 無関係な設定の書き込みが保存済みテンプレートを乱してはならない。
	if _, err := store.Update(func(settings *Config) error {
		return settings.SetAutoSyncInterval(MinimumAutoSyncIntervalSeconds)
	}); err != nil {
		t.Fatal(err)
	}
	afterWrite, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if afterWrite.Prompts != custom {
		t.Fatalf("prompts=%+v, want %+v", afterWrite.Prompts, custom)
	}
}

func TestDefaultPromptTemplatesStayOutOfTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	// prompts と無関係な書き込みが組み込みの文言をファイルに固定してはならない。
	// インストール先は以後のバージョンに追従し続ける。
	if _, err := store.Update(func(settings *Config) error {
		return settings.SetAutoSyncInterval(MinimumAutoSyncIntervalSeconds)
	}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "prompts:") {
		t.Fatalf("config file contains prompts:\n%s", body)
	}

	// カスタマイズされたテンプレートだけが保存され、もう一方は組み込みの
	// 文言に追従し続ける。
	custom := prompt.DefaultTemplates()
	custom.Design = "Design {{task_id}}\n"
	if _, err := store.Update(func(settings *Config) error { return settings.SetPrompts(custom) }); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Design {{task_id}}") {
		t.Fatalf("config file lost the customized design template:\n%s", body)
	}
	if strings.Contains(string(body), "implementation:") {
		t.Fatalf("config file stored the built-in implementation template:\n%s", body)
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Prompts != custom {
		t.Fatalf("prompts=%+v, want %+v", reloaded.Prompts, custom)
	}
}

func TestInvalidPromptTemplateFailsTheConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	settings := Default()
	err = settings.SetPrompts(prompt.Templates{Design: "no target", Implementation: "{{task_id}}"})
	if err == nil || !strings.Contains(err.Error(), "prompts.design") {
		t.Fatalf("error=%v, want a rejected design template", err)
	}
	if ErrorCodeOf(err) != ErrorCodeInvalid {
		t.Fatalf("code=%q, want %q", ErrorCodeOf(err), ErrorCodeInvalid)
	}
	// 拒否された組は保持されないため、呼び出し元は妥当な値を持ったままになる。
	if settings.Prompts != prompt.DefaultTemplates() {
		t.Fatalf("prompts=%+v, want the previous templates", settings.Prompts)
	}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}

	broken := "version: 1\nprompts:\n  design: \"{{plan_body}}\"\n  implementation: \"{{task_id}}\"\n"
	if err := os.WriteFile(path, []byte(broken), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil || !strings.Contains(err.Error(), "unsupported placeholder") {
		t.Fatalf("error=%v, want an unsupported placeholder failure", err)
	}
}

func TestDebugInputReportsWhetherPromptsWereEdited(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	// 手つかずのインストールに対するレポートが「誰かが文言を編集した」と
	// 読めてはならない。それがコピーしたプロンプトの不具合時に問われる点。
	input := store.DebugInput()
	if input.Prompts.Design.Customized || input.Prompts.Implementation.Customized {
		t.Fatalf("prompts=%+v, want neither reported as customized", input.Prompts)
	}
	if input.Prompts.Design.Bytes != len(prompt.DefaultTemplates().Design) {
		t.Fatalf("design bytes=%d", input.Prompts.Design.Bytes)
	}

	custom := prompt.DefaultTemplates()
	custom.Design = "Design {{task_id}}\n"
	if _, err := store.Update(func(settings *Config) error { return settings.SetPrompts(custom) }); err != nil {
		t.Fatal(err)
	}
	edited := store.DebugInput()
	if !edited.Prompts.Design.Customized || edited.Prompts.Implementation.Customized {
		t.Fatalf("prompts=%+v, want only the design template reported as customized", edited.Prompts)
	}
	if edited.Prompts.Design.Bytes != len(custom.Design) {
		t.Fatalf("design bytes=%d, want %d", edited.Prompts.Design.Bytes, len(custom.Design))
	}
}
