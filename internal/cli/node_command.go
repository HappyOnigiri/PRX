package cli

import (
	"github.com/spf13/cobra"
)

func (s *state) nodeCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "node",
		Short: "Inspect feature and task nodes",
	}
	command.AddCommand(&cobra.Command{
		Use:     "get NODE_ID",
		Short:   "Show a feature or task by its public ID",
		Example: "prx node get F-1 --json\nprx node get T-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.GetNode(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(value)
		},
	})
	return command
}
