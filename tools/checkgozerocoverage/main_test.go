package main

import (
	"reflect"
	"testing"
)

func TestParseCoverageOutput(t *testing.T) {
	got, err := parseCoverageOutput("" +
		"internal/z.go:20: zero\t\t0.0%\n" +
		"internal/a.go:10: covered\t\t0.1%\n" +
		"internal/a.go:2: missing\t\t0.0%\n" +
		"total: (statements) 0.0%\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []coverageFunction{
		{file: "internal/a.go", line: 2, name: "missing"},
		{file: "internal/z.go", line: 20, name: "zero"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCoverageOutput()=%v, want %v", got, want)
	}
}

func TestParseCoverageOutputRejectsEmptyOrInvalidOutput(t *testing.T) {
	for _, test := range []struct {
		name   string
		output string
	}{
		{name: "empty", output: ""},
		{name: "missing total", output: "file.go:1: Function\t\t0.0%\n"},
		{name: "invalid line", output: "not cover output\ntotal: (statements) 1.0%\n"},
		{name: "no functions", output: "total: (statements) 1.0%\n"},
		{name: "invalid total", output: "file.go:1: Function\t\t1.0%\ntotal: invalid\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseCoverageOutput(test.output); err == nil {
				t.Fatal("parseCoverageOutput() unexpectedly succeeded")
			}
		})
	}
}

func TestParseCoverageOutputRejectsMalformedPercentage(t *testing.T) {
	_, err := parseCoverageOutput("file.go:1: Function\t\t0%\n" + "total: (statements) 0.0%\n")
	if err == nil {
		t.Fatal("parseCoverageOutput() unexpectedly accepted malformed percentage")
	}
}

func TestCoverageFunctionString(t *testing.T) {
	function := coverageFunction{file: "internal/app/service.go", line: 138, name: "(*Service).DeleteFeature"}
	if got, want := function.String(), "internal/app/service.go:138 (*Service).DeleteFeature"; got != want {
		t.Fatalf("coverageFunction.String()=%q, want %q", got, want)
	}
}
