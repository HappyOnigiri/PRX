package cli

import (
	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) featureCommand() *cobra.Command {
	command := &cobra.Command{Use: "feature", Short: "Manage features"}
	command.AddCommand(
		s.featureCreateCommand(),
		s.featureListCommand(),
		s.featureGetCommand(),
		s.featureUpdateCommand(),
		s.featureArchiveCommand(true),
		s.featureArchiveCommand(false),
		s.featureDeleteCommand(),
	)
	return command
}

func (s *state) featureCreateCommand() *cobra.Command {
	var slug, title, description string
	command := &cobra.Command{
		Use:     "create",
		Short:   "Create a feature",
		Example: "prx feature create --slug checkout --title \"Checkout rollout\" --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			value, err := s.service.CreateFeature(cmd.Context(), slug, title, description)
			if err != nil {
				return err
			}
			return s.write(value)
		},
	}
	command.Flags().StringVar(&slug, "slug", "", "stable feature slug")
	command.Flags().StringVar(&title, "title", "", "feature title")
	command.Flags().StringVar(&description, "description", "", "feature description")
	_ = command.MarkFlagRequired("slug")
	_ = command.MarkFlagRequired("title")
	return command
}

func (s *state) featureListCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List features",
		Example: "prx feature list --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			value, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			return s.write(map[string]any{"features": value.Features})
		},
	}
}

func (s *state) featureGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get FEATURE_ID_OR_SLUG",
		Short:   "Show a feature by ID or slug",
		Example: "prx feature get checkout --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.ResolveFeature(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(value)
		},
	}
}

func (s *state) featureUpdateCommand() *cobra.Command {
	var slug, title, description, status string
	var archived bool
	command := &cobra.Command{
		Use:     "update FEATURE_ID_OR_SLUG",
		Short:   "Update a feature by ID or slug",
		Example: "prx feature update checkout --archived=false --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.UpdateFeature(
				cmd.Context(),
				args[0],
				changedFlag(cmd, "slug", &slug),
				changedFlag(cmd, "title", &title),
				changedFlag(
					cmd,
					"description",
					&description,
				),
				changedStringType[domain.FeatureStatus](cmd, "status", &status),
				changedBoolFlag(cmd, "archived", &archived),
			)
			if err != nil {
				return err
			}
			return s.write(value)
		},
	}
	command.Flags().StringVar(&slug, "slug", "", "new slug")
	command.Flags().StringVar(&title, "title", "", "new title")
	command.Flags().StringVar(&description, "description", "", "new description")
	command.Flags().StringVar(&status, "status", "", "active, paused, completed, or cancelled")
	command.Flags().BoolVar(&archived, "archived", false, "archive (true) or unarchive (false) the feature")
	return command
}

func (s *state) featureArchiveCommand(archived bool) *cobra.Command {
	verb := "archive"
	short := "Archive a feature by ID or slug"
	if !archived {
		verb = "unarchive"
		short = "Unarchive a feature by ID or slug"
	}
	return &cobra.Command{
		Use:     verb + " FEATURE_ID_OR_SLUG",
		Short:   short,
		Example: "prx feature " + verb + " checkout --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.UpdateFeature(cmd.Context(), args[0], nil, nil, nil, nil, &archived)
			if err != nil {
				return err
			}
			return s.write(value)
		},
	}
}

func (s *state) featureDeleteCommand() *cobra.Command {
	var cascade bool
	command := &cobra.Command{
		Use:     "delete FEATURE_ID_OR_SLUG",
		Short:   "Delete a feature and optionally its contained data",
		Example: "prx feature delete checkout --cascade --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteFeature(cmd.Context(), args[0], cascade); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]})
		},
	}
	command.Flags().BoolVar(&cascade, "cascade", false, "delete contained tasks and references")
	return command
}
