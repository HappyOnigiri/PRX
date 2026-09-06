package prompt_test

import (
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

func designTask() domain.Task {
	return domain.Task{
		ID: "T-7", FeatureID: "F-3", Title: "Add the checkout API",
		Scope: "Server only", Kind: domain.TaskKindPullRequest,
	}
}

func TestKindFollowsOnlyTheImplementationPlan(t *testing.T) {
	task := designTask()
	if got := prompt.KindFor(task); got != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", got, prompt.KindDesign)
	}
	// A finished manual task without a plan is still a design request: progress
	// does not answer the question the prompt asks.
	task.Kind = domain.TaskKindManual
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
		Design:         "design {{task_id}} {{feature_id}} {{task_title}} {{task_scope}} {{task_kind}}",
		Implementation: "implement {{task_id}}",
	}
	kind, body, err := prompt.Render(designTask(), templates)
	if err != nil {
		t.Fatal(err)
	}
	if kind != prompt.KindDesign {
		t.Fatalf("kind=%q, want %q", kind, prompt.KindDesign)
	}
	if want := "design T-7 F-3 Add the checkout API Server only pr"; body != want {
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
	if want := "design T-7 F-3 Add the checkout API (not specified) pr"; body != want {
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
	for _, command := range []string{"prx task {{task_id}}", "prx graph {{feature_id}}", "prx plan set {{task_id}}"} {
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
