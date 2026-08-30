package cli

import "github.com/spf13/cobra"

func (s *state) dependencyCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "dependency",
		Aliases: []string{"dep"},
		Short:   "List or manage directed blocker edges",
		Long:    "List or manage directed blocker edges.\n\nAlias: dep.",
		Example: "prx dependency --json\nprx dep --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			dependencies := nonNilSlice(snapshot.Dependencies)
			return s.write(map[string]any{"dependencies": dependencies}, renderDependencyList(dependencies))
		},
	}
	add := &cobra.Command{
		Use:     "add BLOCKER_TASK_ID BLOCKED_TASK_ID",
		Short:   "Add a blocker-to-blocked dependency",
		Example: "prx dependency add BLOCKER_TASK_ID BLOCKED_TASK_ID --json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.AddDependency(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Added dependency %s -> %s.", value.BlockerTaskID, value.BlockedTaskID))
		},
	}
	remove := &cobra.Command{
		Use:     "remove BLOCKER_TASK_ID BLOCKED_TASK_ID",
		Short:   "Remove a dependency; missing edges return not_found",
		Example: "prx dependency remove BLOCKER_TASK_ID BLOCKED_TASK_ID --json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.RemoveDependency(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			return s.write(map[string]string{"removed": args[0] + "->" + args[1]},
				renderMessage("Removed dependency %s -> %s.", args[0], args[1]))
		},
	}
	command.AddCommand(add, remove)
	return command
}
