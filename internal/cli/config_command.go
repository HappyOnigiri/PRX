package cli

import (
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) configCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "config",
		Short:   "Show or manage GitHub hosts and authentication",
		Example: "prx config",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			value, err := store.Public()
			if err != nil {
				return configCommandError(err)
			}
			return s.write(value, renderConfig(value))
		},
	}
	command.AddCommand(
		s.configPathCommand(),
		s.configValidateCommand(),
		s.configSyncCommand(),
		s.configHostCommand(),
		s.configAuthCommand(),
	)
	return command
}

func (s *state) configStore() (*config.Store, error) {
	return config.NewStore(s.configPath)
}

func (s *state) configSyncCommand() *cobra.Command {
	command := &cobra.Command{Use: "sync", Short: "Manage automatic GitHub synchronization settings"}
	command.AddCommand(s.configSyncUpdateCommand())
	return command
}

func (s *state) configSyncUpdateCommand() *cobra.Command {
	var interval int64
	command := &cobra.Command{
		Use:     "update",
		Short:   "Update the automatic GitHub synchronization interval",
		Example: "prx config sync update --interval-seconds 3600 --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Update(func(settings *config.Config) error {
				return settings.SetAutoSyncInterval(interval)
			})
			if err != nil {
				return configCommandError(err)
			}
			value := map[string]int64{"interval_seconds": settings.GitHub.AutoSyncIntervalSeconds}
			return s.write(value, renderMessage(
				"Automatic sync interval: %d seconds.",
				settings.GitHub.AutoSyncIntervalSeconds,
			))
		},
	}
	command.Flags().Int64Var(&interval, "interval-seconds", 0, "automatic sync interval in seconds (minimum 600)")
	_ = command.MarkFlagRequired("interval-seconds")
	return command
}

func (s *state) configPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "path",
		Short:   "Show the resolved configuration path",
		Example: "prx config path",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			return s.write(map[string]string{"path": store.Path()}, renderMessage("Config path: %s", store.Path()))
		},
	}
}

func (s *state) configValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "validate",
		Short:   "Validate the GitHub configuration",
		Example: "prx config validate",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			if err := store.Validate(); err != nil {
				return configCommandError(err)
			}
			return s.write(map[string]bool{"valid": true}, renderMessage("Configuration is valid."))
		},
	}
}

func (s *state) configHostCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "host",
		Short:   "List or manage configured GitHub hosts",
		Example: "prx config host",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Load()
			if err != nil {
				return configCommandError(err)
			}
			hosts := settings.Public().GitHub.Hosts
			return s.write(map[string]any{"hosts": hosts}, renderHostList(hosts))
		},
	}
	command.AddCommand(
		s.configHostAddCommand(),
		s.configHostUpdateCommand(),
		s.configHostRemoveCommand(),
	)
	return command
}

func (s *state) configHostAddCommand() *cobra.Command {
	var host, webURL, apiURL, uploadURL, graphqlURL string
	command := &cobra.Command{
		Use:     "add",
		Short:   "Add a GitHub.com or Enterprise host",
		Example: "prx config host add --host ghe.example.com",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			value, err := store.Update(func(settings *config.Config) error {
				return settings.AddHost(config.Host{
					Host: host, WebURL: webURL, APIURL: apiURL, UploadURL: uploadURL, GraphQLURL: graphqlURL,
				})
			})
			if err != nil {
				return configCommandError(err)
			}
			created := findHost(value, host)
			return s.write(created, renderMessage("Added GitHub host %s.", created.Host))
		},
	}
	command.Flags().StringVar(&host, "host", "", "hostname with optional port")
	command.Flags().StringVar(&webURL, "web-url", "", "HTTPS web URL (defaults from host)")
	command.Flags().StringVar(&apiURL, "api-url", "", "HTTPS API base URL (defaults from host)")
	command.Flags().StringVar(&uploadURL, "upload-url", "", "HTTPS upload base URL (defaults from host)")
	command.Flags().StringVar(&graphqlURL, "graphql-url", "", "HTTPS GraphQL URL (defaults from host)")
	_ = command.MarkFlagRequired("host")
	return command
}

