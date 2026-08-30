package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) planCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "plan TASK_ID",
		Short:   "Show or manage a task's implementation plan",
		Example: "prx plan T-1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.GetImplementationPlan(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(value, renderImplementationPlan(value))
		},
	}

	set := &cobra.Command{
		Use:   "set TASK_ID FILE_OR_DASH",
		Short: "Create or replace a task's implementation plan",
		Long: "Create or replace a task's implementation plan.\n\n" +
			"FILE_OR_DASH is a Markdown file path, or - to read the plan content from standard input.",
		Example: "prx plan set TASK_ID plan.md\nprx plan set TASK_ID -",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var content []byte
			var err error
			if args[1] == "-" {
				content, err = io.ReadAll(cmd.InOrStdin())
			} else {
				content, err = os.ReadFile(args[1])
			}
			if err != nil {
				return domain.NewError(
					domain.DomainErrorCodeInvalidImplementationPlan,
					"could not read implementation plan: %v",
					err,
				)
			}
			value, err := s.service.UpsertImplementationPlan(cmd.Context(), args[0], string(content))
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Set implementation plan for task %s.", value.TaskID))
		},
	}

	deleteCmd := &cobra.Command{
		Use:     "delete TASK_ID",
		Short:   "Delete a task's implementation plan",
		Example: "prx plan delete TASK_ID",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteImplementationPlan(cmd.Context(), args[0]); err != nil {
				return err
			}
			return s.write(
				map[string]string{"deleted": args[0]},
				renderMessage("Deleted implementation plan for task %s.", args[0]),
			)
		},
	}

	command.AddCommand(set, deleteCmd)
	return command
}
