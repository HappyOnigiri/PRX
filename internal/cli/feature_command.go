package cli

import (
	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) featureCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "feature [FEATURE_ID_OR_SLUG]",
		Aliases: []string{"f"},
		Short:   "List features or show one by ID or slug",
		Long:    "List features or show one by ID or slug.\n\nAlias: f.",
		Example: "prx feature\nprx feature F-1\nprx f checkout\nprx show create",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				value, err := s.service.ResolveFeature(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return s.write(value, renderFeatureDetail(value))
			}
			value, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			features := nonNilSlice(value.Features)
			return s.write(map[string]any{"features": features}, renderFeatureList(features))
		},
	}
	command.AddCommand(
		s.featureCreateCommand(),
		s.featureUpdateCommand(),
		s.featureArchiveCommand(true),
		s.featureArchiveCommand(false),
		s.featureDeleteCommand(),
	)
	return command
}

func (s *state) featureCreateCommand() *cobra.Command {
	var description string
	command := &cobra.Command{
		Use:     "create SLUG TITLE",
		Short:   "Create a feature",
		Example: "prx feature create checkout \"Checkout rollout\"",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.CreateFeature(cmd.Context(), args[0], args[1], description)
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Created feature %s (%s).", value.Slug, value.ID))
		},
	}
	command.Flags().StringVar(&description, "description", "", "feature description")
	return command
}

func (s *state) featureUpdateCommand() *cobra.Command {
	var slug, title, description, status string
	var archived bool
	command := &cobra.Command{
		Use:     "update FEATURE_ID_OR_SLUG",
		Short:   "Update a feature by ID or slug",
		Example: "prx feature update checkout --archived=false",
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
			return s.write(value, renderMessage("Updated feature %s (%s).", value.Slug, value.ID))
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
		Example: "prx feature " + verb + " checkout",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.UpdateFeature(cmd.Context(), args[0], nil, nil, nil, nil, &archived)
			if err != nil {
				return err
			}
			action := "Archived"
			if !archived {
				action = "Unarchived"
			}
			return s.write(value, renderMessage("%s feature %s (%s).", action, value.Slug, value.ID))
		},
	}
}

func (s *state) featureDeleteCommand() *cobra.Command {
	var cascade bool
	command := &cobra.Command{
		Use:     "delete FEATURE_ID_OR_SLUG",
		Short:   "Delete a feature and optionally its contained data",
		Example: "prx feature delete checkout --cascade",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteFeature(cmd.Context(), args[0], cascade); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]}, renderMessage("Deleted feature %s.", args[0]))
		},
	}
	command.Flags().BoolVar(&cascade, "cascade", false, "delete contained tasks and references")
	return command
}
