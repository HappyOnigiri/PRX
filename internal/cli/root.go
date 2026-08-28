package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/store"
	"github.com/HappyOnigiri/PRX/internal/webui"
	"github.com/spf13/cobra"
)

type state struct {
	dbPath  string
	json    bool
	fixture string
	out     io.Writer
	errOut  io.Writer
	store   *store.Store
	service *app.Service
}

type envelope struct {
	SchemaVersion string        `json:"schema_version"`
	OK            bool          `json:"ok"`
	Data          any           `json:"data,omitempty"`
	Error         *domain.Error `json:"error,omitempty"`
}

func NewRoot(out, errOut io.Writer) *cobra.Command {
	root, _ := newRootWithState(out, errOut)
	return root
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
		PrintError(out, err)
	} else {
		_, _ = fmt.Fprintln(errOut, "error:", err)
	}
	return err
}

func PrintError(out io.Writer, err error) {
	value := &domain.Error{Code: domain.ErrorCode(err), Message: err.Error()}
	var typed *domain.Error
	if errors.As(err, &typed) {
		value = typed
	}
	_ = json.NewEncoder(out).Encode(envelope{SchemaVersion: "1", OK: false, Error: value})
}

func (s *state) write(value any) error {
	encoder := json.NewEncoder(s.out)
	if !s.json {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(envelope{SchemaVersion: "1", OK: true, Data: value})
}

func (s *state) ensureLiveProvider(ctx context.Context) error {
	if s.fixture != "" {
		return nil
	}
	provider, err := githubprovider.NewLiveProvider(ctx)
	if err != nil {
		return &domain.Error{Code: "github_auth", Message: err.Error()}
	}
	s.service = app.New(s.store, provider)
	return nil
}

func (s *state) featureCommand() *cobra.Command {
	command := &cobra.Command{Use: "feature", Short: "Manage features"}
	var slug, title, description, status string
	var archived, cascade bool
	create := &cobra.Command{Use: "create", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		value, err := s.service.CreateFeature(cmd.Context(), slug, title, description)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	create.Flags().StringVar(&slug, "slug", "", "stable feature slug")
	create.Flags().StringVar(&title, "title", "", "feature title")
	create.Flags().StringVar(&description, "description", "", "feature description")
	_ = create.MarkFlagRequired("slug")
	_ = create.MarkFlagRequired("title")
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		value, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		return s.write(value.Features)
	}}
	get := &cobra.Command{Use: "get ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := s.service.ResolveFeature(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	update := &cobra.Command{Use: "update ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := s.service.UpdateFeature(cmd.Context(), args[0],
			changedFlag(cmd, "slug", &slug), changedFlag(cmd, "title", &title),
			changedFlag(cmd, "description", &description), changedFlag(cmd, "status", &status),
			changedBoolFlag(cmd, "archived", &archived))
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	update.Flags().StringVar(&slug, "slug", "", "new slug")
	update.Flags().StringVar(&title, "title", "", "new title")
	update.Flags().StringVar(&description, "description", "", "new description")
	update.Flags().StringVar(&status, "status", "", "active, paused, completed, or cancelled")
	update.Flags().BoolVar(&archived, "archived", false, "archive (true) or unarchive (false) the feature")
	archive := &cobra.Command{Use: "archive ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		archive := true
		value, err := s.service.UpdateFeature(cmd.Context(), args[0], nil, nil, nil, nil, &archive)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	unarchive := &cobra.Command{Use: "unarchive ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		archive := false
		value, err := s.service.UpdateFeature(cmd.Context(), args[0], nil, nil, nil, nil, &archive)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	deleteCmd := &cobra.Command{Use: "delete ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := s.service.DeleteFeature(cmd.Context(), args[0], cascade); err != nil {
			return err
		}
		return s.write(map[string]string{"deleted": args[0]})
	}}
	deleteCmd.Flags().BoolVar(&cascade, "cascade", false, "delete contained tasks and references")
	command.AddCommand(create, list, get, update, archive, unarchive, deleteCmd)
	return command
}

func (s *state) taskCommand() *cobra.Command {
	command := &cobra.Command{Use: "task", Short: "Manage implementation and manual tasks"}
	var feature, title, scope, kind, assignee, status string
	var cascade bool
	create := &cobra.Command{Use: "create", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		value, err := s.service.CreateTask(cmd.Context(), feature, title, scope, kind, assignee)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	create.Flags().StringVar(&feature, "feature", "", "feature ID or slug")
	create.Flags().StringVar(&title, "title", "", "task title")
	create.Flags().StringVar(&scope, "scope", "", "scope description")
	create.Flags().StringVar(&kind, "kind", "pr", "pr or manual")
	create.Flags().StringVar(&assignee, "assignee", "", "assignee")
	_ = create.MarkFlagRequired("feature")
	_ = create.MarkFlagRequired("title")
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		snapshot, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		tasks := snapshot.Tasks
		if feature != "" {
			f, err := s.service.ResolveFeature(cmd.Context(), feature)
			if err != nil {
				return err
			}
			tasks = filterTasks(tasks, func(task domain.Task) bool { return task.FeatureID == f.ID })
		}
		return s.write(tasks)
	}}
	list.Flags().StringVar(&feature, "feature", "", "filter by feature")
	get := &cobra.Command{Use: "get ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		snapshot, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		for _, task := range snapshot.Tasks {
			if task.ID == args[0] {
				return s.write(task)
			}
		}
		return domain.NewError("not_found", "task %q was not found", args[0])
	}}
	update := &cobra.Command{Use: "update ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := s.service.UpdateTask(cmd.Context(), args[0],
			changedFlag(cmd, "title", &title), changedFlag(cmd, "scope", &scope),
			changedFlag(cmd, "status", &status), changedFlag(cmd, "assignee", &assignee))
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	update.Flags().StringVar(&title, "title", "", "new title")
	update.Flags().StringVar(&scope, "scope", "", "new scope")
	update.Flags().StringVar(&status, "status", "", "planned, in_progress, completed, or cancelled")
	update.Flags().StringVar(&assignee, "assignee", "", "new assignee")
	deleteCmd := &cobra.Command{Use: "delete ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := s.service.DeleteTask(cmd.Context(), args[0], cascade); err != nil {
			return err
		}
		return s.write(map[string]string{"deleted": args[0]})
	}}
	deleteCmd.Flags().BoolVar(&cascade, "cascade", false, "delete dependencies and references")
	command.AddCommand(create, list, get, update, deleteCmd)
	return command
}

