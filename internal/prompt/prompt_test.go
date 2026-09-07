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
	// 計画のない完了タスクも依然として設計依頼になる。進捗はプロンプトが
	// 投げる問いの答えにはならない。
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

	// タスクはスコープなしでも作れるが、テンプレートは「上記のスコープ」内に
	// 留まるよう指示する。そこが空行だと、制約がないのではなく値の読み込みに
	// 失敗したと読めてしまう。
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

// 既定テンプレートは多くのインストールがそのまま使うため、各手順が依存する
// コマンドを明示しなければならない。
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

// PRX はローカルのツールなので、リポジトリの読み手には解決できない参照になる。
// 既定テンプレートは、成果物に PRX を持ち込まないようエージェントに指示する。
func TestDefaultTemplatesKeepPRXOutOfTheRepository(t *testing.T) {
	defaults := prompt.DefaultTemplates()
	for name, template := range map[string]string{
		"design":         defaults.Design,
		"implementation": defaults.Implementation,
		"batch":          defaults.Batch,
	} {
		if !strings.Contains(template, "PRX runs on this machine only") ||
			!strings.Contains(template, "may mention PRX, its identifiers, or its commands.") {
			t.Fatalf("default %s template does not forbid mentioning PRX in the repository: %q", name, template)
		}
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

// batch テンプレートはタスク用テンプレートと並べて保存されるため、一度も
// カスタマイズしていないインストールには組み込みの文言が届き続ける必要がある。
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

// batch の語彙はタスクの語彙とは別物。batch は複数タスクを対象にするため、
// タスク用プレースホルダを置いても展開元がない。
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
