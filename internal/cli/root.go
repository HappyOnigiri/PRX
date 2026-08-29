package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/HappyOnigiri/PRX/internal/app"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
	"github.com/spf13/cobra"
)

type state struct {
	dbPath  string
	json    bool
	fixture string
	out     io.Writer
	errOut  io.Writer
	store   *store.Store
	service Service
}

func newRootWithState(out, errOut io.Writer) (*cobra.Command, *state) {
	s := &state{out: out, errOut: errOut}
	root := &cobra.Command{
		Use: "prx", Short: "Manage pull-request dependency roadmaps", SilenceErrors: true, SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == "help" {
				return nil
			}
			database, err := store.Open(cmd.Context(), s.dbPath)
			if err != nil {
				return err
			}
			s.store = database
			var provider githubprovider.Provider
			if s.fixture != "" {
				provider, err = githubprovider.NewFixtureProvider(s.fixture)
				if err != nil {
					_ = database.Close()
					return err
				}
			}
			s.service = app.New(database, provider)
			return nil
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			if s.store != nil {
				_ = s.store.Close()
			}
		},
	}
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVar(&s.dbPath, "db", os.Getenv("PRX_DB"), "SQLite database path (env: PRX_DB)")
	root.PersistentFlags().BoolVar(&s.json, "json", false, "emit a stable JSON envelope")
	root.PersistentFlags().StringVar(&s.fixture, "github-fixture", "", "GitHub fixture JSON path, or demo")
	root.AddCommand(s.featureCommand(), s.taskCommand(), s.dependencyCommand(), s.pullRequestCommand(), s.documentCommand())
	root.AddCommand(s.snapshotCommand(), s.graphCommand(), s.queueCommand("ready"), s.queueCommand("reviews"), s.queueCommand("conflicts"), s.queueCommand("stale"), s.syncCommand(), s.validateCommand(), s.seedCommand(), s.serveCommand())
	return root, s
}

// Execute runs the CLI and formats any error according to the parsed --json
// flag. Deciding that from os.Args would miss --json=true and would misread a
// flag value that happens to be the literal string.
func Execute(ctx context.Context, args []string, out, errOut io.Writer) error {
	root, s := newRootWithState(out, errOut)
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
