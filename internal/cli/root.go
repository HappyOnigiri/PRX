package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/config"
)

type state struct {
	dbPath      string
	configPath  string
	json        bool
	fixture     string
	out         io.Writer
	errOut      io.Writer
	openService OpenService
	service     Service
	closer      io.Closer
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
			if cmd.Name() == "help" || isConfigCommand(cmd) {
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
				return err
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
	root.PersistentFlags().BoolVar(&s.json, "json", false, "emit a stable JSON envelope")
	root.PersistentFlags().StringVar(&s.fixture, "github-fixture", "", "GitHub fixture JSON path, or demo")
	root.AddCommand(
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
	return root, s
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
	if s.json {
		if printErr := PrintError(out, err); printErr != nil {
			_, _ = fmt.Fprintln(errOut, "error:", err)
		}
	} else {
		_, _ = fmt.Fprintln(errOut, "error:", err)
	}
	return err
}
