// Package config owns the versioned YAML configuration shared by the CLI,
// server, and GitHub authentication resolver.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	CurrentVersion                 = 1
	DefaultAutoSyncIntervalSeconds = int64(3600)
	MinimumAutoSyncIntervalSeconds = int64(600)
)

type AuthMethodType string

const (
	AuthMethodTypeKeychain    AuthMethodType = "keychain"
	AuthMethodTypeEnvironment AuthMethodType = "environment"
	AuthMethodTypeInline      AuthMethodType = "inline"
	AuthMethodTypeGHCLI       AuthMethodType = "gh_cli"
)

type Host struct {
	Host       string `yaml:"host"        json:"host"`
	WebURL     string `yaml:"web_url"     json:"web_url"`
	APIURL     string `yaml:"api_url"     json:"api_url"`
	UploadURL  string `yaml:"upload_url"  json:"upload_url"`
	GraphQLURL string `yaml:"graphql_url" json:"graphql_url"`
}

type AuthMethod struct {
	ID       string         `yaml:"id"                 json:"id"`
	Host     string         `yaml:"host"               json:"host"`
	Type     AuthMethodType `yaml:"type"               json:"type"`
	Account  string         `yaml:"account,omitempty"  json:"account,omitempty"`
	Service  string         `yaml:"service,omitempty"  json:"service,omitempty"`
	Variable string         `yaml:"variable,omitempty" json:"variable,omitempty"`
	Token    string         `yaml:"token,omitempty"    json:"-"`
	User     string         `yaml:"user,omitempty"     json:"user,omitempty"`
}

type GitHubConfig struct {
	Hosts                   []Host       `yaml:"hosts"                                json:"hosts"`
	AuthMethods             []AuthMethod `yaml:"auth_methods"                         json:"auth_methods"`
	AutoSyncIntervalSeconds int64        `yaml:"auto_sync_interval_seconds,omitempty" json:"auto_sync_interval_seconds"`
}

// MarshalYAML keeps an omitted auth_methods list distinct from an explicitly
// empty list. The former enables the historical GitHub.com credential
// discovery; the latter intentionally disables all implicit candidates.
func (c GitHubConfig) MarshalYAML() (any, error) {
	type yamlGitHubConfig struct {
		Hosts                   []Host        `yaml:"hosts"`
		AuthMethods             *[]AuthMethod `yaml:"auth_methods,omitempty"`
		AutoSyncIntervalSeconds int64         `yaml:"auto_sync_interval_seconds"`
	}
	var methods *[]AuthMethod
	if c.AuthMethods != nil {
		methodsCopy := append([]AuthMethod{}, c.AuthMethods...)
		methods = &methodsCopy
	}
	return yamlGitHubConfig{
		Hosts: c.Hosts, AuthMethods: methods, AutoSyncIntervalSeconds: c.AutoSyncIntervalSeconds,
	}, nil
}

// Config is the on-disk configuration. AuthMethod.Token is deliberately not
// serializable as JSON; callers should use Public for any human or RPC output.
type Config struct {
	Version int          `yaml:"version" json:"version"`
	GitHub  GitHubConfig `yaml:"github"  json:"github"`
}

type PublicAuthMethod struct {
	ID               string         `json:"id"`
	Host             string         `json:"host"`
	Type             AuthMethodType `json:"type"`
	Account          string         `json:"account,omitempty"`
	Service          string         `json:"service,omitempty"`
	Variable         string         `json:"variable,omitempty"`
	User             string         `json:"user,omitempty"`
	SecretConfigured bool           `json:"secret_configured"`
	SecretHint       string         `json:"secret_hint,omitempty"`
}

type PublicGitHubConfig struct {
	Hosts                   []Host             `json:"hosts"`
	AuthMethods             []PublicAuthMethod `json:"auth_methods"`
	AutoSyncIntervalSeconds int64              `json:"auto_sync_interval_seconds"`
}

