package github

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gh "github.com/google/go-github/v80/github"

	"github.com/HappyOnigiri/PRX/internal/config"
)

func TestResolverScopesCandidatesAndSkipsDuplicateCredentials(t *testing.T) {
	t.Setenv("FIRST_TOKEN", "same-token")
	value := config.Config{
		Version: config.CurrentVersion,
		GitHub: config.GitHubConfig{
			Hosts: []config.Host{config.DefaultHost(), {Host: "ghe.example.com"}},
			AuthMethods: []config.AuthMethod{
				{ID: "github", Host: "github.com", Type: config.AuthMethodTypeEnvironment, Variable: "FIRST_TOKEN"},
				{ID: "ghe", Host: "ghe.example.com", Type: config.AuthMethodTypeInline, Token: "ghe-token"},
			},
		},
	}
	resolver, err := NewResolver(value)
	if err != nil {
		t.Fatal(err)
	}
	if resolver.Config().Version != config.CurrentVersion {
		t.Fatalf("resolver config version=%d", resolver.Config().Version)
	}
	if !resolver.HasHost("GHE.EXAMPLE.COM") || resolver.HasHost("unknown.example.com") {
		t.Fatal("host lookup did not respect normalized host boundaries")
	}
	candidates := resolver.Candidates("GHE.EXAMPLE.COM")
	if len(candidates) != 1 || candidates[0].ID != "ghe" {
		t.Fatalf("GHE candidates=%+v", candidates)
	}
	githubCandidates := resolver.Candidates("github.com")
	if len(githubCandidates) != 1 || githubCandidates[0].ID != "github" {
		t.Fatalf("GitHub candidates=%+v", githubCandidates)
	}
	provider, err := resolver.Open(context.Background(), githubCandidates[0])
	if err != nil || provider == nil {
		t.Fatalf("open provider=%v err=%v", provider, err)
	}
	duplicate, err := resolver.Open(context.Background(), githubCandidates[0])
	if duplicate != nil || ClassOf(err) != ErrorClassAuthUnavailable {
		t.Fatalf("duplicate provider=%v class=%s err=%v", duplicate, ClassOf(err), err)
	}
	resolver.MarkUnauthorized(githubCandidates[0].ID)
	if _, err := resolver.Open(context.Background(), githubCandidates[0]); ClassOf(err) != ErrorClassAuthUnavailable {
		t.Fatalf("unauthorized method class=%s err=%v", ClassOf(err), err)
	}
	if _, err := readToken(
		context.Background(),
		"ghe.example.com",
		config.AuthMethod{Type: config.AuthMethodTypeInline, Token: "inline-token"},
	); err != nil {
		t.Fatal(err)
	}
}

