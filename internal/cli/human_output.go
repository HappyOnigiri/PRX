package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func renderMessage(format string, args ...any) humanRenderer {
	return func(out io.Writer) error {
		_, err := fmt.Fprintf(out, format+"\n", args...)
		return err
	}
}

func renderProjectList(projects []domain.Project) humanRenderer {
	return func(out io.Writer) error { return writeProjectTable(out, projects) }
}

func writeProjectTable(out io.Writer, projects []domain.Project) error {
	if len(projects) == 0 {
		_, err := fmt.Fprintln(out, "No projects found.")
		return err
	}
	return writeTable(out, []string{"ID", "ARCHIVED", "TITLE"}, func(table *tabwriter.Writer) {
		for _, project := range projects {
			writeRow(table, project.ID, yesNo(project.Archived), project.Title)
		}
	})
}

// renderProjectFields は project 単体を描画する。`show` と project 詳細は
// 同じ描画部品を使う。
func renderProjectFields(project domain.Project) humanRenderer {
	return func(out io.Writer) error {
		return writeFields(out, [][2]string{
			{"ID", project.ID},
			{"Title", project.Title},
			{"Description", displayValue(project.Description)},
			{"Archived", yesNo(project.Archived)},
			{"Prompt overrides", promptOverrideSummary(project.PromptOverrides)},
			{"Task label overrides", taskLabelOverrideSummary(project.TaskLabelOverrides)},
			{"Created", formatTime(project.CreatedAt)},
			{"Updated", formatTime(project.UpdatedAt)},
		})
	}
}

func renderProjectDetail(
	project domain.Project,
	features []domain.Feature,
	documents []domain.Document,
) humanRenderer {
	return func(out io.Writer) error {
		if err := renderProjectFields(project)(out); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nFeatures"); err != nil {
			return err
		}
		if err := writeFeatureTable(out, features); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nDocuments"); err != nil {
			return err
		}
		return writeDocumentTable(out, documents)
	}
}

func renderFeatureList(features []domain.Feature) humanRenderer {
	return func(out io.Writer) error { return writeFeatureTable(out, features) }
}

func writeFeatureTable(out io.Writer, features []domain.Feature) error {
	if len(features) == 0 {
		_, err := fmt.Fprintln(out, "No features found.")
		return err
	}
	return writeTable(
		out,
		[]string{"ID", "PROJECT", "STATUS", "ARCHIVED", "READ-ONLY", "TASKS", "TITLE"},
		func(table *tabwriter.Writer) {
			for _, feature := range features {
				writeRow(
					table,
					feature.ID,
					feature.ProjectID,
					feature.DisplayStatus,
					yesNo(feature.Archived),
					yesNo(feature.ReadOnly),
					feature.TaskCount,
					feature.Title,
				)
			}
		},
	)
}

func renderFeatureDetail(feature domain.Feature) humanRenderer {
	return func(out io.Writer) error {
		return writeFields(out, [][2]string{
			{"ID", feature.ID},
			{"Project", feature.ProjectID},
			{"Title", feature.Title},
			{"Description", displayValue(feature.Description)},
			{"Prompt overrides", promptOverrideSummary(feature.PromptOverrides)},
			{"Task label overrides", taskLabelOverrideSummary(feature.TaskLabelOverrides)},
			{"Status", string(feature.Status)},
			{"Display status", string(feature.DisplayStatus)},
			{"Archived", yesNo(feature.Archived)},
			{"Read-only", yesNo(feature.ReadOnly)},
			{"Tasks", fmt.Sprint(feature.TaskCount)},
			{"Ready", fmt.Sprint(feature.ReadyCount)},
			{"Reviews", fmt.Sprint(feature.ReviewWaitingCount)},
			{"Conflicts", fmt.Sprint(feature.ConflictCount)},
			{"Merged", fmt.Sprint(feature.MergedCount)},
			{"Finished", fmt.Sprint(feature.FinishedCount)},
			{"Created", formatTime(feature.CreatedAt)},
			{"Updated", formatTime(feature.UpdatedAt)},
		})
	}
}

func promptOverrideSummary(overrides domain.PromptTemplateOverrides) string {
	values := make([]string, 0, 3)
	if overrides.Design != "" {
		values = append(values, "design")
	}
	if overrides.Implementation != "" {
		values = append(values, "implementation")
	}
	if overrides.Batch != "" {
		values = append(values, "batch")
	}
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func taskLabelOverrideSummary(overrides domain.TaskLabelOverrides) string {
	if len(overrides) == 0 {
		return "none"
	}
	keys := domain.SortedTaskLabelKeys(overrides)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		value := overrides[key]
		parts := make([]string, 0, 2)
		if value.Text != "" {
			parts = append(parts, "text")
		}
		if value.Color != "" {
			parts = append(parts, "color")
		}
		values = append(values, fmt.Sprintf("%s(%s)", key, strings.Join(parts, ",")))
	}
	return strings.Join(values, ", ")
}

