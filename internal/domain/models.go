package domain

import "time"

type FeatureStatus string

const (
	FeatureStatusActive    FeatureStatus = "active"
	FeatureStatusPaused    FeatureStatus = "paused"
	FeatureStatusCompleted FeatureStatus = "completed"
	FeatureStatusCancelled FeatureStatus = "cancelled"
)

type TaskKind string

const (
	TaskKindPR          TaskKind = "pr"
	TaskKindManual      TaskKind = "manual"
	TaskKindPullRequest TaskKind = TaskKindPR
)

type TaskStatus string

const (
	TaskStatusAuto       TaskStatus = "auto"
	TaskStatusNotStarted TaskStatus = "not_started"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusClosed     TaskStatus = "closed"
)

type TaskDisplayState string

const (
	TaskDisplayStateNotStarted       TaskDisplayState = "not_started"
	TaskDisplayStateInProgress       TaskDisplayState = "in_progress"
	TaskDisplayStateCompleted        TaskDisplayState = "completed"
	TaskDisplayStateClosed           TaskDisplayState = "closed"
	TaskDisplayStateMerged           TaskDisplayState = "merged"
	TaskDisplayStateDraft            TaskDisplayState = "draft"
	TaskDisplayStateConflict         TaskDisplayState = "conflict"
	TaskDisplayStateChangesRequested TaskDisplayState = "changes_requested"
	TaskDisplayStateApproved         TaskDisplayState = "approved"
	TaskDisplayStateReviewWaiting    TaskDisplayState = "review_waiting"
	TaskDisplayStateOpen             TaskDisplayState = "open"
	TaskDisplayStateUnknown          TaskDisplayState = "unknown"
)

type PullRequestState string

const (
	PullRequestStateOpen    PullRequestState = "open"
	PullRequestStateClosed  PullRequestState = "closed"
	PullRequestStateMerged  PullRequestState = "merged"
	PullRequestStateUnknown PullRequestState = "unknown"
)

type ReviewState string

const (
	ReviewStateNone             ReviewState = "none"
	ReviewStateRequired         ReviewState = "required"
	ReviewStateApproved         ReviewState = "approved"
	ReviewStateChangesRequested ReviewState = "changes_requested"
	ReviewStateUnknown          ReviewState = "unknown"
)

type Mergeability string

const (
	MergeabilityMergeable   Mergeability = "mergeable"
	MergeabilityConflicting Mergeability = "conflicting"
	MergeabilityUnknown     Mergeability = "unknown"
)

type PullRequestDisplayState string

const (
	PullRequestDisplayStateMerged           PullRequestDisplayState = "merged"
	PullRequestDisplayStateClosed           PullRequestDisplayState = "closed"
	PullRequestDisplayStateDraft            PullRequestDisplayState = "draft"
	PullRequestDisplayStateConflict         PullRequestDisplayState = "conflict"
	PullRequestDisplayStateChangesRequested PullRequestDisplayState = "changes_requested"
	PullRequestDisplayStateApproved         PullRequestDisplayState = "approved"
	PullRequestDisplayStateReviewWaiting    PullRequestDisplayState = "review_waiting"
	PullRequestDisplayStateOpen             PullRequestDisplayState = "open"
	PullRequestDisplayStateUnknown          PullRequestDisplayState = "unknown"
)

type DocumentKind string

const (
	DocumentKindURL       DocumentKind = "url"
	DocumentKindLocalFile DocumentKind = "local_file"
	DocumentKindMarkdown  DocumentKind = "markdown"
)

type BlockedReasonCode string

const (
	BlockedReasonCodeDependencyDataIncomplete BlockedReasonCode = "dependency_data_incomplete"
	BlockedReasonCodeWaitingForBlocker        BlockedReasonCode = "waiting_for_blocker"
)

type Feature struct {
	ID                 string        `json:"id"`
	StorageID          string        `json:"-"`
	Slug               string        `json:"slug"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	Status             FeatureStatus `json:"status"`
	Archived           bool          `json:"archived"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	TaskCount          int           `json:"task_count,omitempty"`
	ReadyCount         int           `json:"ready_count,omitempty"`
	ReviewWaitingCount int           `json:"review_waiting_count,omitempty"`
	ConflictCount      int           `json:"conflict_count,omitempty"`
	MergedCount        int           `json:"merged_count,omitempty"`
}

type Task struct {
	ID                    string            `json:"id"`
	StorageID             string            `json:"-"`
	FeatureID             string            `json:"feature_id"`
	StorageFeatureID      string            `json:"-"`
	Title                 string            `json:"title"`
	Scope                 string            `json:"scope"`
	Kind                  TaskKind          `json:"kind"`
	Status                TaskStatus        `json:"status"`
	Assignee              string            `json:"assignee"`
	HasImplementationPlan bool              `json:"has_implementation_plan"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
	Ready                 bool              `json:"ready"`
	DisplayState          TaskDisplayState  `json:"display_state"`
	BlockedReason         string            `json:"blocked_reason,omitempty"`
	BlockedCode           BlockedReasonCode `json:"-"`
	BlockerTaskID         string            `json:"-"`
}

type Dependency struct {
	BlockerTaskID string    `json:"blocker_task_id"`
	BlockedTaskID string    `json:"blocked_task_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type PullRequest struct {
	TaskID          string                  `json:"task_id"`
	Host            string                  `json:"host"`
	Owner           string                  `json:"owner"`
	Repository      string                  `json:"repository"`
	Number          int64                   `json:"number"`
	URL             string                  `json:"url"`
	NodeID          string                  `json:"node_id"`
	Author          string                  `json:"author"`
	Assignees       []string                `json:"assignees"`
	State           PullRequestState        `json:"state"`
	Draft           bool                    `json:"draft"`
	ReviewState     ReviewState             `json:"review_state"`
	Mergeability    Mergeability            `json:"mergeability"`
	GitHubUpdatedAt *time.Time              `json:"github_updated_at,omitempty"`
	LastSyncedAt    *time.Time              `json:"last_synced_at,omitempty"`
	SyncError       string                  `json:"sync_error,omitempty"`
	Stale           bool                    `json:"stale"`
	DisplayState    PullRequestDisplayState `json:"display_state"`
}

type GitHubSyncState struct {
	LastAttemptAt   *time.Time `json:"last_attempt_at,omitempty"`
	LastCompletedAt *time.Time `json:"last_updated_at,omitempty"`
	Succeeded       int        `json:"succeeded"`
	Failed          int        `json:"failed"`
	Error           string     `json:"error,omitempty"`
}

type GitHubSyncStatus struct {
	IntervalSeconds int64      `json:"interval_seconds"`
	LastAttemptAt   *time.Time `json:"last_attempt_at"`
	LastUpdatedAt   *time.Time `json:"last_updated_at"`
	Succeeded       int        `json:"succeeded"`
	Failed          int        `json:"failed"`
	Error           string     `json:"error,omitempty"`
}

type Document struct {
	ID                   string       `json:"id"`
	FeatureID            string       `json:"feature_id,omitempty"`
	TaskID               string       `json:"task_id,omitempty"`
	Kind                 DocumentKind `json:"kind"`
	Title                string       `json:"title"`
	Locator              string       `json:"locator,omitempty"`
	Content              string       `json:"content,omitempty"`
	IsImplementationPlan bool         `json:"is_implementation_plan"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
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
