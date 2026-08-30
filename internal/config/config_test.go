package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if value, err := store.Load(); err != nil || value.Version != CurrentVersion || len(value.GitHub.Hosts) != 1 {
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

func TestConfigStoreRejectsUnknownFieldsAndInsecureFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("version: 1\nunknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Validate(); ErrorCodeOf(err) != ErrorCodeInvalid {
		t.Fatalf("unknown field error=%v code=%s", err, ErrorCodeOf(err))
	}
	if err := os.WriteFile(path, []byte("version: 1\ngithub:\n  hosts: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); ErrorCodeOf(err) != ErrorCodeInvalid {
		t.Fatalf("insecure mode error=%v code=%s", err, ErrorCodeOf(err))
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