func (s *state) dependencyCommand() *cobra.Command {
	command := &cobra.Command{Use: "dependency", Short: "Manage directed blocker edges"}
	add := &cobra.Command{Use: "add BLOCKER BLOCKED", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := s.service.AddDependency(cmd.Context(), args[0], args[1])
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	remove := &cobra.Command{Use: "remove BLOCKER BLOCKED", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		if err := s.service.RemoveDependency(cmd.Context(), args[0], args[1]); err != nil {
			return err
		}
		return s.write(map[string]string{"removed": args[0] + "->" + args[1]})
	}}
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		snapshot, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		return s.write(snapshot.Dependencies)
	}}
	command.AddCommand(add, remove, list)
	return command
}

func (s *state) pullRequestCommand() *cobra.Command {
	command := &cobra.Command{Use: "pr", Short: "Attach GitHub pull requests"}
	var task, url string
	attach := &cobra.Command{Use: "attach", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		value, err := s.service.AttachPullRequest(cmd.Context(), task, url)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	attach.Flags().StringVar(&task, "task", "", "task ID")
	attach.Flags().StringVar(&url, "url", "", "GitHub pull request URL")
	_ = attach.MarkFlagRequired("task")
	_ = attach.MarkFlagRequired("url")
	detach := &cobra.Command{Use: "detach TASK", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := s.service.DetachPullRequest(cmd.Context(), args[0]); err != nil {
			return err
		}
		return s.write(map[string]string{"detached": args[0]})
	}}
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		snapshot, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		return s.write(snapshot.PullRequests)
	}}
	command.AddCommand(attach, detach, list)
	return command
}

func (s *state) documentCommand() *cobra.Command {
	command := &cobra.Command{Use: "document", Short: "Manage URL and local Markdown references"}
	var feature, task, kind, title, value string
	add := &cobra.Command{Use: "add", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		doc, err := s.service.AddDocument(cmd.Context(), feature, task, kind, title, value)
		if err != nil {
			return err
		}
		return s.write(doc)
	}}
	add.Flags().StringVar(&feature, "feature", "", "feature ID or slug")
	add.Flags().StringVar(&task, "task", "", "task ID")
	add.Flags().StringVar(&kind, "kind", "url", "url or markdown_path")
	add.Flags().StringVar(&title, "title", "", "document title")
	add.Flags().StringVar(&value, "value", "", "URL or Markdown path")
	_ = add.MarkFlagRequired("value")
	deleteCmd := &cobra.Command{Use: "delete ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := s.service.DeleteDocument(cmd.Context(), args[0]); err != nil {
			return err
		}
		return s.write(map[string]string{"deleted": args[0]})
	}}
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		snapshot, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		return s.write(snapshot.Documents)
	}}
	command.AddCommand(add, deleteCmd, list)
	return command
}

func (s *state) snapshotCommand() *cobra.Command {
	return &cobra.Command{Use: "snapshot", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		value, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		return s.write(value)
	}}
}

func (s *state) graphCommand() *cobra.Command {
	return &cobra.Command{Use: "graph FEATURE", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		feature, err := s.service.ResolveFeature(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		snapshot, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		tasks := filterTasks(snapshot.Tasks, func(task domain.Task) bool { return task.FeatureID == feature.ID })
		ids := map[string]bool{}
		for _, task := range tasks {
			ids[task.ID] = true
		}
		deps := make([]domain.Dependency, 0)
		for _, dep := range snapshot.Dependencies {
			if ids[dep.BlockerTaskID] {
				deps = append(deps, dep)
			}
		}
		return s.write(map[string]any{"feature": feature, "tasks": tasks, "dependencies": deps})
	}}
}

