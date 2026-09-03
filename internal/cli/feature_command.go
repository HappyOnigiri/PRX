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
			// Both forms read the snapshot so the derived status and the task
			// counts are the ones the server computes, matching prx task.
			value, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			if len(args) == 1 {
				for _, feature := range value.Features {
					if feature.ID == args[0] || feature.Slug == args[0] {
						return s.write(feature, renderFeatureDetail(feature))
					}
				}
				return domain.NewError(domain.DomainErrorCodeNotFound, "feature %q was not found", args[0])
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
	var description, project string
	command := &cobra.Command{
		Use:   "create SLUG TITLE",
		Short: "Create a feature",
		Example: "prx feature create checkout \"Checkout rollout\"\n" +
			"prx feature create checkout \"Checkout rollout\" --project payments\n" +
			"prx feature create checkout -- \"-fix checkout\"",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.CreateFeature(cmd.Context(), args[0], args[1], description, project)
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Created feature %s (%s).", value.Slug, value.ID))
		},
	}
	command.Flags().StringVar(&description, "description", "", "feature description")
	command.Flags().StringVar(&project, "project", "", "project ID or slug to join")
	return command
}

func (s *state) featureUpdateCommand() *cobra.Command {
	var slug, title, description, status, project string
	var archived bool
	command := &cobra.Command{
		Use:     "update FEATURE_ID_OR_SLUG",
		Short:   "Update a feature by ID or slug",
		Example: "prx feature update checkout --archived=false\nprx feature update checkout --project=",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.UpdateFeature(cmd.Context(), args[0], domain.FeatureUpdate{
				Slug:        changedFlag(cmd, "slug", &slug),
				Title:       changedFlag(cmd, "title", &title),
				Description: changedFlag(cmd, "description", &description),
				Status:      changedStringType[domain.FeatureStatus](cmd, "status", &status),
				Archived:    changedBoolFlag(cmd, "archived", &archived),
				ProjectID:   changedFlag(cmd, "project", &project),
			})
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Updated feature %s (%s).", value.Slug, value.ID))
		},
	}
	command.Flags().StringVar(&slug, "slug", "", "new slug")
	command.Flags().StringVar(&title, "title", "", "new title")
	command.Flags().StringVar(&description, "description", "", "new description")
	command.Flags().StringVar(&status, "status", "", "auto, active, paused, completed, or cancelled")
	command.Flags().BoolVar(&archived, "archived", false, "archive (true) or unarchive (false) the feature")
	command.Flags().StringVar(&project, "project", "", "project ID or slug; an empty value leaves the project")
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
			value, err := s.service.UpdateFeature(
				cmd.Context(),
				args[0],
				domain.FeatureUpdate{Archived: &archived},
			)
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
