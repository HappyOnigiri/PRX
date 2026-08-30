package domain

import (
	"encoding/json"
	"testing"
)

func TestTypedValuesKeepTheirJSONStrings(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"feature status", struct {
			Status FeatureStatus `json:"status"`
		}{FeatureStatusPaused}, `{"status":"paused"}`},
		{"task kind", struct {
			Kind TaskKind `json:"kind"`
		}{TaskKindManual}, `{"kind":"manual"}`},
		{"task status", struct {
			Status TaskStatus `json:"status"`
		}{TaskStatusInProgress}, `{"status":"in_progress"}`},
		{"pull request state", struct {
			State PullRequestState `json:"state"`
		}{PullRequestStateMerged}, `{"state":"merged"}`},
		{"document kind", struct {
			Kind DocumentKind `json:"kind"`
		}{DocumentKindLocalFile}, `{"kind":"local_file"}`},
		{"domain error code", struct {
			Code DomainErrorCode `json:"code"`
		}{DomainErrorCodeCycle}, `{"code":"cycle"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("JSON=%s, want %s", got, test.want)
			}
		})
	}
}
