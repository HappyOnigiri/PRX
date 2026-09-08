package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/browser"
	"github.com/HappyOnigiri/PRX/internal/daemon"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/runstate"
	"github.com/HappyOnigiri/PRX/internal/tui"
)

// setupIsTerminal は差し替えられるよう変数にする。go test の stdin は端末ではないため、テストは判定だけを置き換える。
var setupIsTerminal = tui.IsTerminal

var setupOpenTTY = func() (*os.File, error) { return os.Open("/dev/tty") }

type setupAction string

const (
	setupInstall setupAction = "install"
	setupUpdate  setupAction = "update"
	setupStart   setupAction = "start"
	setupSkip    setupAction = "skip"
	setupOpen    setupAction = "open"
	setupClose   setupAction = "close"
)

type setupSession struct {
	input  io.Reader
	out    io.Writer
	errOut io.Writer
}

func (s *state) setupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Choose how PRX should start",
		Long: "Choose how PRX should start. On macOS, the setup walk can register the " +
			"LaunchAgent, start the background server, and open the WebUI.",
		Example: "prx setup",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if s.json {
				return errors.New("prx setup does not support --json")
			}
			return s.runSetup(cmd.Context())
		},
	}
}

func (s *state) runSetup(ctx context.Context) error {
	status := daemon.Inspect(launchd.New(), prx.Version())
	if !status.Supported {
		return renderMessage("The PRX daemon requires macOS. Run prx serve directly instead.")(s.out)
	}
	session, closeSession, err := interactiveSetupSession(s.out, s.errOut)
	if err != nil {
		return err
	}
	defer closeSession()

	state, installedNow, err := s.applySetupDaemon(ctx, session, status)
	if err != nil {
		return err
	}
	if !shouldOfferSetupOpen(installedNow, state) {
		return nil
	}
	return s.offerSetupOpen(ctx, session, state.URL)
}

// shouldOfferSetupOpen は daemon を新規導入した直後に URL が得られた場合だけブラウザを案内する。
func shouldOfferSetupOpen(installedNow bool, state runstate.State) bool {
	return installedNow && state.URL != ""
}

func interactiveSetupSession(out, errOut io.Writer) (setupSession, func(), error) {
	input := os.Stdin
	closeInput := func() {}
	if !setupIsTerminal(int(input.Fd())) {
		tty, err := setupOpenTTY()
		if err != nil {
			return setupSession{}, nil, errors.New("prx setup needs a terminal for its questions")
		}
		input = tty
		closeInput = func() { _ = tty.Close() }
	}
	if !setupIsTerminal(int(input.Fd())) {
		closeInput()
		return setupSession{}, nil, errors.New("prx setup needs a terminal for its questions")
	}
	return setupSession{input: input, out: out, errOut: errOut}, closeInput, nil
}

func (s *state) applySetupDaemon(
	ctx context.Context,
	session setupSession,
	status daemon.Status,
) (runstate.State, bool, error) {
	if !status.Installed || status.PlistStatus == launchd.PlistUnknown {
		action, err := selectSetupAction(ctx, session, tui.Selection{
			Title:       "Install PRX as a background service?",
			Description: "The LaunchAgent starts PRX automatically when you log in.",
			Initial:     0,
			Options: []tui.Option{
				{
					Value:       string(setupInstall),
					Label:       "Install daemon",
					Description: "start PRX at login and wait until it is ready",
				},
				{
					Value:       string(setupSkip),
					Label:       "Run manually",
					Description: "do not install a LaunchAgent; use prx serve when needed",
				},
			},
		})
		if err != nil {
			return runstate.State{}, false, err
		}
		if action == setupSkip {
			_, _ = fmt.Fprintln(session.out, "Skipped daemon installation. Run prx serve to start PRX manually.")
			return runstate.State{}, false, nil
		}
		state, err := s.installSetupDaemon(ctx, session)
		return state, true, err
	}
	if status.PlistStatus == launchd.PlistStale {
		action, err := selectSetupAction(ctx, session, tui.Selection{
			Title:       "Update the PRX background service?",
			Description: "The LaunchAgent does not match this PRX binary.",
			Initial:     0,
			Options: []tui.Option{
				{
					Value:       string(setupUpdate),
					Label:       "Update daemon",
					Description: "rewrite the LaunchAgent and start the current PRX",
				},
				{
					Value:       string(setupSkip),
					Label:       "Keep current",
					Description: "leave the existing LaunchAgent unchanged",
				},
			},
		})
		if err != nil {
			return runstate.State{}, false, err
		}
		if action == setupSkip {
			return status.State, false, nil
		}
		state, err := s.installSetupDaemon(ctx, session)
		return state, false, err
	}
	if status.Running {
		return status.State, false, nil
	}
	action, err := selectSetupAction(ctx, session, tui.Selection{
		Title:       "Start the PRX background service?",
		Description: "The LaunchAgent is installed but the server is not running.",
		Initial:     0,
		Options: []tui.Option{
			{Value: string(setupStart), Label: "Start daemon", Description: "start PRX now and wait until it is ready"},
			{Value: string(setupSkip), Label: "Leave stopped", Description: "leave the server stopped for now"},
		},
	})
	if err != nil {
		return runstate.State{}, false, err
	}
	if action == setupSkip {
		return runstate.State{}, false, nil
	}
	state, err := s.startSetupDaemon(ctx, session)
	return state, false, err
}

func (s *state) installSetupDaemon(ctx context.Context, session setupSession) (runstate.State, error) {
	manager := launchd.New()
	if err := manager.Install(ctx); err != nil {
		return runstate.State{}, daemonError(err)
	}
	state, err := waitForRunState(ctx, func(runstate.State) bool { return true })
	if err != nil {
		return runstate.State{}, err
	}
	path, _ := manager.PlistPath()
	_, _ = fmt.Fprintf(session.out, "Installed %s at %s. PRX is listening on %s.\n", launchd.Label, path, state.URL)
	return state, nil
}

func (s *state) startSetupDaemon(ctx context.Context, session setupSession) (runstate.State, error) {
	manager := launchd.New()
	if err := manager.Start(ctx); err != nil {
		return runstate.State{}, daemonError(err)
	}
	state, err := waitForRunState(ctx, func(runstate.State) bool { return true })
	if err != nil {
		return runstate.State{}, err
	}
	_, _ = fmt.Fprintf(session.out, "PRX is listening on %s.\n", state.URL)
	return state, nil
}

func (s *state) offerSetupOpen(ctx context.Context, session setupSession, url string) error {
	action, err := selectSetupAction(ctx, session, tui.Selection{
		Title:       "Open PRX in your browser?",
		Description: "The WebUI is ready at " + url + ".",
		Initial:     0,
		Options: []tui.Option{
			{Value: string(setupOpen), Label: "Open browser", Description: "open the PRX WebUI now"},
			{Value: string(setupClose), Label: "Keep terminal", Description: "leave the browser closed"},
		},
	})
	if err != nil {
		return err
	}
	if action == setupClose {
		return nil
	}
	opener := browser.New()
	if err := opener.Open(ctx, url); err != nil {
		return fmt.Errorf("open PRX in browser: %w", err)
	}
	_, _ = fmt.Fprintf(session.out, "Opened %s.\n", url)
	return nil
}

func selectSetupAction(ctx context.Context, session setupSession, selection tui.Selection) (setupAction, error) {
	answer, err := tui.Select(ctx, session.input, session.errOut, selection)
	if err != nil {
		return "", err
	}
	return setupAction(strings.TrimSpace(answer)), nil
}
