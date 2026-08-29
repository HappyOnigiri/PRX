package main

import "testing"

func TestParseMinimums(t *testing.T) {
	data := []byte("# package minimums\n\ninternal/app 20.3\ninternal/domain 83.1\n")

	minimums, err := parseMinimums(data)
	if err != nil {
		t.Fatalf("parseMinimums() error = %v", err)
	}
	if len(minimums) != 2 {
		t.Fatalf("parseMinimums() returned %d entries, want 2", len(minimums))
	}
	if minimums[0].Package != "internal/app" || minimums[0].Value != 20.3 {
		t.Fatalf("first minimum = %#v, want internal/app 20.3", minimums[0])
	}
	if minimums[1].Package != "internal/domain" || minimums[1].Value != 83.1 {
		t.Fatalf("second minimum = %#v, want internal/domain 83.1", minimums[1])
	}
}

func TestParseMinimumsRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "missing value", data: "internal/app\n"},
		{name: "invalid value", data: "internal/app high\n"},
		{name: "out of range", data: "internal/app 100.1\n"},
		{name: "duplicate package", data: "internal/app 20.3\ninternal/app 20.4\n"},
		{name: "empty", data: "# no entries\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseMinimums([]byte(test.data)); err == nil {
				t.Error("parseMinimums() error = nil, want error")
			}
		})
	}
}

func TestCoverageIssueFor(t *testing.T) {
	tests := []struct {
		name    string
		actual  float64
		minimum float64
		want    coverageIssue
	}{
		{name: "below minimum", actual: 20.2, minimum: 20.3, want: coverageBelowMinimum},
		{name: "at minimum", actual: 20.3, minimum: 20.3, want: coverageOK},
		{name: "within drift allowance", actual: 21.3, minimum: 20.3, want: coverageOK},
		{name: "drift", actual: 21.4, minimum: 20.3, want: coverageDrift},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := coverageIssueFor(test.actual, test.minimum); got != test.want {
				t.Errorf("coverageIssueFor(%v, %v) = %d, want %d", test.actual, test.minimum, got, test.want)
			}
		})
	}
}
