package cli

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

// promptResponse is the JSON form of `prx prompt`. The kind is carried so a
// caller can tell a design request from an implementation request without
// re-deriving it from the task.
type promptResponse struct {
	TaskID string `json:"task_id"`
	Kind   string `json:"kind"`
	Prompt string `json:"prompt"`
}

func (s *state) promptCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prompt TASK_ID",
		Short: "Print the agent prompt for a task",
		Long: "Print the agent prompt for a task.\n\n" +
			"A task without an implementation plan gets the design prompt, and a task with one gets " +
			"the implementation prompt.\n" +
			"Both templates come from the shared configuration, so the WebUI copies the same text.",
		Example: "prx prompt T-1\nprx prompt T-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			task, ok := findSnapshotTask(snapshot, args[0])
			if !ok {
				return domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", args[0])
			}
			store, err := s.configStore()
			if err != nil {
				return configCommandError(err)
			}
			settings, err := store.Load()
			if err != nil {
				return configCommandError(err)
			}
			kind, body, err := prompt.Render(task, settings.Prompts)
			if err != nil {
				return configCommandError(err)
			}
			value := promptResponse{TaskID: task.ID, Kind: string(kind), Prompt: body}
			return s.write(value, renderPrompt(body))
		},
	}
}

func findSnapshotTask(snapshot domain.Snapshot, id string) (domain.Task, bool) {
	for _, task := range snapshot.Tasks {
		if task.ID == id {
			return task, true
		}
	}
	return domain.Task{}, false
}

// renderPrompt prints the prompt body alone. Anything else would have to be
// deleted by hand before the text could be handed to another agent.
func renderPrompt(body string) humanRenderer {
	return func(out io.Writer) error {
		if _, err := io.WriteString(out, body); err != nil {
			return err
		}
		if strings.HasSuffix(body, "\n") {
			return nil
		}
		_, err := io.WriteString(out, "\n")
		return err
	}
}
