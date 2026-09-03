package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

type documentSourceFlags struct {
	url, localFile, markdownFile string
	stdin                        bool
}

func (s *state) documentCommand() *cobra.Command {
	var projectFilter, featureFilter, taskFilter string
	command := &cobra.Command{
		Use:     "document",
		Aliases: []string{"doc"},
		Short:   "List or manage documents",
		Long:    "List or manage URL, local file, and stored Markdown documents.\n\nAlias: doc.",
		Example: "prx document\nprx document --task T-1\nprx document --project P-1\nprx doc",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			parent, err := s.resolveDocumentFilter(cmd, snapshot, projectFilter, featureFilter, taskFilter)
			if err != nil {
				return err
			}
			documents := make([]domain.Document, 0, len(snapshot.Documents))
			for _, document := range snapshot.Documents {
				if parent.ProjectID != "" && document.ProjectID != parent.ProjectID {
					continue
				}
				if parent.FeatureID != "" && document.FeatureID != parent.FeatureID {
					continue
				}
				if parent.TaskID != "" && document.TaskID != parent.TaskID {
					continue
				}
				documents = append(documents, document)
			}
			return s.write(map[string]any{"documents": documents}, renderDocumentList(documents))
		},
	}
	command.Flags().StringVar(&projectFilter, "project", "", "filter by project ID or slug")
	command.Flags().StringVar(&featureFilter, "feature", "", "filter by feature ID or slug")
	command.Flags().StringVar(&taskFilter, "task", "", "filter by task ID")

	command.AddCommand(
		s.documentAddCommand(),
		s.documentGetCommand(),
		s.documentUpdateCommand(),
		s.documentDeleteCommand(),
	)
	return command
}

// resolveDocumentFilter turns the list filters into public identifiers, so the
// listing compares against the same values the snapshot carries.
func (s *state) resolveDocumentFilter(
	cmd *cobra.Command,
	snapshot domain.Snapshot,
	projectFilter, featureFilter, taskFilter string,
) (domain.DocumentParent, error) {
	parent := domain.DocumentParent{TaskID: taskFilter}
	if projectFilter != "" {
		project, err := s.service.ResolveProject(cmd.Context(), projectFilter)
		if err != nil {
			return domain.DocumentParent{}, err
		}
		parent.ProjectID = project.ID
	}
	if featureFilter != "" {
		feature, err := s.service.ResolveFeature(cmd.Context(), featureFilter)
		if err != nil {
			return domain.DocumentParent{}, err
		}
		parent.FeatureID = feature.ID
	}
	if taskFilter != "" {
		if err := requireTask(snapshot, taskFilter); err != nil {
			return domain.DocumentParent{}, err
		}
	}
	return parent, nil
}

func (s *state) documentAddCommand() *cobra.Command {
	var title string
	var implementationPlan bool
	var source documentSourceFlags
	command := &cobra.Command{
		Use:   "add PROJECT_OR_FEATURE_OR_TASK",
		Short: "Add a document to a project, a feature, or a task",
		Long: "Add a document to a project, a feature, or a task.\n\n" +
			"The operand is a public project, feature, or task ID, or a feature or project slug.",
		Example: "prx document add T-1 --url https://example.com\n" +
			"prx document add P-1 --url https://example.com\n" +
			"prx document add checkout --markdown-file notes.md",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := readDocumentSource(cmd, source, domain.DomainErrorCodeInvalidDocument)
			if err != nil {
				return err
			}
			node, err := s.service.GetNode(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			var parent domain.DocumentParent
			switch typed := node.(type) {
			case domain.Project:
				parent.ProjectID = typed.ID
			case domain.Feature:
				parent.FeatureID = typed.ID
			case domain.Task:
				parent.TaskID = typed.ID
			default:
				return domain.NewError(
					domain.DomainErrorCodeInvalidParent,
					"%q resolved to an unsupported document parent",
					args[0],
				)
			}
			doc, err := s.service.AddDocument(
				cmd.Context(),
				parent,
				document.Kind,
				title,
				document.Locator,
				document.Content,
				implementationPlan,
			)
			if err != nil {
				return err
			}
			return s.write(doc, renderMessage("Added document %s.", doc.ID))
		},
	}
	command.Flags().StringVar(&title, "title", "", "document title")
	command.Flags().BoolVar(&implementationPlan, "implementation-plan", false, "mark as the task implementation plan")
	bindDocumentSourceFlags(command, &source)
	return command
}

