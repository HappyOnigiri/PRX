package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/daemon"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/runstate"
)

const (
	// daemonPollInterval は起動と停止を待つ間に稼働記録を読み直す間隔。
	daemonPollInterval = 50 * time.Millisecond
	// daemonWaitTimeout は launchd に依頼してからサーバーが応答するまで待つ上限。
	daemonWaitTimeout = 30 * time.Second
)

// daemonStatusResponse は `prx daemon` の応答。稼働中でない項目はゼロ値になる。
type daemonStatusResponse struct {
	Supported     bool   `json:"supported"`
	Installed     bool   `json:"installed"`
	PlistPath     string `json:"plist_path"`
	PlistStatus   string `json:"plist_status"`
	Running       bool   `json:"running"`
	PID           int    `json:"pid"`
	Address       string `json:"address"`
	URL           string `json:"url"`
	StartedAt     string `json:"started_at"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Version       string `json:"version"`
	BinaryMatches bool   `json:"binary_matches"`
	LogPath       string `json:"log_path"`
}

func (s *state) daemonCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "daemon",
		Short: "Show or manage the background PRX server",
		Long: "Show or manage the background PRX server.\n\n" +
			"On macOS a LaunchAgent starts prx serve at login. The LaunchAgent never passes --addr or " +
			"--demo, so the background server always listens on loopback with real data.\n" +
			"Other operating systems report daemon_unsupported; run prx serve directly there.",
		Example: "prx daemon",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status := s.inspectDaemon()
			return s.write(daemonStatus(status), renderDaemonStatus(status))
		},
	}
	command.AddCommand(
		s.daemonInstallCommand(),
		s.daemonUninstallCommand(),
		s.daemonStartCommand(),
		s.daemonStopCommand(),
		s.daemonRestartCommand(),
	)
	return command
}

func (s *state) inspectDaemon() daemon.Status {
	return daemon.Inspect(launchd.New(), prx.Version())
}

func daemonStatus(status daemon.Status) daemonStatusResponse {
	// plist_status は current・stale・unknown の 3 値である。plist を持たない OS でも空を
	// 返さず unknown に寄せて、`prx debug` の daemon セクションと語彙を揃える。
	plistStatus := status.PlistStatus
	if plistStatus == "" {
		plistStatus = launchd.PlistUnknown
	}
	result := daemonStatusResponse{
		Supported:     status.Supported,
		Installed:     status.Installed,
		PlistPath:     status.PlistPath,
		PlistStatus:   string(plistStatus),
		Running:       status.Running,
		BinaryMatches: status.BinaryMatches,
		LogPath:       status.LogPath,
	}
	if !status.Running || status.AddressUnknown {
		return result
	}
	result.PID = status.State.PID
	result.Address = status.State.Address
	result.URL = status.State.URL
	result.StartedAt = status.State.StartedAt.Format(time.RFC3339)
	result.UptimeSeconds = int64(time.Since(status.State.StartedAt).Seconds())
	result.Version = status.State.Version
	return result
}

func renderDaemonStatus(status daemon.Status) humanRenderer {
	return func(out io.Writer) error {
		if !status.Supported {
			return renderMessage("The PRX daemon requires macOS. Run prx serve directly instead.")(out)
		}
		value := daemonStatus(status)
		fields := [][2]string{
			{"Installed", yesNo(value.Installed)},
			{"LaunchAgent", value.PlistStatus + " (" + displayValue(value.PlistPath) + ")"},
			{"Log", displayValue(value.LogPath)},
			{"Running", yesNo(value.Running)},
		}
		if value.Running {
			fields = append(fields,
				[2]string{"URL", displayValue(value.URL)},
				[2]string{"PID", fmt.Sprint(value.PID)},
				[2]string{"Started", displayValue(value.StartedAt)},
				[2]string{"Uptime", fmt.Sprintf("%d seconds", value.UptimeSeconds)},
				[2]string{"Version", displayValue(value.Version)},
				[2]string{"Binary matches", yesNo(value.BinaryMatches)},
			)
		}
		if err := writeFields(out, fields); err != nil {
			return err
		}
		return writeDaemonAdvice(out, status)
	}
}

// writeDaemonAdvice は次の 1 手を示す。陳腐化した plist は install を再実行するまで
// 古いバイナリやパスを起動し続けるので、状態の表示だけでは足りない。
func writeDaemonAdvice(out io.Writer, status daemon.Status) error {
	switch {
	case !status.Installed:
		return renderMessage("\nRun prx daemon install to start PRX at login.")(out)
	case status.PlistStatus == launchd.PlistStale:
		return renderMessage("\nThe LaunchAgent is out of date. Run prx daemon install to rewrite it.")(out)
	case !status.Running:
		return renderMessage("\nNo server is running. Run prx daemon start.")(out)
	case !status.BinaryMatches:
		return renderMessage("\nThe running server predates this binary. Run prx daemon restart.")(out)
	default:
		return nil
	}
}

func (s *state) daemonInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Register the LaunchAgent that starts PRX at login",
		Long: "Register the LaunchAgent that starts PRX at login.\n\n" +
			"launchd starts the server right away, so the command waits until that server is listening " +
			"and reports its address.",
		Example: "prx daemon install",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := launchd.New()
			before := daemon.Inspect(manager, prx.Version())
			if err := manager.Install(cmd.Context()); err != nil {
				return daemonError(err)
			}
			// 手で起動したサーバーが動いていると、launchd が起動する serve は flock を
			// 取れずに終わり続ける。exit 0 で抜けるので再起動ループにはならないが、
			// LaunchAgent は稼働中のサーバーを置き換えない。
			if before.Running && !before.State.LaunchdManaged {
				_, _ = fmt.Fprintln(
					s.errOut,
					"Warning: a PRX server started outside launchd is running; "+
						"stop it and run prx daemon start to hand it over",
				)
			}
			path, _ := manager.PlistPath()
			// plist は RunAtLoad なので bootstrap は起動も伴う。依頼の成功は稼働の証明に
			// ならないので、続く `prx open` が空振りしないよう記録が書かれるまで待つ。
			state, err := waitForRunState(cmd.Context(), func(runstate.State) bool { return true })
			if err != nil {
				return err
			}
			return s.write(
				map[string]any{
					"installed": true, "label": launchd.Label, "plist_path": path,
					"address": state.Address, "url": state.URL,
				},
				renderMessage("Installed %s at %s. PRX is listening on %s.", launchd.Label, path, state.URL),
			)
		},
	}
}

func (s *state) daemonUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "uninstall",
		Short:   "Remove the LaunchAgent and stop starting PRX at login",
		Example: "prx daemon uninstall",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := launchd.New()
			path, _ := manager.PlistPath()
			if err := manager.Uninstall(cmd.Context()); err != nil {
				return daemonError(err)
			}
			return s.write(
				map[string]any{"uninstalled": true, "label": launchd.Label, "plist_path": path},
				renderMessage("Removed %s.", launchd.Label),
			)
		},
	}
}

func (s *state) daemonStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Ask launchd to start the background server",
		Long: "Ask launchd to start the background server.\n\n" +
			"Starting an already running server succeeds without replacing it.",
		Example: "prx daemon start",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := launchd.New()
			if !manager.Supported() {
				return daemonError(launchd.ErrUnsupported)
			}
			if status, state, err := runstate.Read(); err == nil && status == runstate.StatusRunning {
				return s.write(startResponse(state, true), renderMessage("PRX is already running on %s.", state.URL))
			}
			if err := manager.Start(cmd.Context()); err != nil {
				return daemonError(err)
			}
			state, err := waitForRunState(cmd.Context(), func(runstate.State) bool { return true })
			if err != nil {
				return err
			}
			return s.write(startResponse(state, false), renderMessage("PRX is listening on %s.", state.URL))
		},
	}
}

func startResponse(state runstate.State, alreadyRunning bool) map[string]any {
	return map[string]any{
		"started":         !alreadyRunning,
		"already_running": alreadyRunning,
		"address":         state.Address,
		"url":             state.URL,
		"pid":             state.PID,
	}
}

func (s *state) daemonStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the background server without removing the LaunchAgent",
		Long: "Stop the background server without removing the LaunchAgent.\n\n" +
			"PRX sends SIGTERM to the recorded process; launchctl bootout is not used because it " +
			"would unregister the LaunchAgent until the next login.",
		Example: "prx daemon stop",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := launchd.New()
			if !manager.Supported() {
				return daemonError(launchd.ErrUnsupported)
			}
			status, state, err := runstate.Read()
			if err != nil {
				return domain.NewError(domain.DomainErrorCodeInternal, "%s", err)
			}
			if status != runstate.StatusRunning {
				return s.write(
					map[string]any{"stopped": false, "already_stopped": true},
					renderMessage("No PRX server is running."),
				)
			}
			// pid の信頼性はロックが保証する。ロックを持ったまま生きているプロセスだけが
			// この記録を書けるので、pid 再利用を掴むことはない。
			if err := syscall.Kill(state.PID, syscall.SIGTERM); err != nil {
				return domain.NewError(domain.DomainErrorCodeDaemonFailed, "stop process %d: %s", state.PID, err)
			}
			if err := waitForStop(cmd.Context()); err != nil {
				return err
			}
			return s.write(
				map[string]any{"stopped": true, "already_stopped": false},
				renderMessage("Stopped the PRX server."),
			)
		},
	}
}

func (s *state) daemonRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Replace the background server with a fresh one",
		Long: "Replace the background server with a fresh one.\n\n" +
			"Use this after installing a new PRX binary so the server stops answering with an older " +
			"embedded schema.",
		Example: "prx daemon restart",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := launchd.New()
			if !manager.Supported() {
				return daemonError(launchd.ErrUnsupported)
			}
			previous := 0
			if status, state, err := runstate.Read(); err == nil && status == runstate.StatusRunning {
				previous = state.PID
			}
			if err := manager.Kickstart(cmd.Context()); err != nil {
				return daemonError(err)
			}
			state, err := waitForRunState(cmd.Context(), func(state runstate.State) bool {
				return state.PID != previous
			})
			if err != nil {
				return err
			}
			return s.write(
				map[string]any{"restarted": true, "address": state.Address, "url": state.URL, "pid": state.PID},
				renderMessage("Restarted the PRX server on %s.", state.URL),
			)
		},
	}
}

// waitForRunState は launchd が起こしたサーバーが記録を書くまで待つ。launchd への依頼は
// 非同期なので、依頼の成功は稼働の証明にならない。
func waitForRunState(ctx context.Context, accept func(runstate.State) bool) (runstate.State, error) {
	deadline := time.Now().Add(daemonWaitTimeout)
	for {
		status, state, err := runstate.Read()
		if err == nil && status == runstate.StatusRunning && accept(state) {
			return state, nil
		}
		if !time.Now().Before(deadline) {
			return runstate.State{}, domain.NewError(
				domain.DomainErrorCodeDaemonFailed,
				"launchd was asked to start %s but no server answered within %s",
				launchd.Label,
				daemonWaitTimeout,
			)
		}
		if err := sleepUntil(ctx, daemonPollInterval); err != nil {
			return runstate.State{}, err
		}
	}
}

func waitForStop(ctx context.Context) error {
	deadline := time.Now().Add(daemonWaitTimeout)
	for {
		status, _, err := runstate.Read()
		if err == nil && status == runstate.StatusNotRunning {
			return nil
		}
		if !time.Now().Before(deadline) {
			return domain.NewError(
				domain.DomainErrorCodeDaemonFailed, "the PRX server did not stop within %s", daemonWaitTimeout,
			)
		}
		if err := sleepUntil(ctx, daemonPollInterval); err != nil {
			return err
		}
	}
}

func sleepUntil(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// daemonError は LaunchAgent の失敗を、読み手が次の 1 手を選べるコードに写す。
func daemonError(err error) error {
	switch {
	case errors.Is(err, launchd.ErrUnsupported):
		return domain.NewError(
			domain.DomainErrorCodeDaemonUnsupported, "the PRX daemon requires macOS; run prx serve instead",
		)
	case errors.Is(err, launchd.ErrServiceMissing):
		return domain.NewError(
			domain.DomainErrorCodeDaemonNotInstalled, "the PRX LaunchAgent is not installed; run prx daemon install",
		)
	default:
		return domain.NewError(domain.DomainErrorCodeDaemonFailed, "%s", err)
	}
}