type PublicConfig struct {
	Version int                `json:"version"`
	GitHub  PublicGitHubConfig `json:"github"`
}

type ErrorCode string

const (
	ErrorCodeInvalid    ErrorCode = "invalid_config"
	ErrorCodeNotFound   ErrorCode = "not_found"
	ErrorCodeReferences ErrorCode = "references_exist"
)

type Error struct {
	Code    ErrorCode
	Message string
}

func (e *Error) Error() string { return e.Message }

func newError(code ErrorCode, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func ErrorCodeOf(err error) ErrorCode {
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	return ErrorCodeInvalid
}

var (
	envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	authIDPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

func DefaultHost() Host {
	return Host{
		Host:       "github.com",
		WebURL:     "https://github.com",
		APIURL:     "https://api.github.com/",
		UploadURL:  "https://uploads.github.com/",
		GraphQLURL: "https://api.github.com/graphql",
	}
}

func Default() Config {
	return Config{
		Version: CurrentVersion,
		GitHub:  GitHubConfig{Hosts: []Host{DefaultHost()}},
	}
}

// NormalizeHost returns the case-insensitive host key used by PRs, config, and
// the authentication cache. A port is part of the key so two local GHE
// instances cannot share credentials accidentally.
func NormalizeHost(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "/?#@\t\r\n ") {
		return "", newError(ErrorCodeInvalid, "host must be a hostname with an optional port")
	}
	parsed, err := url.Parse("https://" + value)
	if err != nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.Path != "" || parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		parsed.User != nil || strings.HasSuffix(parsed.Host, ":") {
		return "", newError(ErrorCodeInvalid, "host %q is invalid", value)
	}
	if port := parsed.Port(); port != "" {
		portNumber, portErr := strconv.Atoi(port)
		if portErr != nil || portNumber < 1 || portNumber > 65535 {
			return "", newError(ErrorCodeInvalid, "host %q has an invalid port", value)
		}
	}
	return strings.ToLower(parsed.Host), nil
}

func (c Config) Normalize() (Config, error) {
	if c.Version != CurrentVersion {
		return Config{}, newError(ErrorCodeInvalid, "config version must be %d", CurrentVersion)
	}
	result := c
	result.GitHub.Hosts = append([]Host(nil), c.GitHub.Hosts...)
	if c.GitHub.AuthMethods != nil {
		result.GitHub.AuthMethods = append([]AuthMethod{}, c.GitHub.AuthMethods...)
	}
	if len(result.GitHub.Hosts) == 0 {
		result.GitHub.Hosts = []Host{DefaultHost()}
	}
	if result.GitHub.AutoSyncIntervalSeconds == 0 {
		result.GitHub.AutoSyncIntervalSeconds = DefaultAutoSyncIntervalSeconds
	}
	if result.GitHub.AutoSyncIntervalSeconds < MinimumAutoSyncIntervalSeconds {
		return Config{}, newError(
			ErrorCodeInvalid,
			"github.auto_sync_interval_seconds must be at least %d",
			MinimumAutoSyncIntervalSeconds,
		)
	}

	hosts := make(map[string]struct{}, len(result.GitHub.Hosts))
	for index := range result.GitHub.Hosts {
		host, err := normalizeHostConfig(result.GitHub.Hosts[index])
		if err != nil {
			return Config{}, fmt.Errorf("github.hosts[%d]: %w", index, err)
		}
		if _, exists := hosts[host.Host]; exists {
			return Config{}, newError(ErrorCodeInvalid, "duplicate GitHub host %q", host.Host)
		}
		hosts[host.Host] = struct{}{}
		result.GitHub.Hosts[index] = host
	}

	ids := make(map[string]struct{}, len(result.GitHub.AuthMethods))
	for index := range result.GitHub.AuthMethods {
		method := result.GitHub.AuthMethods[index]
		method.ID = strings.TrimSpace(method.ID)
		if !authIDPattern.MatchString(method.ID) {
			return Config{}, newError(ErrorCodeInvalid, "auth method id %q is invalid", method.ID)
		}
		normalizedHost, err := NormalizeHost(method.Host)
		if err != nil {
			return Config{}, fmt.Errorf("github.auth_methods[%d]: %w", index, err)
		}
		method.Host = normalizedHost
		if _, exists := hosts[method.Host]; !exists {
			return Config{}, newError(
				ErrorCodeInvalid,
				"auth method %q references unknown host %q",
				method.ID,
				method.Host,
			)
		}
		if _, exists := ids[method.ID]; exists {
			return Config{}, newError(ErrorCodeInvalid, "duplicate auth method id %q", method.ID)
		}
		ids[method.ID] = struct{}{}
		method.Type = AuthMethodType(strings.ToLower(strings.TrimSpace(string(method.Type))))
		if err := validateAuthMethod(method); err != nil {
			return Config{}, fmt.Errorf("github.auth_methods[%d]: %w", index, err)
		}
		method.Token = strings.TrimSpace(method.Token)
		result.GitHub.AuthMethods[index] = method
	}
	return result, nil
}

func (c Config) Validate() error {
	_, err := c.Normalize()
	return err
}

func normalizeHostConfig(value Host) (Host, error) {
	host, err := NormalizeHost(value.Host)
	if err != nil {
		return Host{}, err
	}
	value.Host = host
	if value.WebURL == "" {
		if host == "github.com" {
			value.WebURL = DefaultHost().WebURL
		} else {
			value.WebURL = "https://" + host
		}
	}
	if value.APIURL == "" {
		if host == "github.com" {
			value.APIURL = DefaultHost().APIURL
		} else {
			value.APIURL = "https://" + host + "/api/v3/"
		}
	}
	if value.UploadURL == "" {
		if host == "github.com" {
			value.UploadURL = DefaultHost().UploadURL
		} else {
			value.UploadURL = "https://" + host + "/api/uploads/"
		}
	}
	if value.GraphQLURL == "" {
		if host == "github.com" {
			value.GraphQLURL = DefaultHost().GraphQLURL
		} else {
			value.GraphQLURL = "https://" + host + "/api/graphql"
		}
	}
	var errURL error
	value.WebURL, errURL = normalizeURL(value.WebURL, "web_url", false)
	if errURL != nil {
		return Host{}, errURL
	}
	value.APIURL, errURL = normalizeURL(value.APIURL, "api_url", true)
	if errURL != nil {
		return Host{}, errURL
	}
	value.UploadURL, errURL = normalizeURL(value.UploadURL, "upload_url", true)
	if errURL != nil {
		return Host{}, errURL
	}
	value.GraphQLURL, errURL = normalizeURL(value.GraphQLURL, "graphql_url", false)
	if errURL != nil {
		return Host{}, errURL
	}
	web, _ := url.Parse(value.WebURL)
	if strings.ToLower(web.Host) != host {
		return Host{}, newError(ErrorCodeInvalid, "web_url host %q does not match host %q", web.Host, host)
	}
	api, _ := url.Parse(value.APIURL)
	upload, _ := url.Parse(value.UploadURL)
	graphql, _ := url.Parse(value.GraphQLURL)
	if host == "github.com" {
		if !oneOfHost(api.Host, "api.github.com", "github.com") ||
			!oneOfHost(upload.Host, "uploads.github.com", "github.com") ||
			!oneOfHost(graphql.Host, "api.github.com", "github.com") {
			return Host{}, newError(
				ErrorCodeInvalid,
				"GitHub.com API, upload, and GraphQL URLs must stay on GitHub origins",
			)
		}
	} else if strings.ToLower(api.Host) != host || strings.ToLower(upload.Host) != host ||
		strings.ToLower(graphql.Host) != host {
		return Host{}, newError(ErrorCodeInvalid, "API, upload, and GraphQL URL hosts must match GitHub host %q", host)
	}
	return value, nil
}

func oneOfHost(value string, allowed ...string) bool {
	value = strings.ToLower(value)
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func normalizeURL(value, field string, trailingSlash bool) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return "", newError(ErrorCodeInvalid, "%s must be an HTTPS URL without user info, query, or fragment", field)
	}
	if trailingSlash && !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	return parsed.String(), nil
}

