package domain

import (
	"fmt"
	"testing"
	"time"
)

func TestCyclePathAndTopologicalOrder(t *testing.T) {
	tasks := []Task{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	deps := []Dependency{{BlockerTaskID: "a", BlockedTaskID: "b"}, {BlockerTaskID: "a", BlockedTaskID: "c"}, {BlockerTaskID: "b", BlockedTaskID: "d"}, {BlockerTaskID: "c", BlockedTaskID: "d"}}
	order, err := TopologicalOrder(tasks, deps)
	if err != nil || len(order) != 4 || order[0] != "a" || order[3] != "d" {
		t.Fatalf("unexpected order %v, err=%v", order, err)
	}
	path := CyclePath(tasks, deps, "d", "a")
	if len(path) < 3 || path[0] != path[len(path)-1] {
		t.Fatalf("expected cycle path, got %v", path)
	}
}

func TestReadyFailsClosed(t *testing.T) {
	tasks := []Task{{ID: "a", Title: "API", Kind: TaskKindPR, Status: TaskInProgress}, {ID: "b", Title: "UI", Kind: TaskKindPR, Status: TaskPlanned}}
	deps := []Dependency{{BlockerTaskID: "a", BlockedTaskID: "b"}}
	prs := []PullRequest{{TaskID: "a", State: PullRequestStateMerged, Stale: true}}
	got := Derive(tasks, deps, prs)
	if got[1].Ready || got[1].BlockedReason == "" || got[1].BlockedCode != BlockedByStaleData || got[1].BlockerTaskID != "a" {
		t.Fatalf("stale merged blocker must fail closed: %+v", got[1])
	}
	prs[0].Stale = false
	got = Derive(tasks, deps, prs)
	if !got[1].Ready {
		t.Fatalf("fresh merged blocker should make task ready: %+v", got[1])
	}
}

func TestReadyReportsStructuredWaitingReason(t *testing.T) {
	tasks := []Task{{ID: "a", Title: "API", Kind: TaskKindManual, Status: TaskPlanned}, {ID: "b", Title: "UI", Kind: TaskKindManual, Status: TaskPlanned}}
	deps := []Dependency{{BlockerTaskID: "a", BlockedTaskID: "b"}}
	got := Derive(tasks, deps, nil)
	if got[1].Ready || got[1].BlockedCode != BlockedWaitingForBlocker || got[1].BlockerTaskID != "a" {
		t.Fatalf("unexpected structured waiting reason: %+v", got[1])
	}
}

func TestBlockedReasonAndCodeAreSetTogether(t *testing.T) {
	blocked := Task{ID: "b", Title: "UI", Kind: TaskKindManual, Status: TaskPlanned}
	blocker := Task{ID: "a", Title: "API", Kind: TaskKindPR, Status: TaskInProgress}
	cases := []struct {
		name  string
		tasks []Task
		deps  []Dependency
		prs   []PullRequest
		code  BlockedReasonCode
	}{
		{"dependency data incomplete", []Task{blocked}, []Dependency{{BlockerTaskID: "missing", BlockedTaskID: "b"}}, nil, BlockedReasonCodeDependencyDataIncomplete},
		{"blocker stale", []Task{blocker, blocked}, []Dependency{{BlockerTaskID: "a", BlockedTaskID: "b"}}, []PullRequest{{TaskID: "a", State: PullRequestStateMerged, Stale: true}}, BlockedReasonCodeBlockerStale},
		{"waiting for blocker", []Task{blocker, blocked}, []Dependency{{BlockerTaskID: "a", BlockedTaskID: "b"}}, nil, BlockedReasonCodeWaitingForBlocker},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := Derive(testCase.tasks, testCase.deps, testCase.prs)
			task := got[len(got)-1]
			if task.BlockedCode != testCase.code {
				t.Fatalf("blocked code=%q want %q", task.BlockedCode, testCase.code)
			}
			blockerTitle := ""
			if task.BlockerTaskID == blocker.ID {
				blockerTitle = blocker.Title
			}
			want := BlockedReasonText(testCase.code, blockerTitle)
			if want == "" {
				t.Fatalf("no wording defined for code %q", testCase.code)
			}
			if task.BlockedReason != want {
				t.Fatalf("blocked reason=%q want %q", task.BlockedReason, want)
			}
		})
	}
}