func (s *state) queueCommand(name string) *cobra.Command {
	return &cobra.Command{Use: name, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		snapshot, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		var tasks []domain.Task
		switch name {
		case "ready":
			tasks = snapshot.ReadyTasks
		case "reviews":
			tasks = snapshot.ReviewWaitingTasks
		case "conflicts":
			tasks = snapshot.ConflictTasks
		case "stale":
			tasks = snapshot.StaleTasks
		}
		return s.write(tasks)
	}}
}

func (s *state) syncCommand() *cobra.Command {
	var feature, task string
	command := &cobra.Command{Use: "sync", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := s.ensureLiveProvider(cmd.Context()); err != nil {
			return err
		}
		succeeded, failed, err := s.service.Sync(cmd.Context(), feature, task)
		if err != nil {
			return err
		}
		return s.write(map[string]int{"succeeded": succeeded, "failed": failed})
	}}
	command.Flags().StringVar(&feature, "feature", "", "feature ID or slug")
	command.Flags().StringVar(&task, "task", "", "task ID")
	return command
}

func (s *state) validateCommand() *cobra.Command {
	return &cobra.Command{Use: "validate", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		items := s.service.Validate(cmd.Context())
		if len(items) > 0 {
			return domain.NewError("invalid_database", "database validation failed: %s", strings.Join(items, "; "))
		}
		return s.write(map[string]bool{"valid": true})
	}}
}

func (s *state) seedCommand() *cobra.Command {
	var count int
	var slug string
	var features int
	command := &cobra.Command{Use: "seed", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if features < 1 {
			return domain.NewError("invalid_seed", "features must be at least 1")
		}
		if s.fixture == "" {
			provider, _ := githubprovider.NewFixtureProvider("demo")
			s.service = app.New(s.store, provider)
		}
		for index := 0; index < features; index++ {
			featureSlug := slug
			if features > 1 {
				featureSlug = fmt.Sprintf("%s-%03d", slug, index+1)
			}
			if err := s.service.SeedDemo(cmd.Context(), featureSlug, count); err != nil {
				return err
			}
		}
		snapshot, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		return s.write(snapshot)
	}}
	command.Flags().IntVar(&count, "tasks", 8, "number of demo tasks")
	command.Flags().StringVar(&slug, "slug", "demo-roadmap", "feature slug")
	command.Flags().IntVar(&features, "features", 1, "number of demo features")
	return command
}

func (s *state) serveCommand() *cobra.Command {
	var address string
	command := &cobra.Command{Use: "serve", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if s.fixture == "" {
			if provider, err := githubprovider.NewLiveProvider(cmd.Context()); err == nil {
				s.service = app.New(s.store, provider)
			} else {
				_, _ = fmt.Fprintf(s.errOut, "warning: %v\n", err)
			}
		}
		rpcPath, rpcHandler := rpc.New(s.service)
		mux := http.NewServeMux()
		mux.Handle(rpcPath, rpcHandler)
		mux.Handle("/", webui.Handler())
		listener, err := (&net.ListenConfig{}).Listen(cmd.Context(), "tcp", address)
		if err != nil {
			return err
		}
		server := &http.Server{Addr: address, Handler: localOnly(listener.Addr(), mux), ReadHeaderTimeout: 5 * time.Second}
		_, _ = fmt.Fprintf(s.errOut, "PRX listening on http://%s\n", listener.Addr())
		go func() {
			<-cmd.Context().Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		}()
		err = server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}}
	command.Flags().StringVar(&address, "addr", "127.0.0.1:7331", "listen address")
	return command
}

// localOnly rejects requests whose Host or Origin header does not belong to the
// address the server listens on. Without it a page on an attacker-controlled
// domain that resolves to the loopback address is same-origin from the
// browser's point of view and can drive every mutation on the local database.
func localOnly(addr net.Addr, next http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	if host, port, err := net.SplitHostPort(addr.String()); err == nil {
		for _, name := range []string{host, "127.0.0.1", "localhost", "::1"} {
			allowed[strings.ToLower(net.JoinHostPort(name, port))] = struct{}{}
		}
	}
	permitted := func(hostPort string) bool {
		_, ok := allowed[strings.ToLower(hostPort)]
		return ok
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !permitted(r.Host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || !permitted(parsed.Host) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// changedFlag returns the flag value only when it was given on the command line,
// so an omitted flag leaves the field untouched while --flag "" clears it.
func changedFlag(cmd *cobra.Command, name string, value *string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return value
}

func changedBoolFlag(cmd *cobra.Command, name string, value *bool) *bool {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return value
}

func filterTasks(tasks []domain.Task, keep func(domain.Task) bool) []domain.Task {
	result := make([]domain.Task, 0)
	for _, task := range tasks {
		if keep(task) {
			result = append(result, task)
		}
	}
	return result
}
