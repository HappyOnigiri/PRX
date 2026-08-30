package cli

import "github.com/spf13/cobra"

func (s *state) schemaVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "schema-version",
		Short:   "Show the CLI response schema version",
		Example: "prx schema-version",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return s.writeSchemaVersion()
		},
	}
}
