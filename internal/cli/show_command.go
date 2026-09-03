package cli

import (
	"github.com/spf13/cobra"
)

func (s *state) showCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show PROJECT_OR_FEATURE_OR_TASK",
		Short: "Show a project, a feature, or a task by public identifier",
		Long: "Show a project, a feature, or a task by public identifier.\n\n" +
			"The operand is a public project, feature, or task ID, or a feature or project slug.",
		Example: "prx show F-1\nprx show checkout\nprx show T-1\nprx show P-1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.GetNode(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(value, renderNode(value))
		},
	}
}
