package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) showCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show PROJECT_OR_FEATURE_OR_TASK",
		Short: "Show a project, a feature, or a task by public identifier",
		Long: "Show a project, a feature, or a task by public identifier.\n\n" +
			"The operand is a public project, feature, or task ID.",
		Example: "prx show F-1\nprx show T-1\nprx show P-1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && strings.HasPrefix(args[0], "T-") {
				snapshot, err := s.service.Snapshot(cmd.Context())
				if err != nil {
					return err
				}
				for _, task := range snapshot.Tasks {
					if task.ID == args[0] {
						return s.write(task, renderTaskDetailWithAppearances(task, featureAppearances(snapshot)))
					}
				}
				return domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", args[0])
			}
			value, err := s.service.GetNode(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(value, renderNode(value))
		},
	}
}