func TestResolverSupportsImplicitGitHubDiscoveryOnly(t *testing.T) {
	resolver, err := NewResolver(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	candidates := resolver.Candidates("github.com")
	if len(candidates) != 3 || candidates[0].Method.Variable != "GITHUB_TOKEN" ||
		candidates[1].Method.Variable != "GH_TOKEN" || candidates[2].Method.Type != config.AuthMethodTypeGHCLI {
		t.Fatalf("implicit candidates=%+v", candidates)
	}
	if got := resolver.Candidates("ghe.example.com"); got != nil {
		t.Fatalf("implicit GHE candidates=%+v", got)
	}

	t.Setenv("GH_TOKEN", "env-token")
	if token, err := readToken(
		context.Background(),
		"github.com",
		candidates[1].Method,
	); err != nil ||
		token != "env-token" {
		t.Fatalf("environment token=%q err=%v", token, err)
	}
	if _, err := readToken(
		context.Background(),
		"github.com",
		config.AuthMethod{Type: config.AuthMethodTypeEnvironment, Variable: "MISSING_TOKEN"},
	); ClassOf(
		err,
	) != ErrorClassAuthUnavailable {
		t.Fatalf("missing environment token class=%s err=%v", ClassOf(err), err)
	}
	if _, err := readToken(
		context.Background(),
		"github.com",
		config.AuthMethod{Type: config.AuthMethodType("unknown")},
	); ClassOf(
		err,
	) != ErrorClassAuthUnavailable {
		t.Fatalf("unknown auth type class=%s err=%v", ClassOf(err), err)
	}
}

func TestResolverReadsHostScopedGHCLIWithoutTokenEnvironment(t *testing.T) {
	directory := t.TempDir()
	ghPath := filepath.Join(directory, "gh")
	content := []byte("#!/bin/sh\n" +
		"test -z \"$GITHUB_TOKEN\" || exit 10\n" +
		"test -z \"$GH_TOKEN\" || exit 11\n" +
		"test \"$1\" = auth || exit 12\n" +
		"test \"$2\" = token || exit 13\n" +
		"test \"$4\" = ghe.example.com || exit 14\n" +
		"printf cli-token\n")
	if err := os.WriteFile(ghPath, content, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GITHUB_TOKEN", "should-not-be-inherited")
	t.Setenv("GH_TOKEN", "should-not-be-inherited")
	token, err := readToken(
		context.Background(),
		"ghe.example.com",
		config.AuthMethod{Type: config.AuthMethodTypeGHCLI},
	)
	if err != nil || token != "cli-token" {
		t.Fatalf("gh CLI token=%q err=%v", token, err)
	}
	if got := filteredGHEnvironment(); strings.Contains(strings.Join(got, "\n"), "GITHUB_TOKEN=") ||
		strings.Contains(strings.Join(got, "\n"), "GH_TOKEN=") {
		t.Fatal("filtered gh environment retained a GitHub token variable")
	}
}

func TestProviderErrorClassification(t *testing.T) {
	cases := []struct {
		name   string
		status int
		header string
		want   ErrorClass
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, want: ErrorClassUnauthorized},
		{name: "not found", status: http.StatusNotFound, want: ErrorClassNotFound},
		{name: "permission", status: http.StatusForbidden, want: ErrorClassPermission},
		{name: "rate limit", status: http.StatusForbidden, header: "0", want: ErrorClassRateLimit},
		{name: "too many requests", status: http.StatusTooManyRequests, want: ErrorClassRateLimit},
		{name: "server", status: http.StatusBadGateway, want: ErrorClassTransient},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			response := &gh.Response{Response: &http.Response{StatusCode: test.status, Header: make(http.Header)}}
			if test.header != "" {
				response.Header.Set("X-RateLimit-Remaining", test.header)
			}
			err := wrapProviderError("operation", errors.New("request failed"), response)
			if ClassOf(err) != test.want || StatusCodeOf(err) != test.status {
				t.Fatalf("class=%s status=%d err=%v", ClassOf(err), StatusCodeOf(err), err)
			}
		})
	}
	sentinel := errors.New("sentinel")
	if !errors.Is(wrapProviderError("operation", sentinel, nil), sentinel) {
		t.Fatal("provider error did not unwrap its cause")
	}
	if ClassOf(context.DeadlineExceeded) != ErrorClassTransient {
		t.Fatalf("deadline class=%s", ClassOf(context.DeadlineExceeded))
	}
	if ClassOf(errors.New("ordinary failure")) != ErrorClassOther {
		t.Fatalf("ordinary class=%s", ClassOf(errors.New("ordinary failure")))
	}
	if StatusCodeOf(errors.New("ordinary failure")) != 0 {
		t.Fatalf("ordinary status=%d", StatusCodeOf(errors.New("ordinary failure")))
	}
}

func TestRedirectOriginChecks(t *testing.T) {
	first, _ := url.Parse("https://ghe.example.com/api/v3/")
	second, _ := url.Parse("https://ghe.example.com/api/v3/repos")
	other, _ := url.Parse("https://github.com/repos")
	if !sameOrigin(first, second) || sameOrigin(first, other) || sameOrigin(nil, second) {
		t.Fatal("same-origin comparison returned an unexpected result")
	}
	if err := rejectCrossOriginRedirect(&http.Request{URL: second}, []*http.Request{{URL: first}}); err != nil {
		t.Fatalf("same-origin redirect rejected: %v", err)
	}
	if err := rejectCrossOriginRedirect(&http.Request{URL: other}, []*http.Request{{URL: first}}); err == nil {
		t.Fatal("cross-origin redirect was accepted")
	}
}

func TestPullRequestURLDetailsAcceptsEnterpriseHost(t *testing.T) {
	host, owner, repository, number, canonical, err := ParsePullRequestURLDetails(
		"https://GHE.Example.com:8443/Acme/API/pull/42",
	)
	if err != nil || host != "ghe.example.com:8443" || owner != "Acme" || repository != "API" || number != 42 ||
		canonical != "https://ghe.example.com:8443/Acme/API/pull/42" {
		t.Fatalf("details=%q/%q/%q/%d/%q err=%v", host, owner, repository, number, canonical, err)
	}
}
