package domain

import "testing"

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
	prs := []PullRequest{{TaskID: "a", State: "merged", Stale: true}}
	got := Derive(tasks, deps, prs)
	if got[1].Ready || got[1].BlockedReason == "" {
		t.Fatalf("stale merged blocker must fail closed: %+v", got[1])
	}
	prs[0].Stale = false
	got = Derive(tasks, deps, prs)
	if !got[1].Ready {
		t.Fatalf("fresh merged blocker should make task ready: %+v", got[1])
	}
}

func TestPRDisplayPriority(t *testing.T) {
	pr := &PullRequest{State: "merged", Draft: true, Mergeability: "conflicting", ReviewState: "changes_requested"}
	if got := PRDisplayState(pr); got != "merged" {
		t.Fatalf("merged must win, got %q", got)
	}
	pr.State = "open"
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
		{name: "PR merged", task: Task{Kind: TaskKindPR, Status: TaskInProgress}, pr: &PullRequest{State: "merged"}, want: true},
		{name: "PR merged but stale", task: Task{Kind: TaskKindPR, Status: TaskInProgress}, pr: &PullRequest{State: "merged", Stale: true}, want: false},
		{name: "closed without merge", task: Task{Kind: TaskKindPR, Status: TaskInProgress}, pr: &PullRequest{State: "closed"}, want: false},
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