func renderTaskListWithAppearances(
	tasks []domain.Task,
	appearances map[string]domain.TaskLabelAppearances,
) humanRenderer {
	return func(out io.Writer) error { return writeTaskTableWithAppearances(out, tasks, appearances) }
}

func writeTaskTableWithAppearances(
	out io.Writer,
	tasks []domain.Task,
	appearances map[string]domain.TaskLabelAppearances,
) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(out, "No tasks found.")
		return err
	}
	return writeTable(
		out,
		[]string{"ID", "STATUS", "READY", "ASSIGNEE", "TITLE"},
		func(table *tabwriter.Writer) {
			for _, task := range tasks {
				writeRow(
					table,
					task.ID,
					taskDisplayLabel(task, appearances),
					yesNo(task.Ready),
					displayValue(task.Assignee),
					task.Title,
				)
			}
		},
	)
}

func renderTaskDetail(task domain.Task) humanRenderer {
	return renderTaskDetailWithAppearances(task, nil)
}

func renderTaskDetailWithAppearances(
	task domain.Task,
	appearances map[string]domain.TaskLabelAppearances,
) humanRenderer {
	return func(out io.Writer) error {
		return writeFields(out, [][2]string{
			{"ID", task.ID},
			{"Feature", task.FeatureID},
			{"Title", task.Title},
			{"Scope", displayValue(task.Scope)},
			{"Status", string(task.Status)},
			{"Display state", taskDisplayLabel(task, appearances)},
			{"Ready", yesNo(task.Ready)},
			{"Assignee", displayValue(task.Assignee)},
			{"Implementation plan", yesNo(task.HasImplementationPlan)},
			{"Blocks", displayValue(blockLabelsTextWithAppearances(task, appearances))},
			{"Blocked reason", displayValue(task.BlockedReason)},
			{"Created", formatTime(task.CreatedAt)},
			{"Updated", formatTime(task.UpdatedAt)},
		})
	}
}

func taskDisplayLabel(task domain.Task, appearances map[string]domain.TaskLabelAppearances) string {
	if values := appearances[task.FeatureID]; values != nil {
		if value, ok := values[domain.TaskLabelKeyForDisplayState(task.DisplayState)]; ok && value.Text != "" {
			return value.Text
		}
	}
	return string(task.DisplayState)
}

func blockLabelsTextWithAppearances(task domain.Task, appearances map[string]domain.TaskLabelAppearances) string {
	values := make([]string, 0, len(task.BlockLabels))
	for _, label := range task.BlockLabels {
		text := string(label)
		if valuesForFeature := appearances[task.FeatureID]; valuesForFeature != nil {
			if value, ok := valuesForFeature[domain.TaskLabelKeyForBlockLabel(label)]; ok && value.Text != "" {
				text = value.Text
			}
		}
		values = append(values, text)
	}
	return strings.Join(values, ", ")
}

func renderNode(value any) humanRenderer {
	switch typed := value.(type) {
	case domain.Project:
		return renderProjectFields(typed)
	case domain.Feature:
		return renderFeatureDetail(typed)
	case domain.Task:
		return renderTaskDetail(typed)
	default:
		return renderMessage("Node: %v", value)
	}
}

func renderDependencyList(dependencies []domain.Dependency) humanRenderer {
	return func(out io.Writer) error { return writeDependencyTable(out, dependencies) }
}

func writeDependencyTable(out io.Writer, dependencies []domain.Dependency) error {
	if len(dependencies) == 0 {
		_, err := fmt.Fprintln(out, "No dependencies found.")
		return err
	}
	return writeTable(out, []string{"BLOCKER", "BLOCKED"}, func(table *tabwriter.Writer) {
		for _, dependency := range dependencies {
			writeRow(table, dependency.BlockerTaskID, dependency.BlockedTaskID)
		}
	})
}

func renderPullRequestList(pullRequests []domain.PullRequest) humanRenderer {
	return func(out io.Writer) error { return writePullRequestTable(out, pullRequests) }
}

func writePullRequestTable(out io.Writer, pullRequests []domain.PullRequest) error {
	if len(pullRequests) == 0 {
		_, err := fmt.Fprintln(out, "No pull requests attached.")
		return err
	}
	return writeTable(
		out,
		[]string{"TASK", "HOST", "REPOSITORY", "NUMBER", "STATE", "STALE", "URL"},
		func(table *tabwriter.Writer) {
			for _, pullRequest := range pullRequests {
				writeRow(table, pullRequest.TaskID, pullRequest.Host, pullRequest.Owner+"/"+pullRequest.Repository,
					pullRequest.Number, pullRequest.DisplayState, yesNo(pullRequest.Stale), pullRequest.URL)
			}
		},
	)
}

