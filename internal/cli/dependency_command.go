package cli

import "github.com/spf13/cobra"

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
