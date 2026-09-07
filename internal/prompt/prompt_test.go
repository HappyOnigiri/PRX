package prompt_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

func designTask() domain.Task {
	return domain.Task{
		ID: "T-7", FeatureID: "F-3", Title: "Add the checkout API",
		Scope: "Server only",
	}
}

func TestKindFollowsOnlyTheImplementationPlan(t *testing.T) {
	task := designTask()
	if got := prompt.KindFor(task); got != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", got, prompt.KindDesign)
	}
	// A finished task without a plan is still a design request: progress does not
	// answer the question the prompt asks.
	task.Status = domain.TaskStatusCompleted
	if got := prompt.KindFor(task); got != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", got, prompt.KindDesign)
	}
	task.HasImplementationPlan = true
	if got := prompt.KindFor(task); got != prompt.KindImplementation {
		t.Fatalf("kind=%q, want %q", got, prompt.KindImplementation)
	}
}

func TestRenderExpandsEveryPlaceholderOfTheSelectedTemplate(t *testing.T) {
	templates := prompt.Templates{
		Design:         "design {{task_id}} {{feature_id}} {{task_title}} {{task_scope}}",
		Implementation: "implement {{task_id}}",
	}
	kind, body, err := prompt.Render(designTask(), templates)
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", kind, prompt.KindDesign)
	}
	if want := "design T-7 F-3 Add the checkout API Server only"; body != want {
		t.Fatalf("body=%q, want %q", body, want)
	}

	// A task may be created without a scope, and the templates tell the agent to
	// stay inside "the scope above", so a blank line there would read as a value
	// that failed to load rather than as an absent constraint.
	scopeless := designTask()
	scopeless.Scope = "  "
	_, body, err = prompt.Render(scopeless, templates)
	if err != nil {
		t.Fatal(err)
	}
	if want := "design T-7 F-3 Add the checkout API (not specified)"; body != want {
		t.Fatalf("body=%q, want %q", body, want)
	}

	planned := designTask()
	planned.HasImplementationPlan = true
	kind, body, err = prompt.Render(planned, templates)
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindImplementation || body != "implement T-7" {
		t.Fatalf("kind=%q body=%q", kind, body)
	}
}

func TestRenderFillsAnOmittedTemplateWithItsDefault(t *testing.T) {
	kind, body, err := prompt.Render(designTask(), prompt.Templates{})
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", kind, prompt.KindDesign)
	}
	if !strings.Contains(body, "T-7") || !strings.Contains(body, "F-3") {
		t.Fatalf("default design prompt does not name the task: %q", body)
	}
	if strings.Contains(body, "{{") {
		t.Fatalf("default design prompt kept a placeholder: %q", body)
	}
}

// The default templates are what most installations copy, so they have to name
// the commands their step depends on.
func TestDefaultTemplatesGuideTheAgentThroughPRX(t *testing.T) {
	defaults := prompt.DefaultTemplates()
	for _, command := range []string{
		"prx task update {{task_id}} --status designing",
		"prx task {{task_id}}",
		"prx graph {{feature_id}}",
		"prx plan set {{task_id}}",
	} {
		if !strings.Contains(defaults.Design, command) {
			t.Fatalf("default design template does not mention %q", command)
		}
	}
	for _, command := range []string{"prx plan {{task_id}}", "prx pr attach {{task_id}}", "prx task update {{task_id}}"} {
		if !strings.Contains(defaults.Implementation, command) {
			t.Fatalf("default implementation template does not mention %q", command)
		}
	}
	if strings.Contains(defaults.Design, "prx plan {{task_id}}\n") {
		t.Fatal("the design template reads a plan the task does not have yet")
	}
}

