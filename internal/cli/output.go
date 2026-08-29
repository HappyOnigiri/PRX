package cli

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

type envelope struct {
	SchemaVersion string        `json:"schema_version"`
	OK            bool          `json:"ok"`
	Data          any           `json:"data,omitempty"`
	Error         *domain.Error `json:"error,omitempty"`
}

func PrintError(out io.Writer, err error) error {
	value := &domain.Error{Code: domain.ErrorCode(err), Message: err.Error()}
	var typed *domain.Error
	if errors.As(err, &typed) {
		value = typed
	}
	return json.NewEncoder(out).Encode(envelope{SchemaVersion: "1", OK: false, Error: value})
}

func (s *state) write(value any) error {
	encoder := json.NewEncoder(s.out)
	if !s.json {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(envelope{SchemaVersion: "1", OK: true, Data: value})
}

// changedFlag returns the flag value only when it was given on the command line,
// so an omitted flag leaves the field untouched while --flag "" clears it.
func changedFlag(cmd *cobra.Command, name string, value *string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return value
}

func changedStringType[T ~string](cmd *cobra.Command, name string, value *string) *T {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	typed := T(*value)
	return &typed
}

func changedBoolFlag(cmd *cobra.Command, name string, value *bool) *bool {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return value
}