func validateAuthMethod(value AuthMethod) error {
	if value.Type != AuthMethodTypeInline && strings.TrimSpace(value.Token) != "" {
		return newError(ErrorCodeInvalid, "token is only valid for inline authentication")
	}
	switch value.Type {
	case AuthMethodTypeKeychain:
		if strings.TrimSpace(value.Account) == "" || strings.TrimSpace(value.Service) == "" {
			return newError(ErrorCodeInvalid, "keychain auth requires account and service")
		}
	case AuthMethodTypeEnvironment:
		if !envNamePattern.MatchString(strings.TrimSpace(value.Variable)) {
			return newError(ErrorCodeInvalid, "environment auth requires a valid variable name")
		}
	case AuthMethodTypeInline:
		if strings.TrimSpace(value.Token) == "" {
			return newError(ErrorCodeInvalid, "inline auth requires a token")
		}
	case AuthMethodTypeGHCLI:
	default:
		return newError(ErrorCodeInvalid, "auth method type %q is unsupported", value.Type)
	}
	return nil
}

func (c Config) HostFor(value string) (Host, bool) {
	host, err := NormalizeHost(value)
	if err != nil {
		return Host{}, false
	}
	for _, item := range c.GitHub.Hosts {
		if item.Host == host {
			return item, true
		}
	}
	return Host{}, false
}

