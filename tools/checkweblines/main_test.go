package main

import "testing"

func TestCountLines(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    int
	}{
		{name: "empty", content: "", want: 0},
		{name: "newline terminated", content: "first\nsecond\n", want: 2},
		{name: "not newline terminated", content: "first\nsecond", want: 2},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := countLines([]byte(testCase.content)); got != testCase.want {
				t.Fatalf("countLines() = %d, want %d", got, testCase.want)
			}
		})
	}
}
