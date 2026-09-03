package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

// automaticSyncTimeout bounds the opportunistic refresh that runs before an
// ordinary command. Expiring it records an automatic failure and leaves the
// command itself successful.
const automaticSyncTimeout = 30 * time.Second

type state struct {
	dbPath           string
	dbPathSource     string
	configPath       string
	configPathSource string
	json             bool
	fixture          string
	demo             bool
	out              io.Writer
	errOut           io.Writer
	runStarted       bool
	openService      OpenService
	service          Service
	// serviceOpenErr holds the failure `debug` is allowed to report instead of
	// failing on, so the command that explains a broken installation still runs.
	serviceOpenErr error
	closer         io.Closer
	closeOnce      sync.Once
	closeErr       error
	standardHelp   func(*cobra.Command, []string)
}

// closeService releases what opening the service reserved. A demo reserves a
// temporary directory, and Cobra skips its post-run hooks once a command
// returns an error, so this has to be reachable from the failing path too.
func (s *state) closeService() error {
	s.closeOnce.Do(func() {
		if s.closer != nil {
			s.closeErr = s.closer.Close()
		}
	})
	return s.closeErr
}

func NewRoot(out, errOut io.Writer, openService OpenService) *cobra.Command {
	root, _ := newRootWithState(out, errOut, openService)
	return root
}

func newRootWithState(out, errOut io.Writer, openService OpenService) (*cobra.Command, *state) {
	s := &state{out: out, errOut: errOut, openService: openService}
	root := &cobra.Command{
		Use:           "prx",
		Short:         "Manage pull-request dependency roadmaps",
		Version:       prx.Version(),
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if s.demo {
				for _, name := range []string{"db", "config", "github-fixture"} {
					if cmd.Flags().Changed(name) {
						return fmt.Errorf("--demo cannot be used with --%s", name)
					}
				}
				s.dbPath = ""
				s.configPath = ""
				s.fixture = ""
				s.dbPathSource = "demo"
				s.configPathSource = "demo"
			} else {
				s.applyEnvironmentPaths()
			}
			if cmd.Name() == "help" || cmd.Name() == "schema-version" {
				return nil
			}
			// A demo never reads the normal configuration, so warning about it
			// would report a file the demo run does not use.
			if !s.demo {
				s.warnAboutConfiguration()
			}
			if isConfigCommand(cmd) {
				return nil
			}
			baseContext := cmd.Context()
			if baseContext == nil {
				baseContext = context.Background()
			}
			openContext := config.WithPath(baseContext, s.configPath)
			live := cmd.Name() == "serve" || cmd.Name() == "sync"
			service, closer, err := s.openService(openContext, ServiceOptions{
				DatabasePath:       s.dbPath,
				DatabasePathSource: s.dbPathSource,
				ConfigPathSource:   s.configPathSource,
				FixturePath:        s.fixture,
				Live:               live,
				Demo:               s.demo,
			})
			if err != nil {
				// A broken installation is exactly when the diagnostic report is
				// needed, so `debug` keeps the failure and reports it as the
				// storage section instead of failing the command.
				if !isDebugCommand(cmd) {
					return domain.NewError(domain.DomainErrorCodeInternal, "%s", err)
				}
				s.serviceOpenErr = err
				return nil
			}
			s.service = service
			s.closer = closer
			if !s.demo && !isAutomaticSyncExcluded(cmd) && service != nil {
				// The per-request client timeout does not bound the whole run, so
				// an unreachable host would otherwise block an ordinary command
				// for as long as its credential lookups and requests take.
				syncContext, cancel := context.WithTimeout(baseContext, automaticSyncTimeout)
				_, _, _ = service.SyncIfDue(syncContext)
				cancel()
			}
			return nil
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			_ = s.closeService()
		},
	}
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVar(&s.dbPath, "db", "", "SQLite database path (env: PRX_DB)")
	root.PersistentFlags().
		StringVar(&s.configPath, "config", "", "YAML configuration path (env: PRX_CONFIG)")
	root.PersistentFlags().BoolVar(&s.json, "json", false, "output JSON")
	root.PersistentFlags().StringVar(&s.fixture, "github-fixture", "", "GitHub fixture JSON path, or demo")
	s.addCommands(root)
	s.standardHelp = root.HelpFunc()
	root.SetHelpFunc(s.writeHelp)
	root.SetHelpCommand(s.helpCommand(root))
	markCommandExecution(root, s)
	return root, s
}

