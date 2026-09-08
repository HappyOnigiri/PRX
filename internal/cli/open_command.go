package cli

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/PRX/internal/browser"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/runstate"
)

const (
	// openBindPollInterval は bind 前のサーバーがアドレスを書くのを待つ間隔。
	openBindPollInterval = 200 * time.Millisecond
	// openBindTimeout は bind を待つ上限。これを超えるならサーバーは起動に失敗している。
	openBindTimeout = 3 * time.Second
)

func (s *state) openCommand() *cobra.Command {
	var printOnly bool
	command := &cobra.Command{
		Use:   "open",
		Short: "Open the running PRX WebUI in a browser",
		Long: "Open the running PRX WebUI in a browser.\n\n" +
			"The URL comes from the running server itself, so it is correct even when the port fell back " +
			"to an ephemeral one. This never starts a server; use prx daemon start for that.",
		Example: "prx open --print",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opener := browser.New()
			if !opener.Supported() {
				return domain.NewError(
					domain.DomainErrorCodeDaemonUnsupported,
					"prx open requires macOS; read the address from prx daemon --json instead",
				)
			}
			state, err := waitForServeAddress(cmd.Context())
			if err != nil {
				return err
			}
			if printOnly {
				return s.write(
					openResponse(state, false),
					renderMessage("%s", state.URL),
				)
			}
			if err := opener.Open(cmd.Context(), state.URL); err != nil {
				return domain.NewError(domain.DomainErrorCodeDaemonFailed, "%s", err)
			}
			return s.write(openResponse(state, true), renderMessage("Opened %s.", state.URL))
		},
	}
	command.Flags().BoolVar(&printOnly, "print", false, "print the URL instead of opening a browser")
	return command
}

func openResponse(state runstate.State, opened bool) map[string]any {
	return map[string]any{"url": state.URL, "address": state.Address, "opened": opened}
}

// waitForServeAddress は稼働記録からアドレスを読む。ロックはあるが内容が空という状態は
// bind の直前なので短く待つ。ロックがなければ、残っている内容は信じない。異常終了で
// 残った番号を開くと、たまたまそれを掴んだ別プロセスに繋がる。
func waitForServeAddress(ctx context.Context) (runstate.State, error) {
	deadline := time.Now().Add(openBindTimeout)
	for {
		status, state, err := runstate.Read()
		if err != nil {
			return runstate.State{}, domain.NewError(domain.DomainErrorCodeInternal, "%s", err)
		}
		if status == runstate.StatusRunning {
			return state, nil
		}
		if status == runstate.StatusNotRunning || !time.Now().Before(deadline) {
			return runstate.State{}, domain.NewError(
				domain.DomainErrorCodeDaemonNotRunning,
				"No PRX server is running. Start it with prx daemon start.",
			)
		}
		if err := sleepUntil(ctx, openBindPollInterval); err != nil {
			return runstate.State{}, err
		}
	}
}
