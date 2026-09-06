package domain

import (
	"fmt"
	"sort"
)

func CyclePath(tasks []Task, deps []Dependency, blocker, blocked string) []string {
	if blocker == blocked {
		return []string{blocker, blocker}
	}
	// The stored graph is already acyclic, so adding blocker→blocked closes a
	// cycle exactly when blocker is reachable from blocked. Walking that single
	// question with a visited set keeps the search linear in the graph size;
	// re-running a path-based DFS from every task revisits shared subgraphs once
	// per distinct route, which is exponential on diamond-shaped graphs.
	adj := make(map[string][]string, len(tasks))
	for _, dep := range deps {
		adj[dep.BlockerTaskID] = append(adj[dep.BlockerTaskID], dep.BlockedTaskID)
	}
	for key := range adj {
		sort.Strings(adj[key])
	}
	parent := make(map[string]string, len(tasks))
	visited := map[string]bool{blocked: true}
	stack := []string{blocked}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if node == blocker {
			path := []string{}
			for current := node; ; current = parent[current] {
				path = append(path, current)
				if current == blocked {
					break
				}
			}
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return append(path, blocked)
		}
		next := adj[node]
		for i := len(next) - 1; i >= 0; i-- {
			if visited[next[i]] {
				continue
			}
			visited[next[i]] = true
			parent[next[i]] = node
			stack = append(stack, next[i])
		}
	}
	return nil
}

func TopologicalOrder(tasks []Task, deps []Dependency) ([]string, error) {
	indegree := make(map[string]int, len(tasks))
	adj := make(map[string][]string, len(tasks))
	for _, task := range tasks {
		indegree[task.ID] = 0
	}
	for _, dep := range deps {
		if _, ok := indegree[dep.BlockerTaskID]; !ok {
			return nil, fmt.Errorf("unknown blocker %s", dep.BlockerTaskID)
		}
		if _, ok := indegree[dep.BlockedTaskID]; !ok {
			return nil, fmt.Errorf("unknown blocked task %s", dep.BlockedTaskID)
		}
		indegree[dep.BlockedTaskID]++
		adj[dep.BlockerTaskID] = append(adj[dep.BlockerTaskID], dep.BlockedTaskID)
	}
	queue := make([]string, 0, len(tasks))
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)
	order := make([]string, 0, len(tasks))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, next := range adj[id] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
				sort.Strings(queue)
			}
		}
	}
	if len(order) != len(tasks) {
		return nil, fmt.Errorf("graph contains a cycle")
	}
	return order, nil
}

// BlockedReasonText renders the CLI-facing wording from the structured reason,
// so the JSON output and the code carried over RPC always describe the same
// blocker.
func BlockedReasonText(code BlockedReasonCode, blockerTitle string) string {
	switch code {
	case BlockedReasonCodeDependencyDataIncomplete:
		return "dependency data is incomplete"
	case BlockedReasonCodeWaitingForBlocker:
		return "waiting for " + blockerTitle
	default:
		return ""
	}
}

func Derive(tasks []Task, deps []Dependency, prs []PullRequest) []Task {
	prByTask := make(map[string]*PullRequest, len(prs))
	for i := range prs {
		prByTask[prs[i].TaskID] = &prs[i]
	}
	blockers := make(map[string][]string)
	for _, dep := range deps {
		blockers[dep.BlockedTaskID] = append(blockers[dep.BlockedTaskID], dep.BlockerTaskID)
	}
	taskByID := make(map[string]Task, len(tasks))
	for _, task := range tasks {
		taskByID[task.ID] = task
	}
	result := make([]Task, len(tasks))
	for i, task := range tasks {
		pr := prByTask[task.ID]
		task.Ready = false
		task.BlockedReason = ""
		task.BlockedCode = ""
		task.BlockerTaskID = ""
		if task.Status != TaskStatusAuto {
			switch task.Status {
			case TaskStatusAuto:
				task.DisplayState = TaskDisplayStateNotStarted
			case TaskStatusNotStarted:
				task.DisplayState = TaskDisplayStateNotStarted
			case TaskStatusInProgress:
				task.DisplayState = TaskDisplayStateInProgress
			case TaskStatusCompleted:
				task.DisplayState = TaskDisplayStateCompleted
			case TaskStatusClosed:
				task.DisplayState = TaskDisplayStateClosed
			default:
				task.DisplayState = TaskDisplayStateNotStarted
			}
		} else if pr != nil {
			task.DisplayState = PRDisplayState(pr)
		} else if task.HasImplementationPlan {
			task.DisplayState = TaskDisplayStateDesigned
		} else {
			task.DisplayState = TaskDisplayStateNotStarted
		}
		if !isReadyCandidate(task, task.DisplayState) {
			result[i] = task
			continue
		}
		task.Ready = true
		for _, blockerID := range blockers[task.ID] {
			blocker, ok := taskByID[blockerID]
			if !ok {
				task.Ready = false
				task.BlockedCode = BlockedReasonCodeDependencyDataIncomplete
				break
			}
			blockerPR := prByTask[blockerID]
			if !IsSatisfied(blocker, blockerPR) {
				task.Ready = false
				task.BlockedCode = BlockedReasonCodeWaitingForBlocker
				task.BlockerTaskID = blockerID
				break
			}
		}
		task.BlockedReason = BlockedReasonText(task.BlockedCode, taskByID[task.BlockerTaskID].Title)
		result[i] = task
	}
	return result
}
