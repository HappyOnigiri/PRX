package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) taskCommand() *cobra.Command {
	var filterFeature string
	command := &cobra.Command{
		Use:     "task [TASK_ID]",
		Aliases: []string{"t"},
		Short:   "List tasks or show one by ID",
		Long:    "List tasks or show one by ID.\n\nAlias: t.",
		Example: "prx task\nprx task --feature checkout\nprx task T-1\nprx t T-1",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
				return err
			}
			if len(args) == 1 && cmd.Flags().Changed("feature") {
				return fmt.Errorf("--feature cannot be used with TASK_ID")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			if len(args) == 1 {
				for _, task := range snapshot.Tasks {
					if task.ID == args[0] {
						return s.write(task, renderTaskDetailWithAppearances(task, featureAppearances(snapshot)))
					}
				}
				return domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", args[0])
			}
			tasks := snapshot.Tasks
			if filterFeature != "" {
				feature, err := s.service.ResolveFeature(cmd.Context(), filterFeature)
				if err != nil {
					return err
				}
				tasks = filterTasks(tasks, func(task domain.Task) bool { return task.FeatureID == feature.ID })
			}
			tasks = nonNilSlice(tasks)
			return s.write(
				map[string]any{"tasks": tasks},
				renderTaskListWithAppearances(tasks, featureAppearances(snapshot)),
			)
		},
	}
	command.Flags().StringVar(&filterFeature, "feature", "", "filter by feature ID")
	command.AddCommand(s.taskCreateCommand(), s.taskUpdateCommand(), s.taskDeleteCommand())
	return command
}

func (s *state) taskCreateCommand() *cobra.Command {
	var scope, assignee string
	command := &cobra.Command{
		Use:   "create FEATURE_ID TITLE",
		Short: "Create a task",
		Example: "prx task create F-1 \"Add payment intent API\" --assignee Bob\n" +
			"prx task create F-1 -- \"-fix login redirect\"",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.CreateTask(cmd.Context(), args[0], args[1], scope, assignee)
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Created task %s.", value.ID))
		},
	}
	command.Flags().StringVar(&scope, "scope", "", "scope description")
	command.Flags().StringVar(&assignee, "assignee", "", "assignee")
	return command
}

func (s *state) taskUpdateCommand() *cobra.Command {
	var title, scope, status, assignee string
	command := &cobra.Command{
		Use:     "update TASK_ID",
		Short:   "Update a task by ID",
		Example: "prx task update TASK_ID --status completed",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var statusValue *domain.TaskStatus
			if cmd.Flags().Changed("status") {
				snapshot, err := s.service.Snapshot(cmd.Context())
				if err != nil {
					return err
				}
				statusValue, err = resolveTaskStatusInput(status, args[0], snapshot)
				if err != nil {
					return err
				}
			}
			value, err := s.service.UpdateTask(cmd.Context(), args[0],
				changedFlag(cmd, "title", &title), changedFlag(cmd, "scope", &scope),
				statusValue, changedFlag(cmd, "assignee", &assignee))
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("%s", taskUpdateMessage(cmd, value.ID, title, scope, status, assignee)))
		},
	}
	command.Flags().StringVar(&title, "title", "", "new title")
	command.Flags().StringVar(&scope, "scope", "", "new scope")
	command.Flags().StringVar(&status, "status", "", "not_started, designing, in_progress, completed, or closed")
	command.Flags().StringVar(&assignee, "assignee", "", "new assignee")
	return command
}

func resolveTaskStatusInput(raw, taskID string, snapshot domain.Snapshot) (*domain.TaskStatus, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	stable := map[string]domain.TaskStatus{
		string(domain.TaskStatusNotStarted): domain.TaskStatusNotStarted,
		string(domain.TaskStatusDesigning):  domain.TaskStatusDesigning,
		string(domain.TaskStatusInProgress): domain.TaskStatusInProgress,
		string(domain.TaskStatusCompleted):  domain.TaskStatusCompleted,
		string(domain.TaskStatusClosed):     domain.TaskStatusClosed,
	}
	if status, ok := stable[value]; ok {
		return &status, nil
	}
	featureID := ""
	for _, task := range snapshot.Tasks {
		if task.ID == taskID {
			featureID = task.FeatureID
			break
		}
	}
	if featureID == "" {
		return nil, domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", taskID)
	}
	var matches []domain.TaskStatus
	for _, candidate := range []struct {
		key    domain.TaskLabelKey
		status domain.TaskStatus
	}{
		{domain.TaskLabelStatusNotStarted, domain.TaskStatusNotStarted},
		{domain.TaskLabelStatusDesigning, domain.TaskStatusDesigning},
		{domain.TaskLabelStatusInProgress, domain.TaskStatusInProgress},
		{domain.TaskLabelStatusCompleted, domain.TaskStatusCompleted},
		{domain.TaskLabelStatusClosed, domain.TaskStatusClosed},
	} {
		for _, feature := range snapshot.Features {
			if feature.ID != featureID {
				continue
			}
			if appearance, ok := feature.TaskLabelAppearances[candidate.key]; ok &&
				strings.EqualFold(strings.TrimSpace(appearance.Text), strings.TrimSpace(raw)) {
				matches = append(matches, candidate.status)
			}
		}
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}
	if len(matches) > 1 {
		values := make([]string, len(matches))
		for i, match := range matches {
			values[i] = string(match)
		}
		return nil, domain.NewError(domain.DomainErrorCodeInvalidStatus,
			"status label %q is ambiguous; use one of: %s", raw, strings.Join(values, ", "))
	}
	return nil, domain.NewError(domain.DomainErrorCodeInvalidStatus,
		"invalid task status %q; use not_started, designing, in_progress, completed, or closed", raw)
}

func (s *state) taskDeleteCommand() *cobra.Command {
	var cascade bool
	command := &cobra.Command{
		Use:     "delete TASK_ID",
		Short:   "Delete a task and optionally its dependencies and references",
		Example: "prx task delete TASK_ID --cascade",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteTask(cmd.Context(), args[0], cascade); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]}, renderMessage("Deleted task %s.", args[0]))
		},
	}
	command.Flags().BoolVar(&cascade, "cascade", false, "delete dependencies and references")
	return command
}

func taskUpdateMessage(cmd *cobra.Command, taskID, title, scope, status, assignee string) string {
	changes := make([]string, 0, 4)
	for _, field := range []struct {
		name  string
		value string
	}{{"title", title}, {"scope", scope}, {"status", status}, {"assignee", assignee}} {
		if cmd.Flags().Changed(field.name) {
			changes = append(changes, fmt.Sprintf("%s=%s", field.name, field.value))
		}
	}
	if len(changes) == 0 {
		return fmt.Sprintf("Updated task %s.", taskID)
	}
	return fmt.Sprintf("Updated task %s: %s.", taskID, strings.Join(changes, ", "))
}
