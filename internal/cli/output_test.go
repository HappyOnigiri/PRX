package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestWriteAcceptsObjectAndUsesCompactJSON(t *testing.T) {
	var out bytes.Buffer
	s := &state{out: &out, json: true}
	if err := s.write(map[string]string{"value": "ok"}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), `{"schema_version":"2","ok":true,"data":{"value":"ok"}}
`; got != want {
		t.Fatalf("compact output = %q, want %q", got, want)
	}
}

func TestWriteUsesIndentedJSONByDefault(t *testing.T) {
	var out bytes.Buffer
	s := &state{out: &out}
	if err := s.write(map[string]string{"value": "ok"}); err != nil {
		t.Fatal(err)
	}
	want := "{\n" +
		"  \"schema_version\": \"2\",\n" +
		"  \"ok\": true,\n" +
		"  \"data\": {\n" +
		"    \"value\": \"ok\"\n" +
		"  }\n" +
		"}\n"
	if got := out.String(); got != want {
		t.Fatalf("indented output = %q, want %q", got, want)
	}
}

func TestWriteRejectsNonObjectData(t *testing.T) {
	values := map[string]any{
		"array":  []string{"value"},
		"scalar": "value",
		"null":   nil,
	}
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			s := &state{out: &out}
			err := s.write(value)
			if err == nil || !strings.Contains(err.Error(), "JSON object") {
				t.Fatalf("write error = %v, want JSON object error", err)
			}
			if out.Len() != 0 {
				t.Fatalf("write emitted output after validation failure: %q", out.String())
			}
		})
	}
}

func TestWriteErrorUsesExclusiveStringError(t *testing.T) {
	var out bytes.Buffer
	s := &state{out: &out, json: true}
	if err := s.writeError(errors.New("command failed")); err != nil {
		t.Fatal(err)
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if len(value) != 3 {
		t.Fatalf("error keys = %v, want three keys", value)
	}
	if _, ok := value["data"]; ok {
		t.Fatalf("error response contains data: %s", out.String())
	}
	var message string
	if err := json.Unmarshal(value["error"], &message); err != nil {
		t.Fatal(err)
	}
	if message != "command failed" {
		t.Fatalf("error message = %q", message)
	}
	if got := string(value["schema_version"]); got != `"2"` {
		t.Fatalf("schema version = %s", got)
	}
}

func TestPrintErrorUsesSchemaVersionAndStringMessage(t *testing.T) {
	var out bytes.Buffer
	if err := PrintError(&out, errors.New("boom")); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), `{"schema_version":"2","ok":false,"error":"boom"}
`; got != want {
		t.Fatalf("error output = %q, want %q", got, want)
	}
}

func TestWriteSchemaVersionOmitsEnvelopeFields(t *testing.T) {
	var out bytes.Buffer
	s := &state{out: &out, json: true}
	if err := s.writeSchemaVersion(); err != nil {
		t.Fatal(err)
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if len(value) != 1 || string(value["schema_version"]) != `"2"` {
		t.Fatalf("schema-version output = %s", out.String())
	}
}
