package domain

import "time"

type FeatureStatus string

const (
	FeatureStatusAuto      FeatureStatus = "auto"
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

// Project groups features. Membership is optional, and a project's only state
// is whether it is archived: it has no two-layer status the way a feature does.
type Project struct {
	ID          string    `json:"id"`
	StorageID   string    `json:"-"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Archived    bool      `json:"archived"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Feature struct {
	ID          string        `json:"id"`
	StorageID   string        `json:"-"`
	ProjectID   string        `json:"project_id,omitempty"`
	Slug        string        `json:"slug"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      FeatureStatus `json:"status"`
	// ReadOnly is derived: the feature is archived, or the project it belongs to
	// is. Clients present read-only state from this value instead of combining
	// the feature's own flag with its project's.
	ReadOnly           bool          `json:"read_only"`
	DisplayStatus      FeatureStatus `json:"display_status"`
	Archived           bool          `json:"archived"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	TaskCount          int           `json:"task_count,omitempty"`
	ReadyCount         int           `json:"ready_count,omitempty"`
	ReviewWaitingCount int           `json:"review_waiting_count,omitempty"`
	ConflictCount      int           `json:"conflict_count,omitempty"`
	MergedCount        int           `json:"merged_count,omitempty"`
	FinishedCount      int           `json:"finished_count,omitempty"`
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

// DocumentParent names the single owner of a document. Exactly one field may
// carry a value; Count lets callers reject the other combinations without
// spelling the condition out at each call site. Passing the three identifiers
// as one value also keeps them from being swapped at a call site.
type DocumentParent struct {
	ProjectID string
	FeatureID string
	TaskID    string
}

// Count reports how many parents the value names.
func (p DocumentParent) Count() int {
	count := 0
	for _, value := range []string{p.ProjectID, p.FeatureID, p.TaskID} {
		if value != "" {
			count++
		}
	}
	return count
}

type Document struct {
	ID                   string       `json:"id"`
	ProjectID            string       `json:"project_id,omitempty"`
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
	Projects           []Project     `json:"projects"`
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
