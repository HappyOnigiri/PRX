package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func TestWriteJSONReturnsDataObjectWithoutSuccessEnvelope(t *testing.T) {
	var out bytes.Buffer
	s := &state{out: &out, errOut: io.Discard, json: true}
	if err := s.write(map[string]string{"value": "ok"}, renderMessage("ignored")); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), `{"value":"ok"}
`; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWriteUsesTextRendererByDefault(t *testing.T) {
	var out bytes.Buffer
	s := &state{out: &out, errOut: io.Discard}
	if err := s.write(map[string]string{"value": "ok"}, renderMessage("Value: ok")); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "Value: ok\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWriteRejectsNonObjectJSONData(t *testing.T) {
	values := map[string]any{"array": []string{"value"}, "scalar": "value", "null": nil}
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			s := &state{out: &out, json: true}
			err := s.write(value, renderMessage("ignored"))
			if err == nil || !strings.Contains(err.Error(), "JSON object") {
				t.Fatalf("write error = %v, want JSON object error", err)
			}
			if out.Len() != 0 {
				t.Fatalf("write emitted output after validation failure: %q", out.String())
			}
		})
	}
}

func TestWriteJSONErrorUsesCodeAndMessageOnStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := &state{out: &stdout, errOut: &stderr, json: true}
	err := domain.NewError(domain.DomainErrorCodeNotFound, "resource was not found")
	if writeErr := s.writeError(err, ""); writeErr != nil {
		t.Fatal(writeErr)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	var value errorEnvelope
	if decodeErr := json.Unmarshal(stderr.Bytes(), &value); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if value.Error.Code != string(domain.DomainErrorCodeNotFound) || value.Error.Message != "resource was not found" {
		t.Fatalf("error response = %+v", value)
	}
	if got, want := stderr.String(), `{"error":{"code":"not_found","message":"resource was not found","hint":""}}
`; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestWriteDefaultErrorUsesPrefixOnStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := &state{out: &stdout, errOut: &stderr}
	if err := s.writeError(errors.New("command failed"), ""); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 || stderr.String() != "Error: command failed\n" {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestPrintErrorUsesJSONErrorContract(t *testing.T) {
	var out bytes.Buffer
	if err := PrintError(&out, errors.New("boom")); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), `{"error":{"code":"internal","message":"boom","hint":""}}
`; got != want {
		t.Fatalf("error output = %q, want %q", got, want)
	}
}

func TestWriteSchemaVersionFollowsJSONFlag(t *testing.T) {
	for _, test := range []struct {
		name string
		json bool
		want string
	}{
		{name: "json", json: true, want: `{"schema_version":"2"}
`},
		{name: "default text", want: "Schema version: 2\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			s := &state{out: &out, json: test.json}
			if err := s.writeSchemaVersion(); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != test.want {
				t.Fatalf("schema-version output = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRenderAuthListNeverIncludesSecretHint(t *testing.T) {
	method := configAuthFixture()
	var out bytes.Buffer
	if err := renderAuthList([]config.PublicAuthMethod{method})(&out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), method.SecretHint) || strings.Contains(out.String(), "token-value") {
		t.Fatalf("human auth output leaked secret metadata: %q", out.String())
	}
	if !strings.Contains(out.String(), "yes") {
		t.Fatalf("human auth output omitted configured state: %q", out.String())
	}
}

func configAuthFixture() config.PublicAuthMethod {
	return config.PublicAuthMethod{
		ID: "work", Host: "github.com", Type: config.AuthMethodTypeKeychain,
		Account: "user", Service: "gh-token", SecretConfigured: true, SecretHint: "token-value",
	}
}

func TestHumanTablesAreDeterministic(t *testing.T) {
	feature := domain.Feature{ID: "F-1", Slug: "checkout", Status: domain.FeatureStatusActive, Title: "Checkout"}
	var first, second bytes.Buffer
	for _, out := range []*bytes.Buffer{&first, &second} {
		if err := renderFeatureList([]domain.Feature{feature})(out); err != nil {
			t.Fatal(err)
		}
	}
	if first.String() != second.String() {
		t.Fatalf("table output changed: %q != %q", first.String(), second.String())
	}
	if !strings.Contains(first.String(), "ID") || !strings.Contains(first.String(), "F-1") {
		t.Fatalf("table output = %q", first.String())
	}
}

func Example_renderMessage() {
	_ = renderMessage("Created feature %s.", "checkout")(io.Discard)
	fmt.Println("ok")
	// Output: ok
}
