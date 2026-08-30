package github

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/config"
)

type Candidate struct {
	ID     string
	Method config.AuthMethod
	Host   config.Host
}

type Resolver struct {
	config       config.Config
	unauthorized map[string]bool
	fingerprints map[string]struct{}
	httpClient   *http.Client
	providers    map[string]*LiveProvider
}

func NewResolver(value config.Config, clients ...*http.Client) (*Resolver, error) {
	normalized, err := value.Normalize()
	if err != nil {
		return nil, err
	}
	var httpClient *http.Client
	if len(clients) > 0 {
		httpClient = clients[0]
	}
	return &Resolver{
		config:       normalized,
		unauthorized: map[string]bool{},
		fingerprints: map[string]struct{}{},
		httpClient:   httpClient,
		providers:    map[string]*LiveProvider{},
	}, nil
}

func (r *Resolver) Config() config.Config { return r.config }

func (r *Resolver) HasHost(host string) bool {
	_, ok := r.config.HostFor(host)
	return ok
}

func (r *Resolver) Candidates(host string) []Candidate {
	hostValue, ok := r.config.HostFor(host)
	if !ok {
		return nil
	}
	host = hostValue.Host
	if r.config.GitHub.AuthMethods == nil {
		if host != "github.com" {
			return nil
		}
		return []Candidate{
			{
				ID:   "implicit-github-token",
				Host: hostValue,
				Method: config.AuthMethod{
					ID:       "implicit-github-token",
					Host:     host,
					Type:     config.AuthMethodTypeEnvironment,
					Variable: "GITHUB_TOKEN",
				},
			},
			{
				ID:   "implicit-gh-token",
				Host: hostValue,
				Method: config.AuthMethod{
					ID:       "implicit-gh-token",
					Host:     host,
					Type:     config.AuthMethodTypeEnvironment,
					Variable: "GH_TOKEN",
				},
			},
			{
				ID:     "implicit-gh-cli",
				Host:   hostValue,
				Method: config.AuthMethod{ID: "implicit-gh-cli", Host: host, Type: config.AuthMethodTypeGHCLI},
			},
		}
	}
	result := make([]Candidate, 0, len(r.config.GitHub.AuthMethods))
	for _, method := range r.config.GitHub.AuthMethods {
		if method.Host == host {
			result = append(result, Candidate{ID: method.ID, Method: method, Host: hostValue})
		}
	}
	return result
}

func (r *Resolver) MarkUnauthorized(id string) { r.unauthorized[id] = true }

func (r *Resolver) Open(ctx context.Context, candidate Candidate) (*LiveProvider, error) {
	if r.unauthorized[candidate.ID] {
		return nil, unavailableError("authentication method is unavailable for this sync")
	}
	if provider := r.providers[candidate.ID]; provider != nil {
		return provider, nil
	}
	token, err := readToken(ctx, candidate.Host.Host, candidate.Method)
	if err != nil {
		return nil, err
	}
	fingerprint := credentialFingerprint(candidate.Host.Host, token)
	if _, exists := r.fingerprints[fingerprint]; exists {
		return nil, duplicateCredentialError()
	}
	r.fingerprints[fingerprint] = struct{}{}
	provider, err := NewConfiguredLiveProvider(
		ctx,
		token,
		candidate.Host.APIURL,
		candidate.Host.UploadURL,
		candidate.Host.GraphQLURL,
		r.httpClient,
	)
	if err != nil {
		return nil, err
	}
	r.providers[candidate.ID] = provider
	return provider, nil
}

func readToken(ctx context.Context, host string, method config.AuthMethod) (string, error) {
	var token string
	switch method.Type {
	case config.AuthMethodTypeKeychain:
		command := exec.CommandContext(
			ctx,
			"/usr/bin/security",
			"find-generic-password",
			"-a",
			method.Account,
			"-s",
			method.Service,
			"-w",
		)
		output, err := command.Output()
		if err != nil {
			return "", unavailableError("the configured Keychain credential is unavailable")
		}
		token = string(output)
	case config.AuthMethodTypeEnvironment:
		token = os.Getenv(method.Variable)
	case config.AuthMethodTypeInline:
		token = method.Token
	case config.AuthMethodTypeGHCLI:
		args := []string{"auth", "token", "--hostname", host}
		if method.User != "" {
			args = append(args, "--user", method.User)
		}
		command := exec.CommandContext(ctx, "gh", args...)
		command.Env = filteredGHEnvironment()
		output, err := command.Output()
		if err != nil {
			return "", unavailableError("the configured gh CLI credential is unavailable")
		}
		token = string(output)
	default:
		return "", unavailableError(fmt.Sprintf("unsupported authentication method %q", method.Type))
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", unavailableError("the configured GitHub credential is empty")
	}
	return token, nil
}

func filteredGHEnvironment() []string {
	result := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		name, _, ok := strings.Cut(value, "=")
		if !ok {
			continue
		}
		if slices.Contains([]string{
			"GITHUB_TOKEN",
			"GH_TOKEN",
			"GH_ENTERPRISE_TOKEN",
			"GITHUB_ENTERPRISE_TOKEN",
			"GH_HOST",
		}, name) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func credentialFingerprint(host, token string) string {
	hash := sha256.Sum256([]byte(host + "\x00" + token))
	return hex.EncodeToString(hash[:])
}