func renderDocumentList(documents []domain.Document) humanRenderer {
	return func(out io.Writer) error { return writeDocumentTable(out, documents) }
}

func writeDocumentTable(out io.Writer, documents []domain.Document) error {
	if len(documents) == 0 {
		_, err := fmt.Fprintln(out, "No documents found.")
		return err
	}
	return writeTable(
		out,
		[]string{"ID", "PROJECT", "FEATURE", "TASK", "KIND", "PLAN", "TITLE", "LOCATOR"},
		func(table *tabwriter.Writer) {
			for _, document := range documents {
				writeRow(
					table,
					document.ID,
					displayValue(document.ProjectID),
					displayValue(document.FeatureID),
					displayValue(document.TaskID),
					document.Kind,
					yesNo(document.IsImplementationPlan),
					displayValue(document.Title),
					displayValue(document.Locator),
				)
			}
		},
	)
}

func renderDocumentDetail(document domain.Document) humanRenderer {
	return func(out io.Writer) error {
		if err := writeFields(out, [][2]string{
			{"ID", document.ID},
			{"Project", displayValue(document.ProjectID)},
			{"Feature", displayValue(document.FeatureID)},
			{"Task", displayValue(document.TaskID)},
			{"Kind", string(document.Kind)},
			{"Title", displayValue(document.Title)},
			{"Locator", displayValue(document.Locator)},
			{"Implementation plan", yesNo(document.IsImplementationPlan)},
			{"Created", formatTime(document.CreatedAt)},
			{"Updated", formatTime(document.UpdatedAt)},
		}); err != nil {
			return err
		}
		if document.Content != "" {
			_, err := fmt.Fprintf(out, "\n%s", document.Content)
			if err != nil {
				return err
			}
			if !strings.HasSuffix(document.Content, "\n") {
				_, err = io.WriteString(out, "\n")
			}
			return err
		}
		return nil
	}
}

func renderImplementationPlan(plan domain.Document) humanRenderer {
	return func(out io.Writer) error {
		if _, err := fmt.Fprintf(
			out,
			"Task: %s\nKind: %s\nUpdated: %s\n\n",
			plan.TaskID,
			plan.Kind,
			formatTime(plan.UpdatedAt),
		); err != nil {
			return err
		}
		if plan.Kind != domain.DocumentKindMarkdown {
			_, err := fmt.Fprintln(out, plan.Locator)
			return err
		}
		if _, err := io.WriteString(out, plan.Content); err != nil {
			return err
		}
		if !strings.HasSuffix(plan.Content, "\n") {
			_, err := io.WriteString(out, "\n")
			return err
		}
		return nil
	}
}

func renderGraphWithAppearances(
	feature domain.Feature,
	tasks []domain.Task,
	dependencies []domain.Dependency,
	appearances map[string]domain.TaskLabelAppearances,
) humanRenderer {
	return func(out io.Writer) error {
		if _, err := fmt.Fprintf(
			out,
			"Feature\n  %s — %s (%s)\n\nTasks\n",
			feature.ID,
			feature.Title,
			feature.Status,
		); err != nil {
			return err
		}
		if err := writeTaskTableWithAppearances(out, tasks, appearances); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nDependencies"); err != nil {
			return err
		}
		return writeDependencyTable(out, dependencies)
	}
}

func renderSnapshot(prefix string, snapshot domain.Snapshot) humanRenderer {
	return func(out io.Writer) error {
		if _, err := fmt.Fprintf(
			out,
			"%s: %d projects, %d features, %d tasks, %d dependencies, %d pull requests, %d documents.\n"+
				"Queues: %d ready, %d waiting for review, %d conflicts, %d stale.\n\nProjects\n",
			prefix,
			len(snapshot.Projects),
			len(snapshot.Features),
			len(snapshot.Tasks),
			len(snapshot.Dependencies),
			len(snapshot.PullRequests),
			len(
				snapshot.Documents,
			),
			len(snapshot.ReadyTasks),
			len(snapshot.ReviewWaitingTasks),
			len(snapshot.ConflictTasks),
			len(snapshot.StaleTasks),
		); err != nil {
			return err
		}
		if err := writeProjectTable(out, snapshot.Projects); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nFeatures"); err != nil {
			return err
		}
		if err := writeFeatureTable(out, snapshot.Features); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nTasks"); err != nil {
			return err
		}
		if err := writeTaskTableWithAppearances(out, snapshot.Tasks, featureAppearances(snapshot)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nDependencies"); err != nil {
			return err
		}
		if err := writeDependencyTable(out, snapshot.Dependencies); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nPull requests"); err != nil {
			return err
		}
		if err := writePullRequestTable(out, snapshot.PullRequests); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nDocuments"); err != nil {
			return err
		}
		return writeDocumentTable(out, snapshot.Documents)
	}
}

