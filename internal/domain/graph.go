package domain

import (
	"fmt"
	"sort"
)

func CyclePath(tasks []Task, deps []Dependency, blocker, blocked string) []string {
	if blocker == blocked {
		return []string{blocker, blocker}
	}
	adj := make(map[string][]string, len(tasks))
	for _, dep := range deps {
		adj[dep.BlockerTaskID] = append(adj[dep.BlockerTaskID], dep.BlockedTaskID)
	}
	adj[blocker] = append(adj[blocker], blocked)
	for key := range adj {
		sort.Strings(adj[key])
	}
	var visit func(string, []string, map[string]int) []string
	visit = func(node string, path []string, active map[string]int) []string {
		if start, ok := active[node]; ok {
			cycle := append([]string{}, path[start:]...)
			return append(cycle, node)
		}
		active[node] = len(path)
		path = append(path, node)
		for _, next := range adj[node] {
			if cycle := visit(next, path, active); len(cycle) > 0 {
				return cycle
			}
		}
		delete(active, node)
		return nil
	}
	for _, task := range tasks {
		if cycle := visit(task.ID, nil, map[string]int{}); len(cycle) > 0 {
			return cycle
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
		task.DisplayState = task.Status
		if task.Kind == TaskKindPR {
			task.DisplayState = PRDisplayState(pr)
		}
		if !IsIncomplete(task, pr) || task.Status != TaskPlanned {
			result[i] = task
			continue
		}
		task.Ready = true
		for _, blockerID := range blockers[task.ID] {
			blocker, ok := taskByID[blockerID]
			if !ok {
				task.Ready = false
				task.BlockedReason = "dependency data is incomplete"
				break
			}
			blockerPR := prByTask[blockerID]
			if blockerPR != nil && blockerPR.Stale {
				task.Ready = false
				task.BlockedReason = "a blocker has stale GitHub data"
				break
			}
			if !IsSatisfied(blocker, blockerPR) {
				task.Ready = false
				task.BlockedReason = "waiting for " + blocker.Title
				break
			}
		}
		result[i] = task
	}
	return result
}