func (s *state) documentGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get DOCUMENT_ID",
		Short:   "Get one document",
		Example: "prx document get DOCUMENT_ID",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := s.service.GetDocument(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(document, renderDocumentDetail(document))
		},
	}
}

func (s *state) documentUpdateCommand() *cobra.Command {
	var updateTitle string
	var updatePlan bool
	var updateSource documentSourceFlags
	command := &cobra.Command{
		Use:     "update DOCUMENT_ID",
		Short:   "Update a document",
		Example: "prx document update DOCUMENT_ID --title Runbook",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var titlePointer *string
			if cmd.Flags().Changed("title") {
				titlePointer = &updateTitle
			}
			var sourcePointer *domain.Document
			if documentSourceCount(updateSource) > 0 {
				value, err := readDocumentSource(cmd, updateSource, domain.DomainErrorCodeInvalidDocument)
				if err != nil {
					return err
				}
				sourcePointer = &value
			}
			var planPointer *bool
			if cmd.Flags().Changed("implementation-plan") {
				planPointer = &updatePlan
			}
			document, err := s.service.UpdateDocument(
				cmd.Context(), args[0], titlePointer, sourcePointer, planPointer,
			)
			if err != nil {
				return err
			}
			return s.write(document, renderDocumentDetail(document))
		},
	}
	command.Flags().StringVar(&updateTitle, "title", "", "replace the document title")
	command.Flags().BoolVar(&updatePlan, "implementation-plan", false, "set or clear the plan designation")
	bindDocumentSourceFlags(command, &updateSource)
	return command
}

func (s *state) documentDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "delete DOCUMENT_ID",
		Short:   "Delete a document; missing documents return not_found",
		Example: "prx document delete DOCUMENT_ID",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteDocument(cmd.Context(), args[0]); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]}, renderMessage("Deleted document %s.", args[0]))
		},
	}
}

func requireTask(snapshot domain.Snapshot, taskID string) error {
	for _, task := range snapshot.Tasks {
		if task.ID == taskID {
			return nil
		}
	}
	return domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", taskID)
}

func bindDocumentSourceFlags(command *cobra.Command, flags *documentSourceFlags) {
	command.Flags().StringVar(&flags.url, "url", "", "HTTP or HTTPS URL")
	command.Flags().StringVar(&flags.localFile, "local-file", "", "registered local file path")
	command.Flags().StringVar(&flags.markdownFile, "markdown-file", "", "read stored Markdown from a file")
	command.Flags().BoolVar(&flags.stdin, "stdin", false, "read stored Markdown from standard input")
}

func documentSourceCount(flags documentSourceFlags) int {
	count := 0
	for _, value := range []bool{
		flags.url != "", flags.localFile != "", flags.markdownFile != "", flags.stdin,
	} {
		if value {
			count++
		}
	}
	return count
}

func readDocumentSource(
	command *cobra.Command,
	flags documentSourceFlags,
	errorCode domain.DomainErrorCode,
) (domain.Document, error) {
	if documentSourceCount(flags) != 1 {
		return domain.Document{}, domain.NewError(errorCode, "specify exactly one document source")
	}
	if flags.url != "" {
		return domain.Document{Kind: domain.DocumentKindURL, Locator: flags.url}, nil
	}
	if flags.localFile != "" {
		return domain.Document{Kind: domain.DocumentKindLocalFile, Locator: flags.localFile}, nil
	}
	var content []byte
	var err error
	if flags.stdin {
		content, err = io.ReadAll(command.InOrStdin())
	} else {
		content, err = os.ReadFile(flags.markdownFile)
	}
	if err != nil {
		return domain.Document{}, domain.NewError(errorCode, "could not read Markdown: %v", err)
	}
	return domain.Document{Kind: domain.DocumentKindMarkdown, Content: string(content)}, nil
}