func TestPRDisplayPriority(t *testing.T) {
	pr := &PullRequest{State: PullRequestStateMerged, Draft: true, Mergeability: MergeabilityConflicting, ReviewState: ReviewStateChangesRequested}
	if got := PRDisplayState(pr); got != "merged" {
		t.Fatalf("merged must win, got %q", got)
	}
	pr.State = PullRequestStateOpen
	if got := PRDisplayState(pr); got != "draft" {
		t.Fatalf("draft must precede conflict, got %q", got)
	}
}

func TestDependencySatisfactionMatrix(t *testing.T) {
	tests := []struct {
		name string
		task Task
		pr   *PullRequest
		want bool
	}{
		{name: "manual completed", task: Task{Kind: TaskKindManual, Status: TaskCompleted}, want: true},
		{name: "manual cancelled", task: Task{Kind: TaskKindManual, Status: TaskCancelled}, want: false},
		{name: "PR merged", task: Task{Kind: TaskKindPR, Status: TaskInProgress}, pr: &PullRequest{State: PullRequestStateMerged}, want: true},
		{name: "PR merged but stale", task: Task{Kind: TaskKindPR, Status: TaskInProgress}, pr: &PullRequest{State: PullRequestStateMerged, Stale: true}, want: false},
		{name: "closed without merge", task: Task{Kind: TaskKindPR, Status: TaskInProgress}, pr: &PullRequest{State: PullRequestStateClosed}, want: false},
		{name: "missing PR", task: Task{Kind: TaskKindPR, Status: TaskPlanned}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSatisfied(test.task, test.pr); got != test.want {
				t.Fatalf("IsSatisfied()=%v want=%v", got, test.want)
			}
		})
	}
}

func TestSelfDependencyIsCycle(t *testing.T) {
	path := CyclePath([]Task{{ID: "task"}}, nil, "task", "task")
	if len(path) != 2 || path[0] != "task" || path[1] != "task" {
		t.Fatalf("self cycle path=%v", path)
	}
}

// A diamond chain merges and splits at every level, so a path-based search that
// does not remember settled nodes revisits shared subgraphs once per route.
func buildDiamondChain(levels int) ([]Task, []Dependency, string, string) {
	tasks := []Task{{ID: "a00"}}
	deps := []Dependency{}
	for level := 0; level < levels; level++ {
		left := fmt.Sprintf("l%02d", level)
		right := fmt.Sprintf("r%02d", level)
		next := fmt.Sprintf("a%02d", level+1)
		current := fmt.Sprintf("a%02d", level)
		tasks = append(tasks, Task{ID: left}, Task{ID: right}, Task{ID: next})
		deps = append(deps,
			Dependency{BlockerTaskID: current, BlockedTaskID: left},
			Dependency{BlockerTaskID: current, BlockedTaskID: right},
			Dependency{BlockerTaskID: left, BlockedTaskID: next},
			Dependency{BlockerTaskID: right, BlockedTaskID: next},
		)
	}
	return tasks, deps, fmt.Sprintf("a%02d", levels), "a00"
}

func TestCyclePathOnDiamondChainReturnsPromptly(t *testing.T) {
	tasks, deps, last, first := buildDiamondChain(40)
	result := make(chan []string, 1)
	go func() { result <- CyclePath(tasks, deps, last, first) }()
	select {
	case path := <-result:
		if len(path) < 3 || path[0] != path[len(path)-1] {
			t.Fatalf("expected a cycle path, got %v", path)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("CyclePath did not return within 5s on a 40-level diamond chain")
	}
}

func TestCyclePathClearsDiamondChainPromptly(t *testing.T) {
	tasks, deps, last, first := buildDiamondChain(40)
	result := make(chan []string, 1)
	// Adding an edge along the existing direction closes no cycle, so the search
	// has to settle the whole graph before it can answer.
	go func() { result <- CyclePath(tasks, deps, first, last) }()
	select {
	case path := <-result:
		if path != nil {
			t.Fatalf("edge along the existing direction is not a cycle: %v", path)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("CyclePath did not return within 5s on a 40-level diamond chain")
	}
}
