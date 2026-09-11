// Package prompt は CLI・RPC サーバー・WebUI が共有するエージェント用プロンプト
// テンプレートを管理する。組み込みテンプレート、その検証、プレースホルダの置換、
// タスクごとにテンプレートを選ぶ規則を含む。
package prompt

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// MaximumTemplateBytes は保存されるテンプレート 1 件の上限。テンプレートは YAML・
// JSON・Protocol Buffers を経由し、他のエージェントの入力に貼るためのものなので、
// 本文が無制限でも誰の得にもならない。
const MaximumTemplateBytes = 8192

// Kind はタスクに必要なテンプレートを示す。タスクから導出される値であり、
// 保存されることはない。
type Kind string

const (
	// KindDesign はエージェントに実装計画の作成を依頼する。
	KindDesign Kind = "design"
	// KindImplementation はエージェントに登録済みの計画の実行を依頼する。
	KindImplementation Kind = "implementation"
)

// Templates は Kind ごとのテンプレートに加え、batch テンプレートを保持する。batch は
// 個々のタスクから導出されるのではなく、複数タスクをまとめて要求する呼び出し元が
// 選ぶものなので Kind ではない。
type Templates struct {
	Design         string `yaml:"design"         json:"design"`
	Implementation string `yaml:"implementation" json:"implementation"`
	Batch          string `yaml:"batch"          json:"batch"`
}

// placeholderPattern は未対応のものも含めて置換トークン全てに一致する。レンダラが
// 黙って残してしまう名前を検証で弾けるようにするため。
var placeholderPattern = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// requiredPlaceholder はテンプレートを 1 つのタスクに向け続ける。これがないと
// 生成されたプロンプトは対象を示さず、受け取ったエージェントが着手できない。
const requiredPlaceholder = "task_id"

// supportedPlaceholders は置換語彙の全体。計画の本文は意図的に含めない。計画は
// 最大 1 MiB になり得るしロケータの先にある場合もあるため、プロンプトでは
// 代わりに `prx plan TASK_ID` で読むようエージェントに指示する。
var supportedPlaceholders = []string{
	requiredPlaceholder,
	"feature_id",
	"task_title",
	"task_scope",
}

// batchRequiredPlaceholder は batch テンプレートを要求されたタスク群に向け続ける。
// これがないと生成されたプロンプトは対象を一切示さず、タスク用プロンプトが
// 識別子を欠くよりさらに悪い。
const batchRequiredPlaceholder = "task_list"

// batchSupportedPlaceholders は batch の語彙で、個々のタスクのタイトルやスコープは
// 含まない。
// docs/design/agent-prompts.md を参照。
var batchSupportedPlaceholders = []string{
	batchRequiredPlaceholder,
	"feature_id",
}

const defaultDesignTemplate = `Design PRX task {{task_id}} of feature {{feature_id}}.

Title: {{task_title}}
Scope: {{task_scope}}

PRX is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`prx --help`" + ` and ` + "`prx <command> --help`" + ` for its exact surface.
PRX runs on this machine only, so nobody reading the repository can see it.
Keep it out of what the repository carries: no code comment, commit message, or pull request
may mention PRX, its identifiers, or its commands.

1. Mark the task as being designed before anything else.
   - ` + "`prx task update {{task_id}} --status designing`" + `
2. Read the task and the work it depends on.
   - ` + "`prx task {{task_id}}`" + `
   - ` + "`prx graph {{feature_id}}`" + `
3. Read the reference material attached in PRX before you decide anything.
   Documents hang off the task, off its feature, and off the project that feature belongs to,
   and any of them may carry the requirements this scope has to meet.
   - ` + "`prx document --task {{task_id}}`" + ` and ` + "`prx document --feature {{feature_id}}`" + `
   - ` + "`prx feature {{feature_id}}`" + ` names the project, then ` + "`prx document --project PROJECT_ID`" + `
   - ` + "`prx document get DOCUMENT_ID`" + ` prints a stored document; one that points at a URL or a
     local file prints that locator instead, so open it yourself.
4. Investigate the repository and decide how the scope above should be built.
5. Register the resulting plan on the task.
   - ` + "`prx plan set {{task_id}} --file PLAN.md`" + `
   - ` + "`prx plan set {{task_id}} --stdin`" + `
   Registering the plan is what presents the task as designed, so leave the status alone afterwards.

Design only: leave the implementation and the pull request to the next step.
`

const defaultImplementationTemplate = `Implement PRX task {{task_id}} of feature {{feature_id}}.

Title: {{task_title}}
Scope: {{task_scope}}