func (c Config) AuthMethod(id string) (AuthMethod, bool) {
	for _, method := range c.GitHub.AuthMethods {
		if method.ID == id {
			return method, true
		}
	}
	return AuthMethod{}, false
}

func (c *Config) AddHost(value Host) error {
	c.GitHub.Hosts = append(c.GitHub.Hosts, value)
	return c.normalizeInPlace()
}

func (c *Config) UpdateHost(existing string, value Host) error {
	existing, err := NormalizeHost(existing)
	if err != nil {
		return err
	}
	for index := range c.GitHub.Hosts {
		if c.GitHub.Hosts[index].Host == existing {
			newHost, err := NormalizeHost(value.Host)
			if err != nil {
				return err
			}
			updated := *c
			updated.GitHub.Hosts = append([]Host{}, c.GitHub.Hosts...)
			if c.GitHub.AuthMethods != nil {
				updated.GitHub.AuthMethods = append([]AuthMethod{}, c.GitHub.AuthMethods...)
			}
			value.Host = newHost
			updated.GitHub.Hosts[index] = value
			for methodIndex := range updated.GitHub.AuthMethods {
				if updated.GitHub.AuthMethods[methodIndex].Host == existing {
					updated.GitHub.AuthMethods[methodIndex].Host = newHost
				}
			}
			normalized, err := updated.Normalize()
			if err != nil {
				return err
			}
			*c = normalized
			return nil
		}
	}
	return newError(ErrorCodeNotFound, "GitHub host %q was not found", existing)
}

func (c *Config) RemoveHost(value string) error {
	host, err := NormalizeHost(value)
	if err != nil {
		return err
	}
	if host == "github.com" {
		return newError(ErrorCodeInvalid, "github.com is the default host and cannot be removed")
	}
	for _, method := range c.GitHub.AuthMethods {
		if method.Host == host {
			return newError(ErrorCodeReferences, "GitHub host %q is used by auth method %q", host, method.ID)
		}
	}
	for index := range c.GitHub.Hosts {
		if c.GitHub.Hosts[index].Host == host {
			c.GitHub.Hosts = append(c.GitHub.Hosts[:index], c.GitHub.Hosts[index+1:]...)
			return c.normalizeInPlace()
		}
	}
	return newError(ErrorCodeNotFound, "GitHub host %q was not found", host)
}

