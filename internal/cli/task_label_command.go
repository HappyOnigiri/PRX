package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func parseTaskLabelKey(raw string) (domain.TaskLabelKey, error) {
	key := domain.TaskLabelKey(strings.ToLower(strings.TrimSpace(raw)))
	for _, candidate := range domain.TaskLabelKeys() {
		if key == candidate {
			return key, nil
		}
	}
	return "", domain.NewError(domain.DomainErrorCodeInvalidTaskLabel, "unknown task label key %q", raw)
}

func taskLabelUpdateFromSet(cmd *cobra.Command, text, color string) (domain.TaskLabelOverrideUpdate, error) {
	if !cmd.Flags().Changed("text") && !cmd.Flags().Changed("color") {
		return domain.TaskLabelOverrideUpdate{}, &usageError{
			err: fmt.Errorf("at least one of --text or --color is required"),
		}
	}
	update := domain.TaskLabelOverrideUpdate{}
	if cmd.Flags().Changed("text") {
		update.Text = &text
	}
	if cmd.Flags().Changed("color") {
		update.Color = &color
	}
	return update, nil
}

func taskLabelUpdateFromUnset(cmd *cobra.Command, text, color bool) (domain.TaskLabelOverrideUpdate, error) {
	if !text && !color {
		return domain.TaskLabelOverrideUpdate{}, &usageError{
			err: fmt.Errorf("at least one of --text or --color is required"),
		}
	}
	update := domain.TaskLabelOverrideUpdate{}
	empty := ""
	if text {
		update.Text = &empty
	}
	if color {
		update.Color = &empty
	}
	return update, nil
}

func taskLabelOverridesUpdate(
	key domain.TaskLabelKey,
	value domain.TaskLabelOverrideUpdate,
) domain.TaskLabelOverridesUpdate {
	result := make(domain.TaskLabelOverridesUpdate)
	result[key] = value
	return result
}

func (s *state) configLabelCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "label",
		Short:   "Show or manage global task label overrides",
		Example: "prx config label set status.in_progress --text Working",
		Args:    cobra.NoArgs,
	}
	command.AddCommand(s.configLabelSetCommand(), s.configLabelUnsetCommand())
	return command
}

func (s *state) configLabelSetCommand() *cobra.Command {
	var text, color string
	command := &cobra.Command{
		Use:     "set KEY",
		Short:   "Set a global task label text or color",
		Example: "prx config label set status.in_progress --text Working",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := parseTaskLabelKey(args[0])
			if err != nil {
				return err
			}
			update, err := taskLabelUpdateFromSet(cmd, text, color)
			if err != nil {
				return err
			}
			updates := taskLabelOverridesUpdate(key, update)
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Update(func(settings *config.Config) error {
				return settings.SetTaskLabelOverrides(updates)
			})
			if err != nil {
				return configCommandError(err)
			}
			return s.write(
				map[string]any{"task_labels": settings.TaskLabels},
				renderMessage("Set global task label %s.", key),
			)
		},
	}
	command.Flags().StringVar(&text, "text", "", "label text")
	command.Flags().StringVar(&color, "color", "", "label color (#RRGGBB)")
	return command
}

func (s *state) configLabelUnsetCommand() *cobra.Command {
	var text, color bool
	command := &cobra.Command{
		Use:     "unset KEY",
		Short:   "Remove a global task label text or color override",
		Example: "prx config label unset status.in_progress --text",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := parseTaskLabelKey(args[0])
			if err != nil {
				return err
			}
			update, err := taskLabelUpdateFromUnset(cmd, text, color)
			if err != nil {
				return err
			}
			updates := taskLabelOverridesUpdate(key, update)
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Update(func(settings *config.Config) error {
				return settings.SetTaskLabelOverrides(updates)
			})
			if err != nil {
				return configCommandError(err)
			}
			return s.write(
				map[string]any{"task_labels": settings.TaskLabels},
				renderMessage("Unset global task label %s.", key),
			)
		},
	}
	command.Flags().BoolVar(&text, "text", false, "remove text override")
	command.Flags().BoolVar(&color, "color", false, "remove color override")
	return command
}

func (s *state) projectLabelCommand() *cobra.Command {
	command := &cobra.Command{Use: "label", Short: "Manage project task label overrides", Args: cobra.NoArgs}
	command.AddCommand(s.scopedLabelSetCommand("project"), s.scopedLabelUnsetCommand("project"))
	return command
}

func (s *state) featureLabelCommand() *cobra.Command {
	command := &cobra.Command{Use: "label", Short: "Manage feature task label overrides", Args: cobra.NoArgs}
	command.AddCommand(s.scopedLabelSetCommand("feature"), s.scopedLabelUnsetCommand("feature"))
	return command
}

func (s *state) scopedLabelSetCommand(scope string) *cobra.Command {
	var text, color string
	command := &cobra.Command{
		Use:     "set " + strings.ToUpper(scope) + "_ID KEY",
		Short:   "Set a task label text or color override",
		Example: "prx " + scope + " label set " + strings.ToUpper(scope) + "_ID status.in_progress --text Working",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := parseTaskLabelKey(args[1])
			if err != nil {
				return err
			}
			update, err := taskLabelUpdateFromSet(cmd, text, color)
			if err != nil {
				return err
			}
			updates := taskLabelOverridesUpdate(key, update)
			if scope == "project" {
				value, updateErr := s.service.UpdateProject(
					cmd.Context(),
					args[0],
					domain.ProjectUpdate{TaskLabelOverrides: &updates},
				)
				if updateErr != nil {
					return updateErr
				}
				return s.write(value, renderMessage("Set project task label %s.", key))
			}
			value, updateErr := s.service.UpdateFeature(
				cmd.Context(),
				args[0],
				domain.FeatureUpdate{TaskLabelOverrides: &updates},
			)
			if updateErr != nil {
				return updateErr
			}
			return s.write(value, renderMessage("Set feature task label %s.", key))
		},
	}
	command.Flags().StringVar(&text, "text", "", "label text")
	command.Flags().StringVar(&color, "color", "", "label color (#RRGGBB)")
	return command
}

func (s *state) scopedLabelUnsetCommand(scope string) *cobra.Command {
	var text, color bool
	command := &cobra.Command{
		Use:     "unset " + strings.ToUpper(scope) + "_ID KEY",
		Short:   "Remove a task label text or color override",
		Example: "prx " + scope + " label unset " + strings.ToUpper(scope) + "_ID status.in_progress --text",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := parseTaskLabelKey(args[1])
			if err != nil {
				return err
			}
			update, err := taskLabelUpdateFromUnset(cmd, text, color)
			if err != nil {
				return err
			}
			updates := taskLabelOverridesUpdate(key, update)
			if scope == "project" {
				value, updateErr := s.service.UpdateProject(
					cmd.Context(),
					args[0],
					domain.ProjectUpdate{TaskLabelOverrides: &updates},
				)
				if updateErr != nil {
					return updateErr
				}
				return s.write(value, renderMessage("Unset project task label %s.", key))
			}
			value, updateErr := s.service.UpdateFeature(
				cmd.Context(),
				args[0],
				domain.FeatureUpdate{TaskLabelOverrides: &updates},
			)
			if updateErr != nil {
				return updateErr
			}
			return s.write(value, renderMessage("Unset feature task label %s.", key))
		},
	}
	command.Flags().BoolVar(&text, "text", false, "remove text override")
	command.Flags().BoolVar(&color, "color", false, "remove color override")
	return command
}
