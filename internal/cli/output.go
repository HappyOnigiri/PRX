package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

const SchemaVersion = "1"

type envelope struct {
	SchemaVersion string          `json:"schema_version"`
	OK            bool            `json:"ok"`
	Data          json.RawMessage `json:"data,omitempty"`
	Error         string          `json:"error,omitempty"`
}

func PrintError(out io.Writer, err error) error {
	return writeFailure(out, err, true)
}

func (s *state) write(value any) error {
	data, err := marshalObject(value)
	if err != nil {
		return err
	}
	if !s.json {
		return encode(s.out, data, true)
	}
	return encode(s.out, envelope{SchemaVersion: SchemaVersion, OK: true, Data: data}, true)
}

func (s *state) writeError(err error) error {
	return writeFailure(s.out, err, s.json)
}

func (s *state) writeSchemaVersion() error {
	return encode(s.out, struct {
		SchemaVersion string `json:"schema_version"`
	}{SchemaVersion: SchemaVersion}, true)
}

func writeFailure(out io.Writer, err error, compact bool) error {
	return encode(out, envelope{SchemaVersion: SchemaVersion, OK: false, Error: errorMessage(err)}, compact)
}

func encode(out io.Writer, value any, compact bool) error {
	encoder := json.NewEncoder(out)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(value)
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