func renderHostList(hosts []config.Host) humanRenderer {
	return func(out io.Writer) error { return writeHostTable(out, hosts) }
}

func writeHostTable(out io.Writer, hosts []config.Host) error {
	if len(hosts) == 0 {
		_, err := fmt.Fprintln(out, "No GitHub hosts configured.")
		return err
	}
	return writeTable(
		out,
		[]string{"HOST", "WEB URL", "API URL", "UPLOAD URL", "GRAPHQL URL"},
		func(table *tabwriter.Writer) {
			for _, host := range hosts {
				writeRow(table, host.Host, host.WebURL, host.APIURL, host.UploadURL, host.GraphQLURL)
			}
		},
	)
}

func renderAuthList(methods []config.PublicAuthMethod) humanRenderer {
	return func(out io.Writer) error { return writeAuthTable(out, methods) }
}

func writeAuthTable(out io.Writer, methods []config.PublicAuthMethod) error {
	if len(methods) == 0 {
		_, err := fmt.Fprintln(out, "No authentication methods configured.")
		return err
	}
	return writeTable(
		out,
		[]string{"ID", "HOST", "TYPE", "ACCOUNT", "SERVICE", "VARIABLE", "USER", "SECRET"},
		func(table *tabwriter.Writer) {
			for _, method := range methods {
				writeRow(
					table,
					method.ID,
					method.Host,
					method.Type,
					displayValue(method.Account),
					displayValue(method.Service),
					displayValue(method.Variable),
					displayValue(method.User),
					yesNo(method.SecretConfigured),
				)
			}
		},
	)
}

func renderConfig(value config.PublicConfig) humanRenderer {
	return func(out io.Writer) error {
		if _, err := fmt.Fprintf(
			out,
			"Config version: %d\nAutomatic sync interval: %d seconds\nServer port: %s\n"+
				"Language: %s (effective: %s)\n\nHosts\n",
			value.Version,
			value.GitHub.AutoSyncIntervalSeconds,
			value.Server.Port,
			value.Language,
			value.EffectiveLanguage,
		); err != nil {
			return err
		}
		if err := writeHostTable(out, value.GitHub.Hosts); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nAuthentication methods"); err != nil {
			return err
		}
		return writeAuthTable(out, value.GitHub.AuthMethods)
	}
}

// renderConfigValidation は、このビルドが知らないフィールドがファイルにあっても
// 妥当な設定を成功扱いのままにし、次の書き込みで失われるフィールドが分かるよう
// それらを列挙する。
func renderConfigValidation(warnings []string) humanRenderer {
	return func(out io.Writer) error {
		if _, err := fmt.Fprintln(out, "Configuration is valid."); err != nil {
			return err
		}
		for _, warning := range warnings {
			if _, err := fmt.Fprintf(out, "Warning: %s\n", warning); err != nil {
				return err
			}
		}
		return nil
	}
}

func renderSyncStatus(status domain.GitHubSyncStatus) humanRenderer {
	return func(out io.Writer) error {
		lastAttempt := "never"
		if status.LastAttemptAt != nil {
			lastAttempt = formatTime(*status.LastAttemptAt)
		}
		lastUpdated := "never"
		if status.LastUpdatedAt != nil {
			lastUpdated = formatTime(*status.LastUpdatedAt)
		}
		return writeFields(out, [][2]string{
			{"Interval", fmt.Sprintf("%d seconds", status.IntervalSeconds)},
			{"Last attempt", lastAttempt},
			{"Last updated", lastUpdated},
			{"Succeeded", fmt.Sprint(status.Succeeded)},
			{"Failed", fmt.Sprint(status.Failed)},
			{"Error", displayValue(status.Error)},
		})
	}
}

func writeFields(out io.Writer, fields [][2]string) error {
	return writeTable(out, nil, func(table *tabwriter.Writer) {
		for _, field := range fields {
			_, _ = fmt.Fprintf(table, "%s:\t%s\n", field[0], field[1])
		}
	})
}

func writeTable(out io.Writer, headers []string, rows func(*tabwriter.Writer)) error {
	table := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	if len(headers) > 0 {
		writeRow(table, stringValues(headers)...)
	}
	rows(table)
	return table.Flush()
}

func writeRow(out io.Writer, values ...any) {
	for index, value := range values {
		if index > 0 {
			_, _ = io.WriteString(out, "\t")
		}
		_, _ = fmt.Fprint(out, value)
	}
	_, _ = io.WriteString(out, "\n")
}

func stringValues(values []string) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

func displayValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format(time.RFC3339)
}
