package domain

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

func TestCyclePathAndTopologicalOrder(t *testing.T) {
	tasks := []Task{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	deps := []Dependency{
		{BlockerTaskID: "a", BlockedTaskID: "b"},
		{BlockerTaskID: "a", BlockedTaskID: "c"},
		{BlockerTaskID: "b", BlockedTaskID: "d"},
		{BlockerTaskID: "c", BlockedTaskID: "d"},
	}
	order, err := TopologicalOrder(tasks, deps)
	if err != nil || len(order) != 4 || order[0] != "a" || order[3] != "d" {
		t.Fatalf("unexpected order %v, err=%v", order, err)
	}
	path := CyclePath(tasks, deps, "d", "a")
	if len(path) < 3 || path[0] != path[len(path)-1] {
		t.Fatalf("expected cycle path, got %v", path)
	}
}

func TestReadyUsesLastKnownPullRequestState(t *testing.T) {
	tasks := []Task{
		{ID: "a", Title: "API", Status: TaskStatusNotStarted},
		{ID: "b", Title: "UI", Status: TaskStatusNotStarted},
	}
	deps := []Dependency{{BlockerTaskID: "a", BlockedTaskID: "b"}}
	prs := []PullRequest{{TaskID: "a", State: PullRequestStateMerged, Stale: true}}
	got := Derive(tasks, deps, prs)
	if !got[1].Ready || got[1].BlockedReason != "" {
		t.Fatalf("stale merged blocker should remain satisfied: %+v", got[1])
	}
	prs[0].State = PullRequestStateUnknown
	got = Derive(tasks, deps, prs)
	if got[1].Ready || got[1].BlockedCode != BlockedReasonCodeWaitingForBlocker ||
		got[1].BlockerTaskID != "a" {
		t.Fatalf("unknown blocker should remain blocked: %+v", got[1])
	}
}

func TestReadyReportsStructuredWaitingReason(t *testing.T) {
	tasks := []Task{
		{ID: "a", Title: "API", Status: TaskStatusNotStarted},
		{ID: "b", Title: "UI", Status: TaskStatusNotStarted},
	}
	deps := []Dependency{{BlockerTaskID: "a", BlockedTaskID: "b"}}
	got := Derive(tasks, deps, nil)
	if got[1].Ready || got[1].BlockedCode != BlockedReasonCodeWaitingForBlocker || got[1].BlockerTaskID != "a" {
		t.Fatalf("unexpected structured waiting reason: %+v", got[1])
	}
}

// blocked なタスクをエージェントに渡す側は、待っている対象をすべて渡す必要がある。そのため
// 理由が最初の blocker だけを挙げる場合でも、集合全体を導出する。
func TestReadyCollectsEveryPendingBlocker(t *testing.T) {
	tasks := []Task{
		{ID: "a", Title: "API", Status: TaskStatusNotStarted},
		{ID: "b", Title: "Billing", Status: TaskStatusCompleted},
		{ID: "c", Title: "Schema", Status: TaskStatusNotStarted},
		{ID: "d", Title: "UI", Status: TaskStatusNotStarted},
	}
	deps := []Dependency{
		{BlockerTaskID: "a", BlockedTaskID: "d"},
		{BlockerTaskID: "b", BlockedTaskID: "d"},
		{BlockerTaskID: "c", BlockedTaskID: "d"},
		{BlockerTaskID: "missing", BlockedTaskID: "d"},
	}
	got := Derive(tasks, deps, nil)
	ui := got[3]
	if ui.Ready {
		t.Fatalf("task waiting on three blockers should not be ready: %+v", ui)
	}
	// "b" は completed なので解消済みとして除外する。グラフで解決できない blocker も
	// 解消済みではないため、"missing" は残す。
	want := []string{"a", "c", "missing"}
	if !slices.Equal(ui.PendingBlockerTaskIDs, want) {
		t.Fatalf("pending blockers=%v want %v", ui.PendingBlockerTaskIDs, want)
	}
	// 理由は依然として、未解消の最初の blocker だけを表す。
	if ui.BlockedCode != BlockedReasonCodeWaitingForBlocker || ui.BlockerTaskID != "a" {
		t.Fatalf("unexpected blocked reason: %+v", ui)
	}
	if ids := got[0].PendingBlockerTaskIDs; ids != nil {
		t.Fatalf("a task with no blocker should carry none: %v", ids)
	}
}

func TestBlockedReasonAndCodeAreSetTogether(t *testing.T) {
	blocked := Task{ID: "b", Title: "UI", Status: TaskStatusNotStarted}
	blocker := Task{ID: "a", Title: "API", Status: TaskStatusNotStarted}
	cases := []struct {
		name  string
		tasks []Task
		deps  []Dependency
		prs   []PullRequest
		code  BlockedReasonCode
	}{
		{
			"dependency data incomplete",
			[]Task{blocked},
			[]Dependency{{BlockerTaskID: "missing", BlockedTaskID: "b"}},
			nil,
			BlockedReasonCodeDependencyDataIncomplete,
		},
		{
			"waiting for blocker",
			[]Task{blocker, blocked},
			[]Dependency{{BlockerTaskID: "a", BlockedTaskID: "b"}},
			nil,
			BlockedReasonCodeWaitingForBlocker,
		},
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
	pr := &PullRequest{
		State:        PullRequestStateMerged,
		Draft:        true,
		Mergeability: MergeabilityConflicting,
		ReviewState:  ReviewStateChangesRequested,
	}
	if got := PRDisplayState(pr); got != "merged" {
		t.Fatalf("merged must win, got %q", got)
	}
	pr.State = PullRequestStateOpen
	if got := PRDisplayState(pr); got != "draft" {
		t.Fatalf("draft must precede conflict, got %q", got)
	}
}

func TestTaskDisplayStateMatrix(t *testing.T) {
	tests := []struct {
		name  string
		task  Task
		pr    []PullRequest
		want  TaskDisplayState
		ready bool
	}{
		{
			name:  "not started without plan",
			task:  Task{ID: "task", Status: TaskStatusNotStarted},
			want:  TaskDisplayStateNotStarted,
			ready: true,
		},
		{
			name:  "not started with plan",
			task:  Task{ID: "task", Status: TaskStatusNotStarted, HasImplementationPlan: true},
			want:  TaskDisplayStateDesigned,
			ready: true,
		},
		{
			name:  "designing without plan",
			task:  Task{ID: "task", Status: TaskStatusDesigning},
			want:  TaskDisplayStateDesigning,
			ready: true,
		},
		{
			name:  "designing yields to a registered plan",
			task:  Task{ID: "task", Status: TaskStatusDesigning, HasImplementationPlan: true},
			want:  TaskDisplayStateDesigned,
			ready: true,
		},
		{
			name:  "designing yields to a pull request",
			task:  Task{ID: "task", Status: TaskStatusDesigning},
			pr:    []PullRequest{{TaskID: "task", State: PullRequestStateOpen}},
			want:  TaskDisplayStateOpen,
			ready: false,
		},
		{
			name:  "not started with plan and pull request",
			task:  Task{ID: "task", Status: TaskStatusNotStarted, HasImplementationPlan: true},
			pr:    []PullRequest{{TaskID: "task", State: PullRequestStateOpen}},
			want:  TaskDisplayStateOpen,
			ready: false,
		},
		{
			name:  "in progress without pull request",
			task:  Task{ID: "task", Status: TaskStatusInProgress, HasImplementationPlan: true},
			want:  TaskDisplayStateInProgress,
			ready: false,
		},
		{
			name: "in progress yields to a pull request waiting for review",
			task: Task{ID: "task", Status: TaskStatusInProgress},
			pr: []PullRequest{{
				TaskID: "task", State: PullRequestStateOpen, ReviewState: ReviewStateRequired,
			}},
			want:  TaskDisplayStateReviewWaiting,
			ready: false,
		},
		{
			name:  "completed outranks an open pull request",
			task:  Task{ID: "task", Status: TaskStatusCompleted},
			pr:    []PullRequest{{TaskID: "task", State: PullRequestStateOpen}},
			want:  TaskDisplayStateCompleted,
			ready: false,
		},
		{
			name:  "closed outranks an open pull request",
			task:  Task{ID: "task", Status: TaskStatusClosed},
			pr:    []PullRequest{{TaskID: "task", State: PullRequestStateOpen}},
			want:  TaskDisplayStateClosed,
			ready: false,
		},
		{
			name:  "not started with an unknown pull request",
			task:  Task{ID: "task", Status: TaskStatusNotStarted},
			pr:    []PullRequest{{TaskID: "task", State: PullRequestStateUnknown}},
			want:  TaskDisplayStateUnknown,
			ready: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Derive([]Task{test.task}, nil, test.pr)[0]
			if got.DisplayState != test.want || got.Ready != test.ready {
				t.Fatalf("task=%+v, want display=%q ready=%v", got, test.want, test.ready)
			}
		})
	}
}

func TestDependencySatisfactionMatrix(t *testing.T) {
	tests := []struct {
		name string
		task Task
		pr   *PullRequest
		want bool
	}{
		{name: "completed", task: Task{Status: TaskStatusCompleted}, want: true},
		{name: "closed", task: Task{Status: TaskStatusClosed}, want: true},
		{name: "in progress without a PR", task: Task{Status: TaskStatusInProgress}, want: false},
		{
			name: "in progress with an open PR",
			task: Task{Status: TaskStatusInProgress},
			pr:   &PullRequest{State: PullRequestStateOpen},
			want: true,
		},
		{
			name: "PR open",
			task: Task{Status: TaskStatusNotStarted},
			pr:   &PullRequest{State: PullRequestStateOpen},
			want: true,
		},
		{
			name: "PR closed",
			task: Task{Status: TaskStatusNotStarted},
			pr:   &PullRequest{State: PullRequestStateClosed},
			want: true,
		},
		{
			name: "PR merged",
			task: Task{Status: TaskStatusNotStarted},
			pr:   &PullRequest{State: PullRequestStateMerged},
			want: true,
		},
		{
			name: "PR merged but stale",
			task: Task{Status: TaskStatusNotStarted},
			pr:   &PullRequest{State: PullRequestStateMerged, Stale: true},
			want: true,
		},
		{
			name: "unknown PR",
			task: Task{Status: TaskStatusNotStarted},
			pr:   &PullRequest{State: PullRequestStateUnknown, Stale: true},
			want: false,
		},
		{name: "missing PR", task: Task{Status: TaskStatusNotStarted}, want: false},
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

// ダイヤモンド連鎖は各段で合流と分岐を繰り返すため、確定済みノードを覚えない経路ベースの
// 探索は、共有する部分グラフを経路の数だけ再訪する。
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
	// 既存の向きに沿った辺を足しても閉路にならないため、探索は答えを出す前に
	// グラフ全体を調べ切る必要がある。
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
