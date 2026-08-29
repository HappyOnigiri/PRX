package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/spf13/cobra"
)

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

func filterTasks(tasks []domain.Task, keep func(domain.Task) bool) []domain.Task {
	result := make([]domain.Task, 0)
	for _, task := range tasks {
		if keep(task) {
			result = append(result, task)
		}
	}
	return result
}
