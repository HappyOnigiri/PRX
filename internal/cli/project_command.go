package cli

import (
	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) projectCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "project [PROJECT_ID]",
		Aliases: []string{"proj"},
		Short:   "List projects or show one by ID",
		Long:    "List projects or show one by ID.\n\nAlias: proj.",
		Example: "prx project\nprx project P-1\nprx proj P-1",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Both forms read the snapshot so the features listed with a project
			// are the ones the server reports, matching prx feature.
			value, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			if len(args) == 1 {
				return s.writeProjectDetail(value, args[0])
			}
			projects := nonNilSlice(value.Projects)
			return s.write(map[string]any{"projects": projects}, renderProjectList(projects))
		},
	}
	command.AddCommand(
		s.projectCreateCommand(),
		s.projectUpdateCommand(),
		s.projectArchiveCommand(true),
		s.projectArchiveCommand(false),
		s.projectDeleteCommand(),
	)
	return command
}

func (s *state) writeProjectDetail(snapshot domain.Snapshot, id string) error {
	for _, project := range snapshot.Projects {
		if project.ID != id {
			continue
		}
		features := make([]domain.Feature, 0, len(snapshot.Features))
		documents := make([]domain.Document, 0, len(snapshot.Documents))
		for _, feature := range snapshot.Features {
			if feature.ProjectID == project.ID {
				features = append(features, feature)
			}
		}
		for _, document := range snapshot.Documents {
			if document.ProjectID == project.ID {
				documents = append(documents, document)
			}
		}
		return s.write(
			map[string]any{"project": project, "features": features, "documents": documents},
			renderProjectDetail(project, features, documents),
		)
	}
	return domain.NewError(domain.DomainErrorCodeNotFound, "project %q was not found", id)
}

func (s *state) projectCreateCommand() *cobra.Command {
	var description string
	command := &cobra.Command{
		Use:     "create TITLE",
		Short:   "Create a project",
		Example: "prx project create \"Payments platform\"",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.CreateProject(cmd.Context(), args[0], description)
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Created project %s (%s).", value.ID, value.Title))
		},
	}
	command.Flags().StringVar(&description, "description", "", "project description")
	return command
}

func (s *state) projectUpdateCommand() *cobra.Command {
	var title, description string
	var archived bool
	command := &cobra.Command{
		Use:     "update PROJECT_ID",
		Short:   "Update a project by ID",
		Example: "prx project update P-1 --title \"Payments platform\"",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.UpdateProject(cmd.Context(), args[0], domain.ProjectUpdate{
				Title:       changedFlag(cmd, "title", &title),
				Description: changedFlag(cmd, "description", &description),
				Archived:    changedBoolFlag(cmd, "archived", &archived),
			})
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Updated project %s (%s).", value.ID, value.Title))
		},
	}
	command.Flags().StringVar(&title, "title", "", "new title")
	command.Flags().StringVar(&description, "description", "", "new description")
	command.Flags().BoolVar(&archived, "archived", false, "archive (true) or unarchive (false) the project")
	return command
}

func (s *state) projectArchiveCommand(archived bool) *cobra.Command {
	verb := "archive"
	short := "Archive a project and make its features read-only"
	if !archived {
		verb = "unarchive"
		short = "Unarchive a project and let its features accept writes again"
	}
	return &cobra.Command{
		Use:     verb + " PROJECT_ID",
		Short:   short,
		Example: "prx project " + verb + " P-1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := s.service.UpdateProject(
				cmd.Context(),
				args[0],
				domain.ProjectUpdate{Archived: &archived},
			)
			if err != nil {
				return err
			}
			action := "Archived"
			if !archived {
				action = "Unarchived"
			}
			return s.write(value, renderMessage("%s project %s (%s).", action, value.ID, value.Title))
		},
	}
}

func (s *state) projectDeleteCommand() *cobra.Command {
	var cascade bool
	command := &cobra.Command{
		Use:   "delete PROJECT_ID",
		Short: "Delete a project; --cascade removes its documents and the features it holds",
		Long: "Delete a project.\n\n" +
			"Without --cascade the command fails while the project still has features or documents.\n\n" +
			"With --cascade it deletes the project's own documents and every feature inside it,\n" +
			"together with the tasks, dependencies, pull-request attachments, and documents those\n" +
			"features own. A feature cannot outlive its project, because it belongs to one.",
		Example: "prx project delete P-1 --cascade",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteProject(cmd.Context(), args[0], cascade); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]}, renderMessage("Deleted project %s.", args[0]))
		},
	}
	command.Flags().BoolVar(&cascade, "cascade", false, "delete the project's documents and the features it holds")
	return command
}
