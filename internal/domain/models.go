package domain

import "time"

const (
	TaskKindPR     = "pr"
	TaskKindManual = "manual"

	TaskPlanned    = "planned"
	TaskInProgress = "in_progress"
	TaskCompleted  = "completed"
	TaskCancelled  = "cancelled"
)

type Feature struct {
	ID                 string    `json:"id"`
	Slug               string    `json:"slug"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Status             string    `json:"status"`
	Archived           bool      `json:"archived"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	TaskCount          int       `json:"task_count,omitempty"`
	ReadyCount         int       `json:"ready_count,omitempty"`
	ReviewWaitingCount int       `json:"review_waiting_count,omitempty"`
	ConflictCount      int       `json:"conflict_count,omitempty"`
	MergedCount        int       `json:"merged_count,omitempty"`
}

type Task struct {
	ID            string    `json:"id"`
	FeatureID     string    `json:"feature_id"`
	Title         string    `json:"title"`
	Scope         string    `json:"scope"`
	Kind          string    `json:"kind"`
	Status        string    `json:"status"`
	Assignee      string    `json:"assignee"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Ready         bool      `json:"ready"`
	DisplayState  string    `json:"display_state"`
	BlockedReason string    `json:"blocked_reason,omitempty"`
}

type Dependency struct {
	BlockerTaskID string    `json:"blocker_task_id"`
	BlockedTaskID string    `json:"blocked_task_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type PullRequest struct {
	TaskID          string     `json:"task_id"`
	Owner           string     `json:"owner"`
	Repository      string     `json:"repository"`
	Number          int64      `json:"number"`
	URL             string     `json:"url"`
	NodeID          string     `json:"node_id"`
	Author          string     `json:"author"`
	Assignees       []string   `json:"assignees"`
	State           string     `json:"state"`
	Draft           bool       `json:"draft"`
	ReviewState     string     `json:"review_state"`
	Mergeability    string     `json:"mergeability"`
	GitHubUpdatedAt *time.Time `json:"github_updated_at,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	SyncError       string     `json:"sync_error,omitempty"`
	Stale           bool       `json:"stale"`
	DisplayState    string     `json:"display_state"`
}

type Document struct {
	ID        string    `json:"id"`
	FeatureID string    `json:"feature_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type Snapshot struct {
	Features           []Feature     `json:"features"`
	Tasks              []Task        `json:"tasks"`
	Dependencies       []Dependency  `json:"dependencies"`
	PullRequests       []PullRequest `json:"pull_requests"`
	Documents          []Document    `json:"documents"`
	ReadyTasks         []Task        `json:"ready_tasks"`
	ReviewWaitingTasks []Task        `json:"review_waiting_tasks"`
	ConflictTasks      []Task        `json:"conflict_tasks"`
	StaleTasks         []Task        `json:"stale_tasks"`
}
