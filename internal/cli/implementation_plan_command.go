package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) implementationPlanCommand() *cobra.Command {
	command := &cobra.Command{Use: "implementation-plan", Short: "Manage task implementation plans"}
	get := &cobra.Command{
		Use:     "get TASK_ID",
		Short:   "Show a task's implementation plan",
		Example: "prx implementation-plan get TASK_ID --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.GetImplementationPlan(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(value)
		},
	}

	var file string
	var stdin bool
	set := &cobra.Command{
		Use:     "set TASK_ID",
		Short:   "Create or replace a task's implementation plan",
		Example: "prx implementation-plan set TASK_ID --file plan.md --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if (file == "") == !stdin {
				return domain.NewError(
					domain.DomainErrorCodeInvalidImplementationPlan,
					"specify exactly one of --file or --stdin",
				)
			}
			var content []byte
			var err error
			if stdin {
				content, err = io.ReadAll(cmd.InOrStdin())
			} else {
				content, err = os.ReadFile(file)
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
			return s.write(value)
		},
	}
	set.Flags().StringVar(&file, "file", "", "read plan content from a file")
	set.Flags().BoolVar(&stdin, "stdin", false, "read plan content from standard input")

	deleteCmd := &cobra.Command{
		Use:     "delete TASK_ID",
		Short:   "Delete a task's implementation plan",
		Example: "prx implementation-plan delete TASK_ID --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteImplementationPlan(cmd.Context(), args[0]); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]})
		},
	}

	command.AddCommand(get, set, deleteCmd)
	return command
}
