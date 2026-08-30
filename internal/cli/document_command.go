package cli

import (
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
	var feature, task, kind, title, value string
	add := &cobra.Command{
		Use:     "add",
		Short:   "Add a URL or local Markdown path",
		Example: "prx document add --task TASK_ID --kind markdown_path --value docs/checkout.md",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			doc, err := s.service.AddDocument(cmd.Context(), feature, task, domain.DocumentKind(kind), title, value)
			if err != nil {
				return err
			}
			return s.write(doc, renderMessage("Added document %s.", doc.ID))
		},
	}
	add.Flags().StringVar(&feature, "feature", "", "feature ID or slug")
	add.Flags().StringVar(&task, "task", "", "task ID")
	add.Flags().StringVar(&kind, "kind", "url", "url or markdown_path")
	add.Flags().StringVar(&title, "title", "", "document title")
	add.Flags().StringVar(&value, "value", "", "URL or Markdown path")
	_ = add.MarkFlagRequired("value")
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
