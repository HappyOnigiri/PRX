package cli

import (
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/spf13/cobra"
)

func (s *state) featureCommand() *cobra.Command {
	command := &cobra.Command{Use: "feature", Short: "Manage features"}
	var slug, title, description, status string
	var archived, cascade bool
	create := &cobra.Command{Use: "create", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
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
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		value, err := s.service.Snapshot(cmd.Context())
		if err != nil {
			return err
		}
		return s.write(value.Features)
	}}
	get := &cobra.Command{Use: "get ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := s.service.ResolveFeature(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	update := &cobra.Command{Use: "update ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
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
	archive := &cobra.Command{Use: "archive ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		archive := true
		value, err := s.service.UpdateFeature(cmd.Context(), args[0], nil, nil, nil, nil, &archive)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	unarchive := &cobra.Command{Use: "unarchive ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		archive := false
		value, err := s.service.UpdateFeature(cmd.Context(), args[0], nil, nil, nil, nil, &archive)
		if err != nil {
			return err
		}
		return s.write(value)
	}}
	deleteCmd := &cobra.Command{Use: "delete ID_OR_SLUG", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := s.service.DeleteFeature(cmd.Context(), args[0], cascade); err != nil {
			return err
		}
		return s.write(map[string]string{"deleted": args[0]})
	}}
	deleteCmd.Flags().BoolVar(&cascade, "cascade", false, "delete contained tasks and references")
	command.AddCommand(create, list, get, update, archive, unarchive, deleteCmd)
	return command
}
