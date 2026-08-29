package cli

import (
	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) documentCommand() *cobra.Command {
	command := &cobra.Command{Use: "document", Short: "Manage URL and local Markdown references"}
	var feature, task, kind, title, value string
	add := &cobra.Command{
		Use:     "add",
		Short:   "Add a URL or local Markdown path",
		Example: "prx document add --task TASK_ID --kind markdown_path --value docs/checkout.md --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			doc, err := s.service.AddDocument(cmd.Context(), feature, task, domain.DocumentKind(kind), title, value)
			if err != nil {
				return err
			}
			return s.write(doc)
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
		Example: "prx document delete DOCUMENT_ID --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteDocument(cmd.Context(), args[0]); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]})
		},
	}
	list := &cobra.Command{
		Use:     "list",
		Short:   "List URL and local Markdown documents",
		Example: "prx document list --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			return s.write(snapshot.Documents)
		},
	}
	command.AddCommand(add, deleteCmd, list)
	return command
}