// addCommands registers the resource commands first and the whole-repository
// commands second, which is the order the rendered help groups them in.
func (s *state) addCommands(root *cobra.Command) {
	root.AddCommand(
		s.schemaVersionCommand(),
		s.projectCommand(),
		s.featureCommand(),
		s.taskCommand(),
		s.showCommand(),
		s.dependencyCommand(),
		s.pullRequestCommand(),
		s.documentCommand(),
		s.planCommand(),
		s.configCommand(),
	)
	root.AddCommand(
		s.snapshotCommand(),
		s.graphCommand(),
		s.queueCommand("ready"),
		s.queueCommand("reviews"),
		s.queueCommand("conflicts"),
		s.queueCommand("stale"),
		s.syncCommand(),
		s.validateCommand(),
		s.debugCommand(),
		s.serveCommand(),
	)
}

// helpCommand replaces the default help command, which prints its own message
// and the root usage to stdout and exits successfully when the topic is
// unknown. Returning the error instead keeps every failure on the shared error
// path, so stdout stays empty and JSON callers always parse the same shape.
func (s *state) helpCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long:  "Help provides help for any command in the application.",
		RunE: func(_ *cobra.Command, args []string) error {
			target, _, err := root.Find(args)
			if err != nil {
				return err
			}
			if target == nil {
				target = root
			}
			s.writeHelp(target, args)
			return nil
		},
	}
}

// applyEnvironmentPaths resolves the environment fallbacks at run time instead
// of registering them as flag defaults, because a flag default is printed in
// the rendered help that failures now carry in their hint.
func (s *state) applyEnvironmentPaths() {
	s.dbPath, s.dbPathSource = resolvePathSource(s.dbPath, "PRX_DB")
	s.configPath, s.configPathSource = resolvePathSource(s.configPath, "PRX_CONFIG")
}

// resolvePathSource records which of the flag, the environment, or the default
// selected a location. The fallback overwrites an empty flag value, so nothing
// downstream could tell the three apart afterwards.
func resolvePathSource(value, variable string) (resolved, source string) {
	if value != "" {
		return value, "flag"
	}
	if fromEnvironment := os.Getenv(variable); fromEnvironment != "" {
		return fromEnvironment, "env"
	}
	return "", "default"
}

// warnAboutConfiguration reports the recoverable configuration problems once per
// run, before any command reads the configuration. A load failure stays silent
// here so the command that needs the configuration still owns its own error, and
// so a command that never reads it keeps succeeding.
func (s *state) warnAboutConfiguration() {
	store, err := config.NewStore(s.configPath)
	if err != nil {
		return
	}
	_, warnings, err := store.LoadWithWarnings()
	if err != nil {
		return
	}
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(s.errOut, "Warning: %s: %s\n", store.Path(), warning)
	}
}

func markCommandExecution(command *cobra.Command, s *state) {
	if command.RunE != nil {
		run := command.RunE
		command.RunE = func(cmd *cobra.Command, args []string) error {
			s.runStarted = true
			return run(cmd, args)
		}
	}
	for _, child := range command.Commands() {
		markCommandExecution(child, s)
	}
}

func isConfigCommand(command *cobra.Command) bool {
	for current := command; current != nil; current = current.Parent() {
		if current.Name() == "config" {
			return true
		}
	}
	return false
}

// isAutomaticSyncExcluded reports the commands the opportunistic refresh must
// skip. `sync` performs the refresh itself, and `debug` reports the recorded
// synchronization state, which a refresh would overwrite before the reader sees
// the failure they were asked to send.
func isAutomaticSyncExcluded(command *cobra.Command) bool {
	for current := command; current != nil; current = current.Parent() {
		if current.Name() == "sync" || current.Name() == "debug" {
			return true
		}
	}
	return false
}