PRX is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`prx --help`" + ` and ` + "`prx <command> --help`" + ` for its exact surface.
PRX runs on this machine only, so nobody reading the repository can see it.
Keep it out of what the repository carries: no code comment, commit message, or pull request
may mention PRX, its identifiers, or its commands.

1. Read the task, the work it depends on, and its registered plan.
   - ` + "`prx task {{task_id}}`" + `
   - ` + "`prx graph {{feature_id}}`" + `
   - ` + "`prx plan {{task_id}}`" + `
2. Read the reference material attached in PRX before you write any code.
   Documents hang off the task, off its feature, and off the project that feature belongs to,
   and any of them may carry the requirements this scope has to meet.
   - ` + "`prx document --task {{task_id}}`" + ` and ` + "`prx document --feature {{feature_id}}`" + `
   - ` + "`prx feature {{feature_id}}`" + ` names the project, then ` + "`prx document --project PROJECT_ID`" + `
   - ` + "`prx document get DOCUMENT_ID`" + ` prints a stored document; one that points at a URL or a
     local file prints that locator instead, so open it yourself.
3. Mark the task as being worked on before you change anything.
   - ` + "`prx task update {{task_id}} --status in_progress`" + `
4. Implement the plan, staying inside the scope above.
   Branch from the base the work actually belongs on rather than from main or master by default.
   A task whose blocker is still open belongs on that blocker's branch, so the two pull requests stack.
5. Record the result in PRX.
   - Work that lands as a pull request: open it against the base you branched from,
     then run ` + "`prx pr attach {{task_id}} PULL_REQUEST_URL`" + `.
     Its state then follows the pull request, so do not set the status by hand.
   - Work without a pull request: run ` + "`prx task update {{task_id}} --status completed`" + ` once it is done.

Report what you changed and anything the plan did not cover.
`

const defaultBatchTemplate = `Implement the PRX tasks of feature {{feature_id}} listed below.

PRX is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`prx --help`" + ` and ` + "`prx <command> --help`" + ` for its exact surface.
PRX runs on this machine only, so nobody reading the repository can see it.
Keep it out of what the repository carries, in your work and in every SubAgent's: no code comment,
commit message, or pull request may mention PRX, its identifiers, or its commands.

Tasks:
{{task_list}}

1. Read the feature graph so you know how the listed tasks relate to the rest of the work.
   - ` + "`prx graph {{feature_id}}`" + `
2. Hand every task to its own SubAgent: one task per SubAgent, and never two tasks to the same one.
   Each SubAgent takes its instructions from PRX rather than from you.
   - It runs ` + "`prx prompt TASK_ID`" + ` for the task it was given and follows the prompt that prints.
   - It reports what it changed and anything the prompt did not cover.
   Tasks the graph shows as independent may run in parallel.
   A task that depends on another task of this list is implemented after that task is finished,
   and its pull request is stacked on the pull request of the task it depends on.
   Branch from the base the work belongs on rather than from main or master by default.
3. Wait for every SubAgent and read what each one reported.
   A task whose SubAgent failed stays unfinished: report it instead of implementing it yourself.
   Whatever depends on it stays unstarted as well, because its base is not there.

