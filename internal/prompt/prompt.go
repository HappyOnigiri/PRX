// Package prompt owns the agent prompt templates shared by the CLI, the RPC
// server, and the WebUI: the built-in templates, their validation, the
// placeholder substitution, and the rule that selects a template for one task.
package prompt

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// MaximumTemplateBytes bounds one stored template. Templates travel through
// YAML, JSON, and Protocol Buffers, and they exist to be pasted into another
// agent's input, so an unbounded body helps nobody.
const MaximumTemplateBytes = 8192

// Kind identifies which template a task needs. It is derived from the task and
// is never stored.
type Kind string

const (
	// KindDesign asks an agent to produce an implementation plan.
	KindDesign Kind = "design"
	// KindImplementation asks an agent to carry out a registered plan.
	KindImplementation Kind = "implementation"
)

// Templates holds one template per Kind, plus the batch template, which is not
// a Kind because it is chosen by the caller asking for several tasks at once
// rather than derived from any single task.
type Templates struct {
	Design         string `yaml:"design"         json:"design"`
	Implementation string `yaml:"implementation" json:"implementation"`
	Batch          string `yaml:"batch"          json:"batch"`
}

// placeholderPattern matches every substitution token, including unsupported
// ones, so validation can reject a name the renderer would silently keep.
var placeholderPattern = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// requiredPlaceholder keeps a template pointed at one task. Without it the
// rendered prompt names no target and the receiving agent cannot start.
const requiredPlaceholder = "task_id"

// supportedPlaceholders is the complete substitution vocabulary. Plan bodies are
// deliberately absent: a plan may be up to 1 MiB or may live behind a locator,
// so the prompt tells the agent to read it with `prx plan TASK_ID` instead.
var supportedPlaceholders = []string{
	requiredPlaceholder,
	"feature_id",
	"task_title",
	"task_scope",
}

// batchRequiredPlaceholder keeps a batch template pointed at the tasks it was
// asked for. Without it the rendered prompt names no target at all, which is
// worse than a task prompt missing its own identifier.
const batchRequiredPlaceholder = "task_list"

// batchSupportedPlaceholders is the batch vocabulary, which carries no
// single task's title or scope.
// See docs/design/agent-prompts.md.
var batchSupportedPlaceholders = []string{
	batchRequiredPlaceholder,
	"feature_id",
}

const defaultDesignTemplate = `Design PRX task {{task_id}} of feature {{feature_id}}.

Title: {{task_title}}
Scope: {{task_scope}}

PRX is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`prx --help`" + ` and ` + "`prx <command> --help`" + ` for its exact surface.

1. Mark the task as being designed before anything else.
   - ` + "`prx task update {{task_id}} --status designing`" + `
2. Read the task and the work it depends on.
   - ` + "`prx task {{task_id}}`" + `
   - ` + "`prx graph {{feature_id}}`" + `
3. Investigate the repository and decide how the scope above should be built.
4. Register the resulting plan on the task.
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

1. Read the task, the work it depends on, and its registered plan.
   - ` + "`prx task {{task_id}}`" + `
   - ` + "`prx graph {{feature_id}}`" + `
   - ` + "`prx plan {{task_id}}`" + `
2. Mark the task as being worked on before you change anything.
   - ` + "`prx task update {{task_id}} --status in_progress`" + `
3. Implement the plan, staying inside the scope above.
   Branch from the base the work actually belongs on rather than from main or master by default.
   A task whose blocker is still open belongs on that blocker's branch, so the two pull requests stack.
4. Record the result in PRX.
   - Work that lands as a pull request: open it against the base you branched from,
     then run ` + "`prx pr attach {{task_id}} PULL_REQUEST_URL`" + `.
     Its state then follows the pull request, so do not set the status by hand.
   - Work without a pull request: run ` + "`prx task update {{task_id}} --status completed`" + ` once it is done.

Report what you changed and anything the plan did not cover.
`

const defaultBatchTemplate = `Implement the PRX tasks of feature {{feature_id}} listed below.

PRX is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`prx --help`" + ` and ` + "`prx <command> --help`" + ` for its exact surface.

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

// SupportedPlaceholders returns the substitution vocabulary, without the
// surrounding braces. It exists so a client can present what the server accepts
// instead of maintaining its own list, which would drift silently.
func SupportedPlaceholders() []string {
	return slices.Clone(supportedPlaceholders)
}

// RequiredPlaceholder returns the placeholder every task template must use.
func RequiredPlaceholder() string { return requiredPlaceholder }

// BatchSupportedPlaceholders returns the batch substitution vocabulary. It is
// served separately because a batch template that used a task placeholder would
// have nothing to expand it from.
func BatchSupportedPlaceholders() []string {
	return slices.Clone(batchSupportedPlaceholders)
}

// BatchRequiredPlaceholder returns the placeholder every batch template must use.
func BatchRequiredPlaceholder() string { return batchRequiredPlaceholder }

// DefaultTemplates returns the built-in templates used when the configuration
// does not define its own.
func DefaultTemplates() Templates {
	return Templates{
		Design:         defaultDesignTemplate,
		Implementation: defaultImplementationTemplate,
		Batch:          defaultBatchTemplate,
	}
}

// KindFor selects the template a task needs. Only the presence of an
// implementation plan decides it: display state and readiness describe progress
// rather than which question the agent is being asked.
func KindFor(task domain.Task) Kind {
	if task.HasImplementationPlan {
		return KindImplementation
	}
	return KindDesign
}

// Template returns the stored template for one kind.
func (t Templates) Template(kind Kind) string {
	if kind == KindImplementation {
		return t.Implementation
	}
	return t.Design
}

// Normalize fills in an omitted template with its built-in default and reports
// the first invalid template. It is what keeps a configuration file written
// before prompts existed loading unchanged.
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

// Render expands one task into its prompt and reports which template produced
// it. The stored templates are validated again here so a file edited by hand
// between a load and a render cannot emit an unexpanded placeholder.
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

// RenderBatch expands the batch template over several tasks, keeping the order
// the caller listed them in and leaving the per-task instructions out.
// See docs/design/agent-prompts.md.
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

// batchTaskList names every task by the identifier the agent passes back to
// `prx prompt`, with the title behind it so a person reading the prompt can tell
// the tasks apart.
func batchTaskList(tasks []domain.Task) string {
	lines := make([]string, 0, len(tasks))
	for _, task := range tasks {
		lines = append(lines, fmt.Sprintf("- %s: %s", task.ID, task.Title))
	}
	return strings.Join(lines, "\n")
}

// unspecifiedScope stands in for a task created without a scope. The templates
// point the agent at "the scope above", and a receiving agent that knows nothing
// about PRX cannot tell a blank line apart from a value that failed to load.
const unspecifiedScope = "(not specified)"

func describedScope(scope string) string {
	if strings.TrimSpace(scope) == "" {
		return unspecifiedScope
	}
	return scope
}

// Error reports a rejected template. The field name is carried so a caller can
// point at the template the reader has to fix.
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