func isDebugCommand(command *cobra.Command) bool {
	for current := command; current != nil; current = current.Parent() {
		if current.Name() == "debug" {
			return true
		}
	}
	return false
}

// Execute runs the CLI and formats any error according to the parsed --json
// flag. Deciding that from os.Args would miss --json=true and would misread a
// flag value that happens to be the literal string.
func Execute(
	ctx context.Context,
	args []string,
	out, errOut io.Writer,
	openService OpenService,
) (err error) {
	root, s := newRootWithState(out, errOut, openService)
	defer func() { err = errors.Join(err, s.closeService()) }()
	s.preScanOutputFlags(root, args)
	root.SetArgs(args)
	failedCommand, err := root.ExecuteContextC(ctx)
	if err == nil {
		return nil
	}
	if !s.runStarted {
		err = domainUsageError(err)
	}
	if failedCommand == nil {
		failedCommand = root
	}
	hint := s.renderHelp(failedCommand)
	if printErr := s.writeError(err, hint); printErr != nil {
		_, _ = fmt.Fprintln(errOut, "Error:", err)
	}
	return err
}

func (s *state) writeHelp(command *cobra.Command, _ []string) {
	hint := s.renderHelp(command)
	if s.json {
		_ = encodeJSON(s.out, map[string]string{"hint": hint})
		return
	}
	_, _ = io.WriteString(s.out, hint)
}

func (s *state) renderHelp(command *cobra.Command) string {
	var buffer bytes.Buffer
	previousOut := command.OutOrStdout()
	command.SetOut(&buffer)
	s.standardHelp(command, nil)
	command.SetOut(previousOut)
	return buffer.String()
}

// preScanOutputFlags preserves explicit output selection when Cobra cannot
// finish command discovery. Known value-taking flags consume the following
// argument so a literal --json value is not mistaken for a flag.
func (s *state) preScanOutputFlags(root *cobra.Command, args []string) {
	current := root
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			return
		}
		if strings.HasPrefix(arg, "--") {
			name, rawValue, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
			switch name {
			case "json":
				value := true
				if hasValue {
					parsed, err := strconv.ParseBool(rawValue)
					if err != nil {
						continue
					}
					value = parsed
				}
				s.json = value
			default:
				if !hasValue && longFlagTakesValue(current, name) && index+1 < len(args) {
					index++
				}
			}
			continue
		}
		if len(arg) == 2 && arg[0] == '-' && shortFlagTakesValue(current, string(arg[1])) && index+1 < len(args) {
			index++
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			if child := directChild(current, arg); child != nil {
				current = child
			}
		}
	}
}

func longFlagTakesValue(command *cobra.Command, name string) bool {
	flag := command.Flag(name)
	return flag != nil && flag.NoOptDefVal == ""
}

func shortFlagTakesValue(command *cobra.Command, shorthand string) bool {
	for current := command; current != nil; current = current.Parent() {
		if flag := current.LocalNonPersistentFlags().ShorthandLookup(shorthand); flag != nil {
			return flag.NoOptDefVal == ""
		}
		if flag := current.PersistentFlags().ShorthandLookup(shorthand); flag != nil {
			return flag.NoOptDefVal == ""
		}
	}
	return false
}

func directChild(command *cobra.Command, name string) *cobra.Command {
	for _, child := range command.Commands() {
		if child.Name() == name || child.HasAlias(name) {
			return child
		}
	}
	return nil
}

func domainUsageError(err error) error {
	if err == nil {
		return nil
	}
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return err
	}
	var usageErr *usageError
	if errors.As(err, &usageErr) {
		return err
	}
	return &usageError{err: err}
}

type usageError struct {
	err error
}

func (e *usageError) Error() string { return e.err.Error() }

func (e *usageError) Unwrap() error { return e.err }
