package cli

import "github.com/spf13/cobra"

func (s *state) pullRequestCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "pr",
		Short:   "List or attach GitHub pull requests",
		Example: "prx pr",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			pullRequests := nonNilSlice(snapshot.PullRequests)
			return s.write(map[string]any{"pull_requests": pullRequests}, renderPullRequestList(pullRequests))
		},
	}
	var task, url string
	attach := &cobra.Command{
		Use:     "attach",
		Short:   "Attach a GitHub pull request to a task",
		Example: "prx pr attach --task TASK_ID --url https://github.com/acme/payments/pull/42",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			value, err := s.service.AttachPullRequest(cmd.Context(), task, url)
			if err != nil {
				return err
			}
			return s.write(value, renderMessage("Attached pull request #%d to task %s.", value.Number, value.TaskID))
		},
	}
	attach.Flags().StringVar(&task, "task", "", "task ID")
	attach.Flags().StringVar(&url, "url", "", "GitHub pull request URL")
	_ = attach.MarkFlagRequired("task")
	_ = attach.MarkFlagRequired("url")
	detach := &cobra.Command{
		Use:     "detach TASK_ID",
		Short:   "Detach a pull request; missing tasks return not_found",
		Example: "prx pr detach TASK_ID",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DetachPullRequest(cmd.Context(), args[0]); err != nil {
				return err
			}
			return s.write(
				map[string]string{"detached": args[0]},
				renderMessage("Detached pull request from task %s.", args[0]),
			)
		},
	}
	command.AddCommand(attach, detach)
	return command
}