func TestNormalizeRejectsTemplatesTheRendererCouldNotExpand(t *testing.T) {
	for name, test := range map[string]struct {
		templates prompt.Templates
		message   string
	}{
		"design without the task": {
			templates: prompt.Templates{Design: "no target", Implementation: "{{task_id}}"},
			message:   "prompts.design: template must use {{task_id}}",
		},
		"implementation with an unknown placeholder": {
			templates: prompt.Templates{Design: "{{task_id}}", Implementation: "{{task_id}} {{plan_body}}"},
			message:   "prompts.implementation: template uses unsupported placeholder {{plan_body}}",
		},
		"design above the size limit": {
			templates: prompt.Templates{
				Design:         "{{task_id}}" + strings.Repeat("x", prompt.MaximumTemplateBytes),
				Implementation: "{{task_id}}",
			},
			message: "prompts.design: template must be at most 8192 bytes",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := test.templates.Normalize(); err == nil || err.Error() != test.message {
				t.Fatalf("error=%v, want %q", err, test.message)
			}
			if _, _, err := prompt.Render(designTask(), test.templates); err == nil {
				t.Fatal("Render accepted a template Normalize rejects")
			}
		})
	}
}

func TestNormalizeKeepsWhitespaceOtherThanAnEmptyTemplate(t *testing.T) {
	templates := prompt.Templates{Design: "  {{task_id}}\n\n", Implementation: "   "}
	normalized, err := templates.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Design != "  {{task_id}}\n\n" {
		t.Fatalf("design=%q, want the stored text unchanged", normalized.Design)
	}
	if normalized.Implementation != prompt.DefaultTemplates().Implementation {
		t.Fatal("a blank implementation template was not replaced by its default")
	}
	if got := normalized.Template(prompt.KindDesign); got != normalized.Design {
		t.Fatalf("Template(design)=%q", got)
	}
}

func batchTasks() []domain.Task {
	return []domain.Task{
		{ID: "T-7", FeatureID: "F-3", Title: "Add the checkout API"},
		{ID: "T-9", FeatureID: "F-3", Title: "Bill the order"},
	}
}

func TestRenderBatchNamesEveryTaskInTheOrderItWasGiven(t *testing.T) {
	body, err := prompt.RenderBatch("F-3", batchTasks(), prompt.Templates{
		Batch: "batch {{feature_id}}\n{{task_list}}",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "batch F-3\n- T-7: Add the checkout API\n- T-9: Bill the order"
	if body != want {
		t.Fatalf("body=%q, want %q", body, want)
	}
}

// The batch template is stored beside the task templates, so an installation
// that never customized it has to keep receiving the built-in wording.
func TestRenderBatchFillsAnOmittedTemplateWithItsDefault(t *testing.T) {
	body, err := prompt.RenderBatch("F-3", batchTasks(), prompt.Templates{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"F-3", "T-7", "T-9", "SubAgent", "prx prompt TASK_ID"} {
		if !strings.Contains(body, want) {
			t.Fatalf("default batch prompt does not mention %q: %q", want, body)
		}
	}
	if strings.Contains(body, "{{") {
		t.Fatalf("default batch prompt kept a placeholder: %q", body)
	}
}

// The batch vocabulary is not the task vocabulary: a batch covers several tasks,
// so a task placeholder in it would have nothing to expand from.
func TestNormalizeRejectsABatchTemplateOutsideItsOwnVocabulary(t *testing.T) {
	for name, test := range map[string]struct {
		batch   string
		message string
	}{
		"without the task list": {
			batch:   "batch of {{feature_id}}",
			message: "prompts.batch: template must use {{task_list}}",
		},
		"with a task placeholder": {
			batch:   "batch {{task_list}} {{task_title}}",
			message: "prompts.batch: template uses unsupported placeholder {{task_title}}",
		},
	} {
		t.Run(name, func(t *testing.T) {
			templates := prompt.Templates{Batch: test.batch}
			if _, err := templates.Normalize(); err == nil || err.Error() != test.message {
				t.Fatalf("error=%v, want %q", err, test.message)
			}
			if _, err := prompt.RenderBatch("F-3", batchTasks(), templates); err == nil {
				t.Fatal("RenderBatch accepted a template Normalize rejects")
			}
		})
	}
	if slices.Contains(prompt.BatchSupportedPlaceholders(), "task_id") {
		t.Fatal("the batch vocabulary offers a placeholder no batch can expand")
	}
	if prompt.BatchRequiredPlaceholder() != "task_list" {
		t.Fatalf("batch required placeholder=%q", prompt.BatchRequiredPlaceholder())
	}
}
