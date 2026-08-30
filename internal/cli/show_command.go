package cli

import (
	"github.com/spf13/cobra"
)

func (s *state) showCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "show FEATURE_ID_OR_SLUG_OR_TASK_ID",
		Short:   "Show a feature or task by public identifier",
		Example: "prx show F-1 --json\nprx show checkout --json\nprx show T-1 --json",
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