func (s *state) configHostUpdateCommand() *cobra.Command {
	var newHost string
	var webURL, apiURL, uploadURL, graphqlURL string
	command := &cobra.Command{
		Use:     "update HOST",
		Short:   "Update a configured GitHub host",
		Example: "prx config host update ghe.example.com --api-url https://ghe.example.com/api/v3/",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			value, err := store.Update(func(settings *config.Config) error {
				current, ok := settings.HostFor(args[0])
				if !ok {
					return &config.Error{Code: config.ErrorCodeNotFound, Message: "GitHub host was not found"}
				}
				if cmd.Flags().Changed("new-host") {
					current.Host = newHost
				}
				if cmd.Flags().Changed("web-url") {
					current.WebURL = webURL
				}
				if cmd.Flags().Changed("api-url") {
					current.APIURL = apiURL
				}
				if cmd.Flags().Changed("upload-url") {
					current.UploadURL = uploadURL
				}
				if cmd.Flags().Changed("graphql-url") {
					current.GraphQLURL = graphqlURL
				}
				return settings.UpdateHost(args[0], current)
			})
			if err != nil {
				return configCommandError(err)
			}
			updated := findHost(value, newOrExistingHost(cmd, args[0], newHost))
			return s.write(updated, renderMessage("Updated GitHub host %s.", updated.Host))
		},
	}
	command.Flags().StringVar(&newHost, "new-host", "", "new hostname")
	command.Flags().StringVar(&webURL, "web-url", "", "new HTTPS web URL")
	command.Flags().StringVar(&apiURL, "api-url", "", "new HTTPS API base URL")
	command.Flags().StringVar(&uploadURL, "upload-url", "", "new HTTPS upload base URL")
	command.Flags().StringVar(&graphqlURL, "graphql-url", "", "new HTTPS GraphQL URL")
	return command
}

func (s *state) configHostRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "remove HOST",
		Short:   "Remove a configured GitHub host",
		Example: "prx config host remove ghe.example.com",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			if _, err := store.Update(
				func(settings *config.Config) error { return settings.RemoveHost(args[0]) },
			); err != nil {
				return configCommandError(err)
			}
			return s.write(map[string]string{"removed": args[0]}, renderMessage("Removed GitHub host %s.", args[0]))
		},
	}
}

func (s *state) configAuthCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "auth",
		Short:   "List or manage host-scoped authentication methods",
		Example: "prx config auth",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Load()
			if err != nil {
				return configCommandError(err)
			}
			methods := settings.Public().GitHub.AuthMethods
			return s.write(map[string]any{"auth_methods": methods}, renderAuthList(methods))
		},
	}
	command.AddCommand(
		s.configAuthAddCommand(),
		s.configAuthUpdateCommand(),
		s.configAuthRemoveCommand(),
		s.configAuthReorderCommand(),
	)
	return command
}

func (s *state) configAuthAddCommand() *cobra.Command {
	var id, host, authType, account, service, variable, user string
	var tokenStdin bool
	command := &cobra.Command{
		Use:     "add",
		Short:   "Add a host-scoped authentication method",
		Example: "prx config auth add --id work-gh --host github.com --type gh_cli",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, err := readConfigToken(cmd, tokenStdin)
			if err != nil {
				return configCommandError(err)
			}
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Update(func(settings *config.Config) error {
				return settings.AddAuthMethod(config.AuthMethod{
					ID: id, Host: host, Type: config.AuthMethodType(authType), Account: account,
					Service: service, Variable: variable, Token: token, User: user,
				})
			})
			if err != nil {
				return configCommandError(err)
			}
			created := findPublicAuth(settings, id)
			return s.write(created, renderMessage("Added authentication method %s.", created.ID))
		},
	}
	bindAuthFlags(command, &host, &authType, &account, &service, &variable, &user, &tokenStdin)
	command.Flags().StringVar(&id, "id", "", "authentication method ID")
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("host")
	_ = command.MarkFlagRequired("type")
	return command
}