func (c *Config) AddAuthMethod(value AuthMethod) error {
	c.GitHub.AuthMethods = append(c.GitHub.AuthMethods, value)
	return c.normalizeInPlace()
}

func (c *Config) UpdateAuthMethod(existing string, value AuthMethod) error {
	for index := range c.GitHub.AuthMethods {
		if c.GitHub.AuthMethods[index].ID == existing {
			if value.Type != AuthMethodTypeInline {
				value.Token = ""
			}
			c.GitHub.AuthMethods[index] = value
			return c.normalizeInPlace()
		}
	}
	return newError(ErrorCodeNotFound, "auth method %q was not found", existing)
}

func (c *Config) RemoveAuthMethod(id string) error {
	for index := range c.GitHub.AuthMethods {
		if c.GitHub.AuthMethods[index].ID == id {
			c.GitHub.AuthMethods = append(c.GitHub.AuthMethods[:index], c.GitHub.AuthMethods[index+1:]...)
			return c.normalizeInPlace()
		}
	}
	return newError(ErrorCodeNotFound, "auth method %q was not found", id)
}

func (c *Config) ReorderAuthMethods(ids []string) error {
	if len(ids) != len(c.GitHub.AuthMethods) {
		return newError(ErrorCodeInvalid, "auth reorder must include every auth method exactly once")
	}
	methods := make(map[string]AuthMethod, len(c.GitHub.AuthMethods))
	for _, method := range c.GitHub.AuthMethods {
		methods[method.ID] = method
	}
	ordered := make([]AuthMethod, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		method, ok := methods[id]
		if !ok {
			return newError(ErrorCodeInvalid, "auth reorder contains unknown method %q", id)
		}
		if _, ok := seen[id]; ok {
			return newError(ErrorCodeInvalid, "auth reorder contains duplicate method %q", id)
		}
		seen[id] = struct{}{}
		ordered = append(ordered, method)
	}
	c.GitHub.AuthMethods = ordered
	return c.normalizeInPlace()
}

func (c *Config) SetAutoSyncInterval(seconds int64) error {
	c.GitHub.AutoSyncIntervalSeconds = seconds
	return c.normalizeInPlace()
}

func (c *Config) normalizeInPlace() error {
	normalized, err := c.Normalize()
	if err != nil {
		return err
	}
	*c = normalized
	return nil
}

func (c Config) Public() PublicConfig {
	result := PublicConfig{
		Version: c.Version,
		GitHub: PublicGitHubConfig{
			Hosts:                   append([]Host(nil), c.GitHub.Hosts...),
			AuthMethods:             make([]PublicAuthMethod, 0, len(c.GitHub.AuthMethods)),
			AutoSyncIntervalSeconds: c.GitHub.AutoSyncIntervalSeconds,
		},
	}
	for _, method := range c.GitHub.AuthMethods {
		public := PublicAuthMethod{
			ID:       method.ID,
			Host:     method.Host,
			Type:     method.Type,
			Account:  method.Account,
			Service:  method.Service,
			Variable: method.Variable,
			User:     method.User,
			SecretConfigured: method.Token != "" || method.Type == AuthMethodTypeEnvironment ||
				method.Type == AuthMethodTypeKeychain ||
				method.Type == AuthMethodTypeGHCLI,
		}
		if method.Type == AuthMethodTypeInline {
			public.SecretConfigured = method.Token != ""
			public.SecretHint = maskSecret(method.Token)
		}
		result.GitHub.AuthMethods = append(result.GitHub.AuthMethods, public)
	}
	return result
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "••••"
	}
	return value[:4] + "…" + value[len(value)-4:]
}
