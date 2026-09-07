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

type TaskStatus string

const (
	TaskStatusNotStarted TaskStatus = "not_started"
	TaskStatusDesigning  TaskStatus = "designing"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusClosed     TaskStatus = "closed"
)

// TaskDisplayState は task につき 1 値で、作業がどこまで進んだかだけを表す。
// 進行を妨げている事情は TaskBlockLabel が別に持つ。docs/design/domain.md を参照。
type TaskDisplayState string

const (
	TaskDisplayStateNotStarted  TaskDisplayState = "not_started"
	TaskDisplayStateDesigning   TaskDisplayState = "designing"
	TaskDisplayStateDesigned    TaskDisplayState = "designed"
	TaskDisplayStateInProgress  TaskDisplayState = "in_progress"
	TaskDisplayStateImplemented TaskDisplayState = "implemented"
	TaskDisplayStateInReview    TaskDisplayState = "in_review"
	TaskDisplayStateApproved    TaskDisplayState = "approved"
	TaskDisplayStateMerged      TaskDisplayState = "merged"
	TaskDisplayStateCompleted   TaskDisplayState = "completed"
	TaskDisplayStateClosed      TaskDisplayState = "closed"
	TaskDisplayStateUnknown     TaskDisplayState = "unknown"
)

// TaskBlockLabel は task の進行を妨げている事情で、1 つの task に 0〜4 個付く。
// 表示順はこの定数の並びに固定する。docs/design/domain.md を参照。
type TaskBlockLabel string

const (
	TaskBlockLabelDependencyUnresolved TaskBlockLabel = "dependency_unresolved"
	TaskBlockLabelConflict             TaskBlockLabel = "conflict"
	TaskBlockLabelChangesRequested     TaskBlockLabel = "changes_requested"
	TaskBlockLabelCIFailed             TaskBlockLabel = "ci_failed"
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

// CheckState は最新コミットのステータスチェックを 1 値に畳んだもの。個々のチェック名や
// 実行 URL は保持しない。docs/design/github-sync.md を参照。
type CheckState string

const (
	CheckStateUnknown CheckState = "unknown"
	CheckStateNone    CheckState = "none"
	CheckStatePending CheckState = "pending"
	CheckStateSuccess CheckState = "success"
	CheckStateFailure CheckState = "failure"
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

// Project は feature をまとめる。feature は必ずいずれかに属し、project が持つ状態は
// archived かどうかだけで、feature のような 2 層の status は持たない。
type Project struct {
	ID          string    `json:"id"`
	StorageID   string    `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Archived    bool      `json:"archived"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectUpdate は project 更新で変更しうる全フィールドを運ぶ。nil ポインタは省略、空文字は
// クリア要求を表す。1 つの比較可能な値にまとめることで、archive の防壁は個々のフィールドを
// 挙げずに変更の有無を判定できる。
type ProjectUpdate struct {
	Title       *string
	Description *string
	Archived    *bool
}

// FeatureUpdate は feature 更新で変更しうる全フィールドを運ぶ。ポインタの規約と、
// 1 つの値にまとめる理由は ProjectUpdate と同じ。
type FeatureUpdate struct {
	Title       *string
	Description *string
	Status      *FeatureStatus
	Archived    *bool
	ProjectID   *string
}

type Feature struct {
	ID          string        `json:"id"`
	StorageID   string        `json:"-"`
	ProjectID   string        `json:"project_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      FeatureStatus `json:"status"`
	// ReadOnly は導出値で、feature 自身か所属 project が archived であることを表す。
	// クライアントは feature と project のフラグを組み合わせず、この値から読み取り専用
	// 状態を表示する。
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
	// PendingBlockerTaskIDs は未解消の blocker をすべて挙げる。BlockerTaskID は
	// blocked reason の文言の元になった最初の 1 件だけを指す。
	// docs/design/domain.md を参照。
	PendingBlockerTaskIDs []string `json:"-"`
	// BlockLabels は DisplayState とは独立に評価する。JSON では空でも null にせず
	// 空配列を出す。docs/design/cli-contract.md を参照。
	BlockLabels []TaskBlockLabel `json:"block_labels"`
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
	// ReviewRequestPending は未応答のレビュー依頼が残っているかを、ReviewState の
	// 畳み込みとは独立に持つ。docs/design/github-sync.md を参照。
	ReviewRequestPending bool `json:"review_request_pending"`
	// ChangesRequestedAt は有効な変更要求レビューの最新提出時刻。
	ChangesRequestedAt *time.Time `json:"changes_requested_at,omitempty"`
	// LastPushedAt は最新コミットの push 時刻。
	LastPushedAt *time.Time `json:"last_pushed_at,omitempty"`
	// CheckState は最新コミットのステータスチェックのロールアップ 1 値。
	CheckState CheckState `json:"check_state"`
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

// DocumentParent は document の唯一の所有者を指す。値を持てるフィールドはちょうど 1 つで、
// それ以外の組み合わせは Count を使って呼び出し側が弾く。1 つの値にまとめることで、
// 呼び出し箇所で 3 つの識別子を取り違えずに済む。
type DocumentParent struct {
	ProjectID string
	FeatureID string
	TaskID    string
}

// Count は値が指し示す親の数を返す。
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
