package cli

import (
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/spf13/cobra"
)

func (s *state) featureCommand() *cobra.Command {
	command := &cobra.Command{Use: "feature", Short: "Manage features"}
	var slug, title, description, status string
	var archived, cascade bool
	create := &cobra.Command{Use: "create", Short: "Create a feature", Example: "prx feature create --slug checkout --title \"Checkout rollout\" --json", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		value, err := s.service.CreateFeature(cmd.Context(), slug, title, description)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	create.Flags().StringVar(&slug, "slug", "", "stable feature slug")
	create.Flags().StringVar(&title, "title", "", "feature title")
	create.Flags().StringVar(&description, "description", "", "feature description")
	_ = create.MarkFlagRequired("slug")
	_ = create.MarkFlagRequired("title")
	list := &cobra.Command{Use: "list", Short: "List features", Example: "prx feature list --json", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		value, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		return s.write(value.Features)
	}}
	get := &cobra.Command{Use: "get FEATURE_ID_OR_SLUG", Short: "Show a feature by ID or slug", Example: "prx feature get checkout --json", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := s.service.ResolveFeature(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	update := &cobra.Command{Use: "update FEATURE_ID_OR_SLUG", Short: "Update a feature by ID or slug", Example: "prx feature update checkout --archived=false --json", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := s.service.UpdateFeature(cmd.Context(), args[0],
			changedFlag(cmd, "slug", &slug), changedFlag(cmd, "title", &title),
			changedFlag(cmd, "description", &description), changedStringType[domain.FeatureStatus](cmd, "status", &status),
			changedBoolFlag(cmd, "archived", &archived))
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	update.Flags().StringVar(&slug, "slug", "", "new slug")
	update.Flags().StringVar(&title, "title", "", "new title")
	update.Flags().StringVar(&description, "description", "", "new description")
	update.Flags().StringVar(&status, "status", "", "active, paused, completed, or cancelled")
	update.Flags().BoolVar(&archived, "archived", false, "archive (true) or unarchive (false) the feature")
	archive := &cobra.Command{Use: "archive FEATURE_ID_OR_SLUG", Short: "Archive a feature by ID or slug", Example: "prx feature archive checkout --json", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		archive := true
		value, err := s.service.UpdateFeature(cmd.Context(), args[0], nil, nil, nil, nil, &archive)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	unarchive := &cobra.Command{Use: "unarchive FEATURE_ID_OR_SLUG", Short: "Unarchive a feature by ID or slug", Example: "prx feature unarchive checkout --json", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		archive := false
		value, err := s.service.UpdateFeature(cmd.Context(), args[0], nil, nil, nil, nil, &archive)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	deleteCmd := &cobra.Command{Use: "delete FEATURE_ID_OR_SLUG", Short: "Delete a feature and optionally its contained data", Example: "prx feature delete checkout --cascade --json", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := s.service.DeleteFeature(cmd.Context(), args[0], cascade); err != nil {
			return err
		}
		return s.write(map[string]string{"deleted": args[0]})
	}}
	deleteCmd.Flags().BoolVar(&cascade, "cascade", false, "delete contained tasks and references")
	command.AddCommand(create, list, get, update, archive, unarchive, deleteCmd)
	return command
}
