package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

const SchemaVersion = "2"

type outputMode uint8

const (
	outputModeJSON outputMode = iota + 1
	outputModeHuman
)

type humanRenderer func(io.Writer) error

type errorEnvelope struct {
	Error errorData `json:"error"`
}

type errorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func PrintError(out io.Writer, err error) error {
	return writeJSONFailure(out, err, "")
}

func (s *state) write(value any, render humanRenderer) error {
	if err := s.resolveOutputMode(); err != nil {
		return err
	}
	if s.mode == outputModeHuman {
		if render == nil {
			return errors.New("human output renderer is required")
		}
		return render(s.out)
	}
	data, err := marshalObject(value)
	if err != nil {
		return err
	}
	return encodeJSON(s.out, data)
}

func (s *state) writeError(err error, hints ...string) error {
	if resolveErr := s.resolveOutputMode(); resolveErr != nil && err == nil {
		err = resolveErr
	}
	if s.mode == outputModeHuman {
		hint := firstHint(hints)
		if hint == "" {
			_, writeErr := fmt.Fprintf(s.errOut, "Error: %s\n", errorMessage(err))
			return writeErr
		}
		_, writeErr := fmt.Fprintf(s.errOut, "Error: %s\n\n%s", errorMessage(err), hint)
		return writeErr
	}
	return writeJSONFailure(s.errOut, err, firstHint(hints))
}

func (s *state) writeSchemaVersion() error {
	return s.write(
		struct {
			SchemaVersion string `json:"schema_version"`
		}{SchemaVersion: SchemaVersion},
		func(out io.Writer) error {
			_, err := fmt.Fprintf(out, "Schema version: %s\n", SchemaVersion)
			return err
		},
	)
}

func writeJSONFailure(out io.Writer, err error, hint string) error {
	return encodeJSON(out, errorEnvelope{Error: errorData{
		Code: commandErrorCode(err), Message: errorMessage(err), Hint: hint,
	}})
}

func firstHint(hints []string) string {
	if len(hints) == 0 {
		return ""
	}
	return hints[0]
}

func commandErrorCode(err error) string {
	var usageErr *usageError
	if errors.As(err, &usageErr) {
		return "usage_error"
	}
	return string(domain.ErrorCode(err))
}

func encodeJSON(out io.Writer, value any) error {
	return json.NewEncoder(out).Encode(value)
}

func marshalObject(value any) (json.RawMessage, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode response data: %w", err)
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, errors.New("response data must be a JSON object")
	}
	return data, nil
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func errorMessage(err error) string {
	if err == nil {
		return "command failed"
	}
	if message := err.Error(); strings.TrimSpace(message) != "" {
		return message
	}
	return "command failed"
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