Report each task's outcome separately, including the ones that failed.
`

// SupportedPlaceholders は置換語彙を波括弧なしで返す。クライアントが独自の一覧を
// 抱えて黙って乖離するのではなく、サーバーが受け付ける内容を提示できるようにある。
func SupportedPlaceholders() []string {
	return slices.Clone(supportedPlaceholders)
}

// RequiredPlaceholder はすべてのタスクテンプレートが使うべきプレースホルダを返す。
func RequiredPlaceholder() string { return requiredPlaceholder }

// BatchSupportedPlaceholders は batch の置換語彙を返す。batch テンプレートがタスク用
// プレースホルダを使っても展開元がないため、別建てで提供している。
func BatchSupportedPlaceholders() []string {
	return slices.Clone(batchSupportedPlaceholders)
}

// BatchRequiredPlaceholder はすべての batch テンプレートが使うべきプレースホルダを返す。
func BatchRequiredPlaceholder() string { return batchRequiredPlaceholder }

// DefaultTemplates は設定が独自のテンプレートを定義していないときに使う
// 組み込みテンプレートを返す。
func DefaultTemplates() Templates {
	return Templates{
		Design:         defaultDesignTemplate,
		Implementation: defaultImplementationTemplate,
		Batch:          defaultBatchTemplate,
	}
}

// KindFor はタスクに必要なテンプレートを選ぶ。判断材料は実装計画の有無だけ。
// 表示状態や着手可否は進捗を表すものであって、エージェントに投げる問いが
// どれかを決めるものではない。
func KindFor(task domain.Task) Kind {
	if task.HasImplementationPlan {
		return KindImplementation
	}
	return KindDesign
}

// Template は指定した kind に対応する保存済みテンプレートを返す。
func (t Templates) Template(kind Kind) string {
	if kind == KindImplementation {
		return t.Implementation
	}
	return t.Design
}

// Normalize は省略されたテンプレートを組み込みの既定値で埋め、最初に見つかった
// 不正なテンプレートを報告する。prompts がなかった頃の設定ファイルがそのまま
// 読み込めるのはこの処理のおかげ。
func (t Templates) Normalize() (Templates, error) {
	result := t
	defaults := DefaultTemplates()
	if strings.TrimSpace(result.Design) == "" {
		result.Design = defaults.Design
	}
	if strings.TrimSpace(result.Implementation) == "" {
		result.Implementation = defaults.Implementation
	}
	if strings.TrimSpace(result.Batch) == "" {
		result.Batch = defaults.Batch
	}
	if err := validateTemplate(
		"prompts.design", result.Design, supportedPlaceholders, requiredPlaceholder,
	); err != nil {
		return Templates{}, err
	}
	if err := validateTemplate(
		"prompts.implementation", result.Implementation, supportedPlaceholders, requiredPlaceholder,
	); err != nil {
		return Templates{}, err
	}
	if err := validateTemplate(
		"prompts.batch", result.Batch, batchSupportedPlaceholders, batchRequiredPlaceholder,
	); err != nil {
		return Templates{}, err
	}
	return result, nil
}

// Render は 1 つのタスクをプロンプトに展開し、どのテンプレートを使ったかを返す。
// 読み込みから描画までの間に手編集されたファイルが未展開のプレースホルダを
// 出さないよう、保存済みテンプレートをここで再検証する。
func Render(task domain.Task, templates Templates) (Kind, string, error) {
	normalized, err := templates.Normalize()
	if err != nil {
		return "", "", err
	}
	kind := KindFor(task)
	values := map[string]string{
		"task_id":    task.ID,
		"feature_id": task.FeatureID,
		"task_title": task.Title,
		"task_scope": describedScope(task.Scope),
	}
	body := placeholderPattern.ReplaceAllStringFunc(normalized.Template(kind), func(match string) string {
		return values[placeholderName(match)]
	})
	return kind, body, nil
}

// RenderBatch は batch テンプレートを複数タスクに展開する。呼び出し元が並べた順を
// 保ち、タスクごとの指示は含めない。
// docs/design/agent-prompts.md を参照。
func RenderBatch(featureID string, tasks []domain.Task, templates Templates) (string, error) {
	normalized, err := templates.Normalize()
	if err != nil {
		return "", err
	}
	values := map[string]string{
		"feature_id": featureID,
		"task_list":  batchTaskList(tasks),
	}
	return placeholderPattern.ReplaceAllStringFunc(normalized.Batch, func(match string) string {
		return values[placeholderName(match)]
	}), nil
}

// batchTaskList は各タスクを、エージェントが `prx prompt` に渡し返す識別子で列挙する。
// プロンプトを読む人がタスクを見分けられるよう、後ろにタイトルを添える。
func batchTaskList(tasks []domain.Task) string {
	lines := make([]string, 0, len(tasks))
	for _, task := range tasks {
		lines = append(lines, fmt.Sprintf("- %s: %s", task.ID, task.Title))
	}
	return strings.Join(lines, "\n")
}

// unspecifiedScope はスコープなしで作られたタスクの代わりに置く値。テンプレートは
// 「上記のスコープ」を参照させるが、PRX を知らないエージェントは空行と読み込みに
// 失敗した値を区別できない。
const unspecifiedScope = "(not specified)"

func describedScope(scope string) string {
	if strings.TrimSpace(scope) == "" {
		return unspecifiedScope
	}
	return scope
}

// Error は拒否されたテンプレートを報告する。呼び出し元が修正すべきテンプレートを
// 指し示せるよう、フィールド名を保持する。
type Error struct {
	Field   string
	Message string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Message) }

func newError(field, format string, args ...any) *Error {
	return &Error{Field: field, Message: fmt.Sprintf(format, args...)}
}

func validateTemplate(field, value string, supported []string, required string) error {
	if len(value) > MaximumTemplateBytes {
		return newError(field, "template must be at most %d bytes", MaximumTemplateBytes)
	}
	found := make(map[string]struct{})
	for _, match := range placeholderPattern.FindAllString(value, -1) {
		name := placeholderName(match)
		if !slices.Contains(supported, name) {
			return newError(field, "template uses unsupported placeholder %s", match)
		}
		found[name] = struct{}{}
	}
	if _, ok := found[required]; !ok {
		return newError(field, "template must use {{%s}}", required)
	}
	return nil
}

func placeholderName(match string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
}
