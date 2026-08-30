package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

type state struct {
	dbPath      string
	configPath  string
	json        bool
	human       bool
	fixture     string
	out         io.Writer
	errOut      io.Writer
	isTerminal  func(io.Writer) bool
	mode        outputMode
	modeSet     bool
	runStarted  bool
	openService OpenService
	service     Service
	closer      io.Closer
}

func NewRoot(out, errOut io.Writer, openService OpenService) *cobra.Command {
	root, _ := newRootWithState(out, errOut, openService)
	return root
}

func newRootWithState(out, errOut io.Writer, openService OpenService) (*cobra.Command, *state) {
	s := &state{out: out, errOut: errOut, openService: openService, isTerminal: writerIsTerminal}
	root := &cobra.Command{
		Use:           "prx",
		Short:         "Manage pull-request dependency roadmaps",
		Version:       prx.Version(),
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := s.resolveOutputMode(); err != nil {
				return err
			}
			if cmd.Name() == "help" || cmd.Name() == "schema-version" || isConfigCommand(cmd) {
				return nil
			}
			baseContext := cmd.Context()
			if baseContext == nil {
				baseContext = context.Background()
			}
			openContext := config.WithPath(baseContext, s.configPath)
			live := cmd.Name() == "serve" || cmd.Name() == "sync"
			service, closer, err := s.openService(openContext, s.dbPath, s.fixture, live)
			if err != nil {
				return domain.NewError(domain.DomainErrorCodeInternal, "%s", err)
			}
			s.service = service
			s.closer = closer
			return nil
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			if s.closer != nil {
				_ = s.closer.Close()
			}
		},
	}
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVar(&s.dbPath, "db", os.Getenv("PRX_DB"), "SQLite database path (env: PRX_DB)")
	root.PersistentFlags().
		StringVar(&s.configPath, "config", os.Getenv("PRX_CONFIG"), "YAML configuration path (env: PRX_CONFIG)")
	root.PersistentFlags().BoolVar(&s.json, "json", false, "force compact JSON responses")
	root.PersistentFlags().BoolVar(&s.human, "human", false, "force human-readable responses")
	root.PersistentFlags().StringVar(&s.fixture, "github-fixture", "", "GitHub fixture JSON path, or demo")
	root.AddCommand(
		s.schemaVersionCommand(),
		s.featureCommand(),
		s.taskCommand(),
		s.nodeCommand(),
		s.dependencyCommand(),
		s.pullRequestCommand(),
		s.documentCommand(),
		s.implementationPlanCommand(),
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
		s.seedCommand(),
		s.serveCommand(),
	)
	markCommandExecution(root, s)
	return root, s
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

// Execute runs the CLI and formats any error according to the parsed --json
// flag. Deciding that from os.Args would miss --json=true and would misread a
// flag value that happens to be the literal string.
func Execute(ctx context.Context, args []string, out, errOut io.Writer, openService OpenService) error {
	root, s := newRootWithState(out, errOut, openService)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return nil
	}
	if !s.runStarted {
		err = domainUsageError(err)
	}
	if printErr := s.writeError(err); printErr != nil {
		_, _ = fmt.Fprintln(errOut, "Error:", err)
	}
	return err
}

func (s *state) resolveOutputMode() error {
	if s.modeSet {
		if s.json && s.human {
			return domainUsageError(errors.New("--json and --human cannot be used together"))
		}
		return nil
	}
	s.modeSet = true
	switch {
	case s.json && s.human:
		if s.isTerminal != nil && s.isTerminal(s.out) {
			s.mode = outputModeHuman
		} else {
			s.mode = outputModeJSON
		}
		return domainUsageError(errors.New("--json and --human cannot be used together"))
	case s.json:
		s.mode = outputModeJSON
	case s.human:
		s.mode = outputModeHuman
	case s.isTerminal != nil && s.isTerminal(s.out):
		s.mode = outputModeHuman
	default:
		s.mode = outputModeJSON
	}
	return nil
}

func writerIsTerminal(out io.Writer) bool {
	file, ok := out.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(int(file.Fd()))
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
