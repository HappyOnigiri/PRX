package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) documentCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "document",
		Aliases: []string{"doc"},
		Short:   "List or manage URL and local Markdown references",
		Long:    "List or manage URL and local Markdown references.\n\nAlias: doc.",
		Example: "prx document\nprx doc",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			documents := nonNilSlice(snapshot.Documents)
			return s.write(map[string]any{"documents": documents}, renderDocumentList(documents))
		},
	}
	var kind, title string
	add := &cobra.Command{
		Use:     "add FEATURE_ID_OR_SLUG_OR_TASK_ID VALUE",
		Short:   "Add a URL or local Markdown path",
		Example: "prx document add TASK_ID docs/checkout.md --kind markdown_path",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			feature, task := args[0], ""
			if strings.HasPrefix(args[0], "T-") {
				feature, task = "", args[0]
			}
			doc, err := s.service.AddDocument(cmd.Context(), feature, task, domain.DocumentKind(kind), title, args[1])
			if err != nil {
				return err
			}
			return s.write(doc, renderMessage("Added document %s.", doc.ID))
		},
	}
	add.Flags().StringVar(&kind, "kind", "url", "url or markdown_path")
	add.Flags().StringVar(&title, "title", "", "document title")
	deleteCmd := &cobra.Command{
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
	command.AddCommand(add, deleteCmd)
	return command
}
