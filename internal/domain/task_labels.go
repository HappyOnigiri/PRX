package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TaskLabelKey は task の表示状態またはブロック理由を示す設定キー。
type TaskLabelKey string

const (
	TaskLabelStatusNotStarted  TaskLabelKey = "status.not_started"
	TaskLabelStatusDesigning   TaskLabelKey = "status.designing"
	TaskLabelStatusDesigned    TaskLabelKey = "status.designed"
	TaskLabelStatusInProgress  TaskLabelKey = "status.in_progress"
	TaskLabelStatusImplemented TaskLabelKey = "status.implemented"
	TaskLabelStatusInReview    TaskLabelKey = "status.in_review"
	TaskLabelStatusApproved    TaskLabelKey = "status.approved"
	TaskLabelStatusCompleted   TaskLabelKey = "status.completed"
	TaskLabelStatusMerged      TaskLabelKey = "status.merged"
	TaskLabelStatusClosed      TaskLabelKey = "status.closed"
	TaskLabelStatusUnknown     TaskLabelKey = "status.unknown"

	TaskLabelBlockDependencyUnresolved TaskLabelKey = "block.dependency_unresolved"
	TaskLabelBlockConflict             TaskLabelKey = "block.conflict"
	TaskLabelBlockChangesRequested     TaskLabelKey = "block.changes_requested"
	TaskLabelBlockCIFailed             TaskLabelKey = "block.ci_failed"
	TaskLabelBlockUnknown              TaskLabelKey = "block.unknown"
)

var taskLabelKeys = []TaskLabelKey{
	TaskLabelStatusNotStarted,
	TaskLabelStatusDesigning,
	TaskLabelStatusDesigned,
	TaskLabelStatusInProgress,
	TaskLabelStatusImplemented,
	TaskLabelStatusInReview,
	TaskLabelStatusApproved,
	TaskLabelStatusCompleted,
	TaskLabelStatusMerged,
	TaskLabelStatusClosed,
	TaskLabelStatusUnknown,
	TaskLabelBlockDependencyUnresolved,
	TaskLabelBlockConflict,
	TaskLabelBlockChangesRequested,
	TaskLabelBlockCIFailed,
	TaskLabelBlockUnknown,
}

// TaskLabelKeys は設定画面と CLI が共有する決定的なキー一覧を返す。
func TaskLabelKeys() []TaskLabelKey { return append([]TaskLabelKey(nil), taskLabelKeys...) }

// TaskLabelOverride は scope が明示した表示項目を保持する。空の項目は保存しない。
type TaskLabelOverride struct {
	Text  string `json:"text,omitempty"  yaml:"text,omitempty"`
	Color string `json:"color,omitempty" yaml:"color,omitempty"`
}

type TaskLabelOverrides map[TaskLabelKey]TaskLabelOverride

// TaskLabelOverrideUpdate は項目単位の部分更新を表す。nil は変更なしを意味する。
type TaskLabelOverrideUpdate struct {
	Text  *string
	Color *string
}

type TaskLabelOverridesUpdate map[TaskLabelKey]TaskLabelOverrideUpdate

// TaskLabelAppearance は解決済み表示値。TextOverridden は分割表示を解除するか
// を、ColorOverridden は編集画面の継承元表示を判断するために使う。
type TaskLabelAppearance struct {
	Text            string `json:"text"`
	Color           string `json:"color"`
	TextOverridden  bool   `json:"text_overridden,omitempty"`
	ColorOverridden bool   `json:"color_overridden,omitempty"`
}

type TaskLabelAppearances map[TaskLabelKey]TaskLabelAppearance

var taskLabelColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// NormalizeTaskLabelOverride は設定値を検証・正規化する。空文字列は解除として扱う。
func NormalizeTaskLabelOverride(value TaskLabelOverride) (TaskLabelOverride, error) {
	text, err := NormalizeTaskLabelText(value.Text)
	if err != nil {
		return TaskLabelOverride{}, err
	}
	color, err := NormalizeTaskLabelColor(value.Color)
	if err != nil {
		return TaskLabelOverride{}, err
	}
	return TaskLabelOverride{Text: text, Color: color}, nil
}

func NormalizeTaskLabelText(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("text must be valid UTF-8")
	}
	for _, runeValue := range value {
		if runeValue == '\n' || runeValue == '\r' || unicode.IsControl(runeValue) {
			return "", fmt.Errorf("text must not contain newlines or control characters")
		}
	}
	if utf8.RuneCountInString(value) > 32 {
		return "", fmt.Errorf("text must be at most 32 Unicode characters")
	}
	return value, nil
}

