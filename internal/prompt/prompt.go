// Package prompt owns the agent prompt templates shared by the CLI, the RPC
// server, and the WebUI. It holds the built-in templates, their validation, the
// placeholder substitution, and the rule that selects a template for one task,
// so no caller can drift from another by re-implementing any of them.
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

// Templates holds one template per Kind.
type Templates struct {
	Design         string `yaml:"design"         json:"design"`
	Implementation string `yaml:"implementation" json:"implementation"`
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

const defaultDesignTemplate = `Design PRX task {{task_id}} of feature {{feature_id}}.

Title: {{task_title}}
Scope: {{task_scope}}

PRX is a local CLI that tracks tasks and the dependencies between them.
Run ` + "`prx --help`" + ` and ` + "`prx <command> --help`" + ` for its exact surface.

1. Read the task and the work it depends on.
   - ` + "`prx task {{task_id}}`" + `
   - ` + "`prx graph {{feature_id}}`" + `
2. Investigate the repository and decide how the scope above should be built.
3. Register the resulting plan on the task.
   - ` + "`prx plan set {{task_id}} --file PLAN.md`" + `
   - ` + "`prx plan set {{task_id}} --stdin`" + `

Design only: leave the implementation, the task status, and the pull request to the next step.
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
2. Implement the plan, staying inside the scope above.
3. Record the result in PRX.
   - Work that lands as a pull request: open it, then run ` + "`prx pr attach {{task_id}} PULL_REQUEST_URL`" + `.
     Its state then follows the pull request, so do not set the status by hand.
   - Work without a pull request: run ` + "`prx task update {{task_id}} --status completed`" + ` once it is done.

Report what you changed and anything the plan did not cover.
`

// SupportedPlaceholders returns the substitution vocabulary, without the
// surrounding braces. It exists so a client can present what the server accepts
// instead of maintaining its own list, which would drift from this one without
// anything failing.
func SupportedPlaceholders() []string {
	return slices.Clone(supportedPlaceholders)
}

// RequiredPlaceholder returns the placeholder every template must use.
func RequiredPlaceholder() string { return requiredPlaceholder }

// DefaultTemplates returns the built-in templates used when the configuration
// does not define its own.
func DefaultTemplates() Templates {
	return Templates{Design: defaultDesignTemplate, Implementation: defaultImplementationTemplate}
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
	if err := validateTemplate("prompts.design", result.Design); err != nil {
		return Templates{}, err
	}
	if err := validateTemplate("prompts.implementation", result.Implementation); err != nil {
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

func validateTemplate(field, value string) error {
	if len(value) > MaximumTemplateBytes {
		return newError(field, "template must be at most %d bytes", MaximumTemplateBytes)
	}
	found := make(map[string]struct{})
	for _, match := range placeholderPattern.FindAllString(value, -1) {
		name := placeholderName(match)
		if !isSupportedPlaceholder(name) {
			return newError(field, "template uses unsupported placeholder %s", match)
		}
		found[name] = struct{}{}
	}
	if _, ok := found[requiredPlaceholder]; !ok {
		return newError(field, "template must use {{%s}}", requiredPlaceholder)
	}
	return nil
}

func placeholderName(match string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
}

func isSupportedPlaceholder(name string) bool {
	return slices.Contains(supportedPlaceholders, name)
}
