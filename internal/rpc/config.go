package rpc

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (h *Handler) requireConfig() (*config.Store, error) {
	if h.configStore == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("GitHub configuration is unavailable"))
	}
	return h.configStore, nil
}

func (h *Handler) GetConfig(
	ctx context.Context,
	_ *connect.Request[prxv1.GetConfigRequest],
) (*connect.Response[prxv1.GetConfigResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	value, err := store.Public()
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.GetConfigResponse{Config: protoGitHubConfig(value)}), nil
}

func (h *Handler) AddGitHubHost(
	ctx context.Context,
	req *connect.Request[prxv1.AddGitHubHostRequest],
) (*connect.Response[prxv1.AddGitHubHostResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Update(func(settings *config.Config) error {
		return settings.AddHost(config.Host{
			Host:      req.Msg.GetHost(),
			WebURL:    req.Msg.GetWebUrl(),
			APIURL:    req.Msg.GetApiUrl(),
			UploadURL: req.Msg.GetUploadUrl(),
		})
	})
	if err != nil {
		return nil, configRPCError(err)
	}
	host, _ := settings.HostFor(req.Msg.GetHost())
	return connect.NewResponse(&prxv1.AddGitHubHostResponse{Host: protoGitHubHost(host)}), nil
}

func (h *Handler) UpdateGitHubHost(
	ctx context.Context,
	req *connect.Request[prxv1.UpdateGitHubHostRequest],
) (*connect.Response[prxv1.UpdateGitHubHostResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Update(func(settings *config.Config) error {
		current, ok := settings.HostFor(req.Msg.GetHost())
		if !ok {
			return &config.Error{
				Code:    config.ErrorCodeNotFound,
				Message: fmt.Sprintf("GitHub host %q was not found", req.Msg.GetHost()),
			}
		}
		if req.Msg.NewHost != nil {
			current.Host = req.Msg.GetNewHost()
		}
		if req.Msg.WebUrl != nil {
			current.WebURL = req.Msg.GetWebUrl()
		}
		if req.Msg.ApiUrl != nil {
			current.APIURL = req.Msg.GetApiUrl()
		}
		if req.Msg.UploadUrl != nil {
			current.UploadURL = req.Msg.GetUploadUrl()
		}
		return settings.UpdateHost(req.Msg.GetHost(), current)
	})
	if err != nil {
		return nil, configRPCError(err)
	}
	hostName := req.Msg.GetHost()
	if req.Msg.NewHost != nil {
		hostName = req.Msg.GetNewHost()
	}
	host, _ := settings.HostFor(hostName)
	return connect.NewResponse(&prxv1.UpdateGitHubHostResponse{Host: protoGitHubHost(host)}), nil
}

func (h *Handler) DeleteGitHubHost(
	ctx context.Context,
	req *connect.Request[prxv1.DeleteGitHubHostRequest],
) (*connect.Response[prxv1.DeleteGitHubHostResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	if _, err := store.Update(
		func(settings *config.Config) error { return settings.RemoveHost(req.Msg.GetHost()) },
	); err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.DeleteGitHubHostResponse{}), nil
}

func (h *Handler) AddGitHubAuthMethod(
	ctx context.Context,
	req *connect.Request[prxv1.AddGitHubAuthMethodRequest],
) (*connect.Response[prxv1.AddGitHubAuthMethodResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	method, err := configAuthMethodFromAdd(req.Msg)
	if err != nil {
		return nil, configRPCError(err)
	}
	settings, err := store.Update(func(settings *config.Config) error { return settings.AddAuthMethod(method) })
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(
		&prxv1.AddGitHubAuthMethodResponse{AuthMethod: protoPublicAuthMethod(settings, method.ID)},
	), nil
}

func (h *Handler) UpdateGitHubAuthMethod(
	ctx context.Context,
	req *connect.Request[prxv1.UpdateGitHubAuthMethodRequest],
) (*connect.Response[prxv1.UpdateGitHubAuthMethodResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Update(func(settings *config.Config) error {
		current, ok := settings.AuthMethod(req.Msg.GetId())
		if !ok {
			return &config.Error{
				Code:    config.ErrorCodeNotFound,
				Message: fmt.Sprintf("auth method %q was not found", req.Msg.GetId()),
			}
		}
		if req.Msg.NewId != nil {
			current.ID = req.Msg.GetNewId()
		}
		if req.Msg.Host != nil {
			current.Host = req.Msg.GetHost()
		}
		if req.Msg.Type != nil {
			var typeErr error
			current.Type, typeErr = configAuthMethodType(req.Msg.GetType())
			if typeErr != nil {
				return typeErr
			}
			if current.Type != config.AuthMethodTypeInline {
				current.Token = ""
			}
		}
		if req.Msg.Account != nil {
			current.Account = req.Msg.GetAccount()
		}
		if req.Msg.Service != nil {
			current.Service = req.Msg.GetService()
		}
		if req.Msg.Variable != nil {
			current.Variable = req.Msg.GetVariable()
		}
		if req.Msg.User != nil {
			current.User = req.Msg.GetUser()
		}
		if req.Msg.Token != nil {
			current.Token = req.Msg.GetToken()
		}
		return settings.UpdateAuthMethod(req.Msg.GetId(), current)
	})
	if err != nil {
		return nil, configRPCError(err)
	}
	methodID := req.Msg.GetId()
	if req.Msg.NewId != nil {
		methodID = req.Msg.GetNewId()
	}
	return connect.NewResponse(
		&prxv1.UpdateGitHubAuthMethodResponse{AuthMethod: protoPublicAuthMethod(settings, methodID)},
	), nil
}

func (h *Handler) DeleteGitHubAuthMethod(
	ctx context.Context,
	req *connect.Request[prxv1.DeleteGitHubAuthMethodRequest],
) (*connect.Response[prxv1.DeleteGitHubAuthMethodResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	if _, err := store.Update(
		func(settings *config.Config) error { return settings.RemoveAuthMethod(req.Msg.GetId()) },
	); err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.DeleteGitHubAuthMethodResponse{}), nil
}

func (h *Handler) ReorderGitHubAuthMethods(
	ctx context.Context,
	req *connect.Request[prxv1.ReorderGitHubAuthMethodsRequest],
) (*connect.Response[prxv1.ReorderGitHubAuthMethodsResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Update(
		func(settings *config.Config) error { return settings.ReorderAuthMethods(req.Msg.GetIds()) },
	)
	if err != nil {
		return nil, configRPCError(err)
	}
	public := settings.Public()
	result := &prxv1.ReorderGitHubAuthMethodsResponse{}
	for _, method := range public.GitHub.AuthMethods {
		result.AuthMethods = append(result.AuthMethods, protoPublicAuth(method))
	}
	return connect.NewResponse(result), nil
}

func (h *Handler) ValidateConfig(
	ctx context.Context,
	_ *connect.Request[prxv1.ValidateConfigRequest],
) (*connect.Response[prxv1.ValidateConfigResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	if err := store.Validate(); err != nil {
		//nolint:nilerr // validation failures are returned as data in a successful RPC response.
		return connect.NewResponse(&prxv1.ValidateConfigResponse{Errors: []string{err.Error()}}), nil
	}
	return connect.NewResponse(&prxv1.ValidateConfigResponse{Valid: true}), nil
}

func configAuthMethodFromAdd(value *prxv1.AddGitHubAuthMethodRequest) (config.AuthMethod, error) {
	authType, err := configAuthMethodType(value.GetType())
	if err != nil {
		return config.AuthMethod{}, err
	}
	method := config.AuthMethod{
		ID: value.GetId(), Host: value.GetHost(), Type: authType, Account: value.GetAccount(),
		Service: value.GetService(), Variable: value.GetVariable(), User: value.GetUser(),
	}
	if value.Token != nil {
		if authType != config.AuthMethodTypeInline {
			return config.AuthMethod{}, &config.Error{
				Code:    config.ErrorCodeInvalid,
				Message: "token is only accepted for inline authentication",
			}
		}
		method.Token = value.GetToken()
	}
	return method, nil
}

func configAuthMethodType(value prxv1.GithubAuthMethodType) (config.AuthMethodType, error) {
	switch value {
	case prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_UNSPECIFIED:
		return "", &config.Error{Code: config.ErrorCodeInvalid, Message: "authentication method type is required"}
	case prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_KEYCHAIN:
		return config.AuthMethodTypeKeychain, nil
	case prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_ENVIRONMENT:
		return config.AuthMethodTypeEnvironment, nil
	case prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_INLINE:
		return config.AuthMethodTypeInline, nil
	case prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_GH_CLI:
		return config.AuthMethodTypeGHCLI, nil
	default:
		return "", &config.Error{Code: config.ErrorCodeInvalid, Message: "authentication method type is required"}
	}
}

func protoGitHubConfig(value config.PublicConfig) *prxv1.GitHubConfig {
	result := &prxv1.GitHubConfig{Version: int32(value.Version)}
	for _, host := range value.GitHub.Hosts {
		result.Hosts = append(result.Hosts, protoGitHubHost(host))
	}
	for _, method := range value.GitHub.AuthMethods {
		result.AuthMethods = append(result.AuthMethods, protoPublicAuth(method))
	}
	return result
}

func protoGitHubHost(value config.Host) *prxv1.GitHubHost {
	return &prxv1.GitHubHost{Host: value.Host, WebUrl: value.WebURL, ApiUrl: value.APIURL, UploadUrl: value.UploadURL}
}

func protoPublicAuthMethod(settings config.Config, id string) *prxv1.GitHubAuthMethod {
	value, ok := settings.AuthMethod(id)
	if !ok {
		return nil
	}
	public := settings.Public()
	for _, method := range public.GitHub.AuthMethods {
		if method.ID == value.ID {
			return protoPublicAuth(method)
		}
	}
	return nil
}

func protoPublicAuth(value config.PublicAuthMethod) *prxv1.GitHubAuthMethod {
	return &prxv1.GitHubAuthMethod{
		Id: value.ID, Host: value.Host, Type: protoAuthMethodType(value.Type), Account: value.Account,
		Service: value.Service, Variable: value.Variable, User: value.User,
		SecretConfigured: value.SecretConfigured, SecretHint: value.SecretHint,
	}
}

func protoAuthMethodType(value config.AuthMethodType) prxv1.GithubAuthMethodType {
	switch value {
	case config.AuthMethodTypeKeychain:
		return prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_KEYCHAIN
	case config.AuthMethodTypeEnvironment:
		return prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_ENVIRONMENT
	case config.AuthMethodTypeInline:
		return prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_INLINE
	case config.AuthMethodTypeGHCLI:
		return prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_GH_CLI
	default:
		return prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_UNSPECIFIED
	}
}

func configRPCError(err error) error {
	var configErr *config.Error
	if errors.As(err, &configErr) {
		code := domain.DomainErrorCodeInvalidConfig
		switch configErr.Code {
		case config.ErrorCodeInvalid:
			code = domain.DomainErrorCodeInvalidConfig
		case config.ErrorCodeNotFound:
			code = domain.DomainErrorCodeNotFound
		case config.ErrorCodeReferences:
			code = domain.DomainErrorCodeReferencesExist
		}
		return rpcError(domain.NewError(code, "%s", configErr.Message))
	}
	return rpcError(domain.NewError(domain.DomainErrorCodeInvalidConfig, "%s", err))
}
