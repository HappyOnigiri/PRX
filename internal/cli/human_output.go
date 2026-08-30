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
		[]string{"ID", "SLUG", "STATUS", "ARCHIVED", "TASKS", "TITLE"},
		func(table *tabwriter.Writer) {
			for _, feature := range features {
				writeRow(
					table,
					feature.ID,
					feature.Slug,
					feature.Status,
					yesNo(feature.Archived),
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
			{"Slug", feature.Slug},
			{"Title", feature.Title},
			{"Description", displayValue(feature.Description)},
			{"Status", string(feature.Status)},
			{"Archived", yesNo(feature.Archived)},
			{"Tasks", fmt.Sprint(feature.TaskCount)},
			{"Ready", fmt.Sprint(feature.ReadyCount)},
			{"Reviews", fmt.Sprint(feature.ReviewWaitingCount)},
			{"Conflicts", fmt.Sprint(feature.ConflictCount)},
			{"Merged", fmt.Sprint(feature.MergedCount)},
			{"Created", formatTime(feature.CreatedAt)},
			{"Updated", formatTime(feature.UpdatedAt)},
		})
	}
}

func renderTaskList(tasks []domain.Task) humanRenderer {
	return func(out io.Writer) error { return writeTaskTable(out, tasks) }
}

func writeTaskTable(out io.Writer, tasks []domain.Task) error {
	if len(tasks) == 0 {
		_, err := fmt.Fprintln(out, "No tasks found.")
		return err
	}
	return writeTable(
		out,
		[]string{"ID", "STATUS", "READY", "KIND", "ASSIGNEE", "TITLE"},
		func(table *tabwriter.Writer) {
			for _, task := range tasks {
				writeRow(
					table,
					task.ID,
					task.DisplayState,
					yesNo(task.Ready),
					task.Kind,
					displayValue(task.Assignee),
					task.Title,
				)
			}
		},
	)
}

func renderTaskDetail(task domain.Task) humanRenderer {
	return func(out io.Writer) error {
		return writeFields(out, [][2]string{
			{"ID", task.ID},
			{"Feature", task.FeatureID},
			{"Title", task.Title},
			{"Scope", displayValue(task.Scope)},
			{"Kind", string(task.Kind)},
			{"Status", string(task.Status)},
			{"Display state", string(task.DisplayState)},
			{"Ready", yesNo(task.Ready)},
			{"Assignee", displayValue(task.Assignee)},
			{"Implementation plan", yesNo(task.HasImplementationPlan)},
			{"Blocked reason", displayValue(task.BlockedReason)},
			{"Created", formatTime(task.CreatedAt)},
			{"Updated", formatTime(task.UpdatedAt)},
		})
	}
}

func renderNode(value any) humanRenderer {
	switch typed := value.(type) {
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
		[]string{"ID", "FEATURE", "TASK", "KIND", "PLAN", "TITLE", "LOCATOR"},
		func(table *tabwriter.Writer) {
			for _, document := range documents {
				writeRow(
					table,
					document.ID,
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

func renderQueue(name string, tasks []domain.Task) humanRenderer {
	return func(out io.Writer) error {
		if len(tasks) == 0 {
			_, err := fmt.Fprintf(out, "No %s tasks found.\n", name)
			return err
		}
		return writeTaskTable(out, tasks)
	}
}

func renderGraph(feature domain.Feature, tasks []domain.Task, dependencies []domain.Dependency) humanRenderer {
	return func(out io.Writer) error {
		if _, err := fmt.Fprintf(
			out,
			"Feature\n  %s — %s (%s)\n\nTasks\n",
			feature.Slug,
			feature.Title,
			feature.Status,
		); err != nil {
			return err
		}
		if err := writeTaskTable(out, tasks); err != nil {
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
			"%s: %d features, %d tasks, %d dependencies, %d pull requests, %d documents.\n"+
				"Queues: %d ready, %d waiting for review, %d conflicts, %d stale.\n\nFeatures\n",
			prefix,
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
		if err := writeFeatureTable(out, snapshot.Features); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "\nTasks"); err != nil {
			return err
		}
		if err := writeTaskTable(out, snapshot.Tasks); err != nil {
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
			"Config version: %d\nAutomatic sync interval: %d seconds\n\nHosts\n",
			value.Version,
			value.GitHub.AutoSyncIntervalSeconds,
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