func (s *state) configAuthUpdateCommand() *cobra.Command {
	var host, authType, account, service, variable, user string
	var tokenStdin bool
	var newID string
	command := &cobra.Command{
		Use:     "update AUTH_METHOD_ID",
		Short:   "Update a host-scoped authentication method",
		Example: "prx config auth update work-gh --user HappyOnigiri",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := readConfigToken(cmd, tokenStdin)
			if err != nil {
				return configCommandError(err)
			}
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Update(func(settings *config.Config) error {
				current, ok := settings.AuthMethod(args[0])
				if !ok {
					return &config.Error{Code: config.ErrorCodeNotFound, Message: "auth method was not found"}
				}
				if cmd.Flags().Changed("new-id") {
					current.ID = newID
				}
				if cmd.Flags().Changed("host") {
					current.Host = host
				}
				if cmd.Flags().Changed("type") {
					current.Type = config.AuthMethodType(authType)
				}
				if cmd.Flags().Changed("account") {
					current.Account = account
				}
				if cmd.Flags().Changed("service") {
					current.Service = service
				}
				if cmd.Flags().Changed("variable") {
					current.Variable = variable
				}
				if cmd.Flags().Changed("user") {
					current.User = user
				}
				if tokenStdin {
					current.Token = token
				}
				return settings.UpdateAuthMethod(args[0], current)
			})
			if err != nil {
				return configCommandError(err)
			}
			updated := findPublicAuth(settings, newOrExistingID(cmd, args[0], newID))
			return s.write(updated, renderMessage("Updated authentication method %s.", updated.ID))
		},
	}
	bindAuthFlags(command, &host, &authType, &account, &service, &variable, &user, &tokenStdin)
	command.Flags().StringVar(&newID, "new-id", "", "new authentication method ID")
	return command
}

func (s *state) configAuthRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "remove AUTH_METHOD_ID",
		Short:   "Remove an authentication method and its cached use",
		Example: "prx config auth remove work-gh",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			if _, err := store.Update(
				func(settings *config.Config) error { return settings.RemoveAuthMethod(args[0]) },
			); err != nil {
				return configCommandError(err)
			}
			return s.write(
				map[string]string{"removed": args[0]},
				renderMessage("Removed authentication method %s.", args[0]),
			)
		},
	}
}

func (s *state) configAuthReorderCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "reorder AUTH_METHOD_ID...",
		Short:   "Set authentication priority order",
		Example: "prx config auth reorder ghe-environment ghe-cli",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Update(
				func(settings *config.Config) error { return settings.ReorderAuthMethods(args) },
			)
			if err != nil {
				return configCommandError(err)
			}
			methods := settings.Public().GitHub.AuthMethods
			return s.write(map[string]any{"auth_methods": methods}, renderAuthList(methods))
		},
	}
}

func bindAuthFlags(command *cobra.Command, host, authType, account, service, variable, user *string, tokenStdin *bool) {
	command.Flags().StringVar(host, "host", "", "configured host")
	command.Flags().StringVar(authType, "type", "", "keychain, environment, inline, or gh_cli")
	command.Flags().StringVar(account, "account", "", "Keychain account")
	command.Flags().StringVar(service, "service", "", "Keychain service")
	command.Flags().StringVar(variable, "variable", "", "environment variable name")
	command.Flags().StringVar(user, "user", "", "gh CLI user")
	command.Flags().BoolVar(tokenStdin, "token-stdin", false, "read an inline token from stdin")
}

func readConfigToken(command *cobra.Command, requested bool) (string, error) {
	if !requested {
		return "", nil
	}
	body, err := io.ReadAll(command.InOrStdin())
	if err != nil {
		return "", &config.Error{Code: config.ErrorCodeInvalid, Message: "read inline token: " + err.Error()}
	}
	token := strings.TrimSpace(string(body))
	if token == "" {
		return "", &config.Error{Code: config.ErrorCodeInvalid, Message: "inline token is empty"}
	}
	return token, nil
}

func findHost(settings config.Config, host string) config.Host {
	value, _ := settings.HostFor(host)
	return value
}

func findPublicAuth(settings config.Config, id string) config.PublicAuthMethod {
	value, ok := settings.AuthMethod(id)
	if !ok {
		return config.PublicAuthMethod{}
	}
	public := config.Config{
		Version: config.CurrentVersion,
		GitHub: config.GitHubConfig{
			Hosts:       []config.Host{config.DefaultHost()},
			AuthMethods: []config.AuthMethod{value},
		},
	}.Public()
	return public.GitHub.AuthMethods[0]
}

func newOrExistingHost(command *cobra.Command, existing, replacement string) string {
	if command.Flags().Changed("new-host") {
		return replacement
	}
	return existing
}

func newOrExistingID(command *cobra.Command, existing, replacement string) string {
	if command.Flags().Changed("new-id") {
		return replacement
	}
	return existing
}

func configCommandError(err error) error {
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
		return domain.NewError(code, "%s", configErr.Message)
	}
	return domain.NewError(domain.DomainErrorCodeInvalidConfig, "%s", err)
}
