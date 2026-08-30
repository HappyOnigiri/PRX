package cli

import (
	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) planCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "plan TASK_ID",
		Short:   "Show or manage a task's implementation plan document",
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

	var file, url, localFile string
	var stdin bool
	set := &cobra.Command{
		Use:     "set TASK_ID",
		Short:   "Create or replace a task's implementation plan document",
		Example: "prx plan set T-1 --file plan.md\nprx plan set T-1 --url https://example.com/plan",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := documentSourceFlags{url: url, localFile: localFile, markdownFile: file, stdin: stdin}
			document, err := readDocumentSource(cmd, source, domain.DomainErrorCodeInvalidImplementationPlan)
			if err != nil {
				return err
			}
			value, err := s.service.UpsertImplementationPlan(cmd.Context(), args[0], document)
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Set implementation plan for task %s.", value.TaskID))
		},
	}
	set.Flags().StringVar(&file, "file", "", "read stored Markdown content from a file")
	set.Flags().BoolVar(&stdin, "stdin", false, "read stored Markdown content from standard input")
	set.Flags().StringVar(&url, "url", "", "store an HTTP or HTTPS plan URL")
	set.Flags().StringVar(&localFile, "local-file", "", "store a local plan path")

	deleteCmd := &cobra.Command{
		Use:     "delete TASK_ID",
		Short:   "Delete a task's implementation plan document",
		Example: "prx plan delete T-1",
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