func NormalizeTaskLabelColor(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !taskLabelColorPattern.MatchString(value) {
		return "", fmt.Errorf("color must be a #RRGGBB value")
	}
	return strings.ToLower(value), nil
}

// ValidateTaskLabelOverrides は未知キーと個別値をまとめて検証する。
func ValidateTaskLabelOverrides(values TaskLabelOverrides) error {
	allowed := make(map[TaskLabelKey]struct{}, len(taskLabelKeys))
	for _, key := range taskLabelKeys {
		allowed[key] = struct{}{}
	}
	for key, value := range values {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("unknown task label key %q", key)
		}
		normalized, err := NormalizeTaskLabelOverride(value)
		if err != nil {
			return fmt.Errorf("task label %s: %w", key, err)
		}
		if normalized == (TaskLabelOverride{}) {
			continue
		}
	}
	return nil
}

// BuiltInTaskLabelAppearances は未設定時の CLI と WebUI の安定値を返す。
func BuiltInTaskLabelAppearances() TaskLabelAppearances {
	result := make(TaskLabelAppearances, len(taskLabelKeys))
	for _, key := range taskLabelKeys {
		value := string(key)
		result[key] = TaskLabelAppearance{Text: value[strings.IndexByte(value, '.')+1:]}
	}
	colors := map[TaskLabelKey]string{
		TaskLabelStatusNotStarted:          "#0c715a",
		TaskLabelStatusDesigning:           "#0b6a8f",
		TaskLabelStatusDesigned:            "#0b6a8f",
		TaskLabelStatusInProgress:          "#0b6a8f",
		TaskLabelStatusImplemented:         "#0b6a8f",
		TaskLabelStatusInReview:            "#7b5100",
		TaskLabelStatusApproved:            "#8250df",
		TaskLabelStatusCompleted:           "#8250df",
		TaskLabelStatusMerged:              "#8250df",
		TaskLabelStatusClosed:              "#747d89",
		TaskLabelStatusUnknown:             "#747d89",
		TaskLabelBlockDependencyUnresolved: "#7b5100",
		TaskLabelBlockConflict:             "#a9322f",
		TaskLabelBlockChangesRequested:     "#a9322f",
		TaskLabelBlockCIFailed:             "#a9322f",
		TaskLabelBlockUnknown:              "#a9322f",
	}
	for key, color := range colors {
		appearance := result[key]
		appearance.Color = color
		result[key] = appearance
	}
	return result
}

// ResolveTaskLabelAppearances は global → project → feature の順で項目ごとに合成する。
func ResolveTaskLabelAppearances(global, project, feature TaskLabelOverrides) TaskLabelAppearances {
	result := BuiltInTaskLabelAppearances()
	for _, overrides := range []TaskLabelOverrides{global, project, feature} {
		for key, value := range overrides {
			appearance, ok := result[key]
			if !ok {
				continue
			}
			if value.Text != "" {
				appearance.Text = value.Text
				appearance.TextOverridden = true
			}
			if value.Color != "" {
				appearance.Color = value.Color
				appearance.ColorOverridden = true
			}
			result[key] = appearance
		}
	}
	return result
}

// SortedTaskLabelKeys は map を決定的に表示するために使う。
func SortedTaskLabelKeys(values TaskLabelOverrides) []TaskLabelKey {
	keys := make([]TaskLabelKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func TaskLabelKeyForDisplayState(state TaskDisplayState) TaskLabelKey {
	switch state {
	case TaskDisplayStateNotStarted,
		TaskDisplayStateDesigning,
		TaskDisplayStateDesigned,
		TaskDisplayStateInProgress,
		TaskDisplayStateImplemented,
		TaskDisplayStateInReview,
		TaskDisplayStateApproved,
		TaskDisplayStateCompleted,
		TaskDisplayStateMerged,
		TaskDisplayStateClosed:
		return TaskLabelKey("status." + string(state))
	case TaskDisplayStateUnknown:
		return TaskLabelStatusUnknown
	default:
		return TaskLabelStatusUnknown
	}
}

func TaskLabelKeyForBlockLabel(label TaskBlockLabel) TaskLabelKey {
	switch label {
	case TaskBlockLabelDependencyUnresolved,
		TaskBlockLabelConflict,
		TaskBlockLabelChangesRequested,
		TaskBlockLabelCIFailed:
		return TaskLabelKey("block." + string(label))
	default:
		return TaskLabelBlockUnknown
	}
}
