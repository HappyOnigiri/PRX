package cli

import (
	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) taskCommand() *cobra.Command {
	command := &cobra.Command{Use: "task", Short: "Manage implementation and manual tasks"}
	var feature, title, scope, kind, assignee, status string
	var cascade bool
	create := &cobra.Command{
		Use:     "create",
		Short:   "Create an implementation or manual task",
		Example: "prx task create --feature checkout --title \"Add payment intent API\" --assignee Mika --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			value, err := s.service.CreateTask(cmd.Context(), feature, title, scope, domain.TaskKind(kind), assignee)
			if err != nil {
				return err
			}
			return s.write(value)
		},
	}
	create.Flags().StringVar(&feature, "feature", "", "feature ID or slug")
	create.Flags().StringVar(&title, "title", "", "task title")
	create.Flags().StringVar(&scope, "scope", "", "scope description")
	create.Flags().StringVar(&kind, "kind", "pr", "pr or manual")
	create.Flags().StringVar(&assignee, "assignee", "", "assignee")
	_ = create.MarkFlagRequired("feature")
	_ = create.MarkFlagRequired("title")
	list := &cobra.Command{
		Use:     "list",
		Short:   "List tasks, optionally filtered by feature",
		Example: "prx task list --feature checkout --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
		},
	}
	list.Flags().StringVar(&feature, "feature", "", "filter by feature")
	get := &cobra.Command{
		Use:     "get TASK_ID",
		Short:   "Show a task by ID",
		Example: "prx task get T-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			for _, task := range snapshot.Tasks {
				if task.ID == args[0] {
					return s.write(task)
				}
			}
			return domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", args[0])
		},
	}
	update := &cobra.Command{
		Use:     "update TASK_ID",
		Short:   "Update a task by ID",
		Example: "prx task update TASK_ID --status completed --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.UpdateTask(cmd.Context(), args[0],
				changedFlag(cmd, "title", &title), changedFlag(cmd, "scope", &scope),
				changedStringType[domain.TaskStatus](cmd, "status", &status), changedFlag(cmd, "assignee", &assignee))
			if err != nil {
				return err
			}
			return s.write(value)
		},
	}
	update.Flags().StringVar(&title, "title", "", "new title")
	update.Flags().StringVar(&scope, "scope", "", "new scope")
	update.Flags().StringVar(&status, "status", "", "planned, in_progress, completed, or cancelled")
	update.Flags().StringVar(&assignee, "assignee", "", "new assignee")
	deleteCmd := &cobra.Command{
		Use:     "delete TASK_ID",
		Short:   "Delete a task and optionally its dependencies and references",
		Example: "prx task delete TASK_ID --cascade --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteTask(cmd.Context(), args[0], cascade); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]})
		},
	}
	deleteCmd.Flags().BoolVar(&cascade, "cascade", false, "delete dependencies and references")
	command.AddCommand(create, list, get, update, deleteCmd)
	return command
}
