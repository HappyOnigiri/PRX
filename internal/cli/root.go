package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/runstate"
)

// automaticSyncTimeout は通常のコマンド実行前に走る日和見的な更新の上限。
// 超過した場合は自動同期の失敗として記録し、コマンド自体は成功のままにする。
const automaticSyncTimeout = 30 * time.Second

type state struct {
	dbPath           string
	dbPathSource     string
	configPath       string
	configPathSource string
	json             bool
	fixture          string
	demo             bool
	out              io.Writer
	errOut           io.Writer
	runStarted       bool
	openService      OpenService
	service          Service
	// serviceOpenErr は `debug` が失敗せずに報告してよいエラーを保持する。
	// 壊れたインストールを説明するコマンドは動き続ける必要がある。
	serviceOpenErr error
	closer         io.Closer
	closeOnce      sync.Once
	closeErr       error
	standardHelp   func(*cobra.Command, []string)
	// runLock は稼働中サーバーの記録を保持するロック。多重起動の防止と稼働発見を
	// 1 つのファイルに統合しているので、`serve` の実行中だけ保持する。
	runLock *runstate.Lock
	// serveLockRefused は別のサーバーがロックを持っていたことを表す。
	serveLockRefused bool
}

// closeService はサービスのオープンで確保した資源を解放する。デモは一時ディレクトリを
// 確保し、Cobra はコマンドがエラーを返すと post-run フックを飛ばすため、
// 失敗経路からも到達できる必要がある。
func (s *state) closeService() error {
	s.closeOnce.Do(func() {
		if s.closer != nil {
			s.closeErr = s.closer.Close()
		}
		if s.runLock != nil {
			s.closeErr = errors.Join(s.closeErr, s.runLock.Release())
			s.runLock = nil
		}
	})
	return s.closeErr
}

func NewRoot(out, errOut io.Writer, openService OpenService) *cobra.Command {
	root, _ := newRootWithState(out, errOut, openService)
	return root
}

func newRootWithState(out, errOut io.Writer, openService OpenService) (*cobra.Command, *state) {
	s := &state{out: out, errOut: errOut, openService: openService}
	root := &cobra.Command{
		Use:           "prx",
		Short:         "Manage pull-request dependency roadmaps",
		Version:       prx.Version(),
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if s.demo {
				for _, name := range []string{"db", "config", "github-fixture"} {
					if cmd.Flags().Changed(name) {
						return fmt.Errorf("--demo cannot be used with --%s", name)
					}
				}
				s.dbPath = ""
				s.configPath = ""
				s.fixture = ""
				s.dbPathSource = "demo"
				s.configPathSource = "demo"
			} else {
				s.applyEnvironmentPaths()
			}
			if cmd.Name() == "help" || cmd.Name() == "schema-version" {
				return nil
			}
			// daemon と open は launchd と稼働記録だけを見る。設定を開くと flock を取り
			// 警告も出すので、docs/design/daemon.md の「設定も開かない」と食い違う。
			if isOfflineCommand(cmd) {
				return nil
			}
			// デモは通常の設定を読まないので、それを警告してもデモ実行が
			// 使わないファイルについて報告することになる。
			if !s.demo {
				s.warnAboutConfiguration()
			}
			if isConfigCommand(cmd) {
				return nil
			}
			// ロックの取得だけは openService より前に済ませる。ロックを取れなかった
			// serve はデータベースを開かずに終わる。
			if isManagedServe(cmd, s.demo) {
				if err := s.acquireRunLock(); err != nil {
					return err
				}
				if s.serveLockRefused {
					return nil
				}
			}
			baseContext := cmd.Context()
			if baseContext == nil {
				baseContext = context.Background()
			}
			openContext := config.WithPath(baseContext, s.configPath)
			live := cmd.Name() == "serve" || cmd.Name() == "sync"
			service, closer, err := s.openService(openContext, ServiceOptions{
				DatabasePath:       s.dbPath,
				DatabasePathSource: s.dbPathSource,
				ConfigPathSource:   s.configPathSource,
				FixturePath:        s.fixture,
				Live:               live,
				Demo:               s.demo,
			})
			if err != nil {
				// 壊れたインストールこそ診断レポートが必要な場面なので、`debug` は
				// 失敗を保持し、コマンドを失敗させる代わりに storage セクションとして
				// 報告する。
				if !isDebugCommand(cmd) {
					return domain.NewError(domain.DomainErrorCodeInternal, "%s", err)
				}
				s.serviceOpenErr = err
				return nil
			}
			s.service = service
			s.closer = closer
			if !s.demo && !isAutomaticSyncExcluded(cmd) && service != nil {
				// リクエスト単位のクライアントタイムアウトは実行全体を縛らないので、
				// 到達できないホストがあると、認証情報の探索とリクエストにかかる時間だけ
				// 通常のコマンドが止まってしまう。
				syncContext, cancel := context.WithTimeout(baseContext, automaticSyncTimeout)
				_, _, _ = service.SyncIfDue(syncContext)
				cancel()
			}
			return nil
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			_ = s.closeService()
		},
	}
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVar(&s.dbPath, "db", "", "SQLite database path (env: PRX_DB)")
	root.PersistentFlags().
		StringVar(&s.configPath, "config", "", "YAML configuration path (env: PRX_CONFIG)")
	root.PersistentFlags().BoolVar(&s.json, "json", false, "output JSON")
	root.PersistentFlags().StringVar(&s.fixture, "github-fixture", "", "GitHub fixture JSON path, or demo")
	s.addCommands(root)
	s.standardHelp = root.HelpFunc()
	root.SetHelpFunc(s.writeHelp)
	root.SetHelpCommand(s.helpCommand(root))
	markCommandExecution(root, s)
	return root, s
}

// addCommands は全コマンドの登録をまとめ、ルートコマンドの構築を読みやすく保つ。
// 2 回の呼び出しはリソース系と全リポジトリ系をこのファイルの読み手向けに分けたもので、
// 実際のヘルプは名前順に並ぶ。
func (s *state) addCommands(root *cobra.Command) {
	root.AddCommand(
		s.schemaVersionCommand(),
		s.projectCommand(),
		s.featureCommand(),
		s.taskCommand(),
		s.showCommand(),
		s.dependencyCommand(),
		s.pullRequestCommand(),
		s.documentCommand(),
		s.planCommand(),
		s.promptCommand(),
		s.configCommand(),
	)
	root.AddCommand(
		s.snapshotCommand(),
		s.graphCommand(),
		s.queueCommand("ready"),
		s.queueCommand("reviews"),
		s.queueCommand("conflicts"),
		s.queueCommand("stale"),
		s.syncCommand(),
		s.validateCommand(),
		s.debugCommand(),
		s.setupCommand(),
		s.serveCommand(),
		s.daemonCommand(),
		s.openCommand(),
	)
}

// helpCommand は既定の help コマンドを置き換える。既定版は未知のトピックでも stdout に
// 出力して成功終了してしまう。エラーを返せば全ての失敗が共通のエラー経路に乗り、
// JSON の呼び出し側は 1 つの形だけを解釈すればよくなる。
func (s *state) helpCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long:  "Help provides help for any command in the application.",
		RunE: func(_ *cobra.Command, args []string) error {
			target, _, err := root.Find(args)
			if err != nil {
				return err
			}
			if target == nil {
				target = root
			}
			s.writeHelp(target, args)
			return nil
		},
	}
}

// applyEnvironmentPaths は環境変数のフォールバックをフラグの既定値としてではなく
// 実行時に解決する。フラグの既定値は、失敗時のヒントに含まれるヘルプに出てしまうため。
func (s *state) applyEnvironmentPaths() {
	s.dbPath, s.dbPathSource = resolvePathSource(s.dbPath, "PRX_DB")
	s.configPath, s.configPathSource = resolvePathSource(s.configPath, "PRX_CONFIG")
}

// resolvePathSource は場所を決めたのがフラグ・環境変数・既定値のどれかを記録する。
// フォールバックは空のフラグ値を上書きするので、後段ではこの 3 つを区別できない。
func resolvePathSource(value, variable string) (resolved, source string) {
	if value != "" {
		return value, "flag"
	}
	if fromEnvironment := os.Getenv(variable); fromEnvironment != "" {
		return fromEnvironment, "env"
	}
	return "", "default"
}

// warnAboutConfiguration は回復可能な設定の問題を、どのコマンドが設定を読むより前に
// 実行あたり 1 回報告する。読み込み失敗はここでは黙って通し、設定を必要とする
// コマンド自身にエラーを持たせる。
func (s *state) warnAboutConfiguration() {
	store, err := config.NewStore(s.configPath)
	if err != nil {
		return
	}
	_, warnings, err := store.LoadWithWarnings()
	if err != nil {
		return
	}
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(s.errOut, "Warning: %s: %s\n", store.Path(), warning)
	}
}

func markCommandExecution(command *cobra.Command, s *state) {
	if command.RunE != nil {
		run := command.RunE
		command.RunE = func(cmd *cobra.Command, args []string) error {
			s.runStarted = true
			return run(cmd, args)
		}
	}
	for _, child := range command.Commands() {
		markCommandExecution(child, s)
	}
}

// isManagedServe は flock と run state を伴う通常の serve を判定する。--addr 指定と demo は
// アドホックなインスタンスで、稼働中サーバーを名乗らないのでどちらも記録しない。
func isManagedServe(command *cobra.Command, demo bool) bool {
	return command.Name() == "serve" && !demo && !command.Flags().Changed("addr")
}

func (s *state) acquireRunLock() error {
	lock, held, err := runstate.Acquire()
	if err != nil {
		return domain.NewError(domain.DomainErrorCodeInternal, "%s", err)
	}
	if !held {
		s.serveLockRefused = true
		return nil
	}
	s.runLock = lock
	return nil
}

// offlineCommandName はデータベースを開かないコマンド群のうち cmd が属するものの名前を
// 返す。これらは launchd と稼働記録だけを見るので、ストレージを触る理由がない。
func offlineCommandName(command *cobra.Command) string {
	for current := command; current != nil; current = current.Parent() {
		if current.Name() == "daemon" || current.Name() == "open" || current.Name() == "setup" {
			return current.Name()
		}
	}
	return ""
}

func isOfflineCommand(command *cobra.Command) bool { return offlineCommandName(command) != "" }

func isConfigCommand(command *cobra.Command) bool {
	for current := command; current != nil; current = current.Parent() {
		if current.Name() == "config" {
			return true
		}
	}
	return false
}

// isAutomaticSyncExcluded は日和見的な更新を飛ばすべきコマンドを判定する。`sync` は
// 更新自体を行い、`debug` は記録済みの同期状態を報告するので、更新すると読む前に
// 上書きしてしまう。
func isAutomaticSyncExcluded(command *cobra.Command) bool {
	for current := command; current != nil; current = current.Parent() {
		if current.Name() == "sync" || current.Name() == "debug" {
			return true
		}
	}
	return false
}

func isDebugCommand(command *cobra.Command) bool {
	for current := command; current != nil; current = current.Parent() {
		if current.Name() == "debug" {
			return true
		}
	}
	return false
}

// Execute は CLI を実行し、解析済みの --json フラグに従ってエラーを整形する。
// os.Args から判断すると --json=true を取りこぼし、たまたま同じ文字列になった
// フラグ値を誤読する。
func Execute(
	ctx context.Context,
	args []string,
	out, errOut io.Writer,
	openService OpenService,
) (err error) {
	root, s := newRootWithState(out, errOut, openService)
	defer func() { err = errors.Join(err, s.closeService()) }()
	s.preScanOutputFlags(root, args)
	root.SetArgs(args)
	failedCommand, err := root.ExecuteContextC(ctx)
	if err == nil {
		return nil
	}
	if !s.runStarted {
		err = domainUsageError(err)
	}
	if failedCommand == nil {
		failedCommand = root
	}
	hint := s.renderHelp(failedCommand)
	if printErr := s.writeError(err, hint); printErr != nil {
		_, _ = fmt.Fprintln(errOut, "Error:", err)
	}
	// エラーを表示してから待つ。ログには失敗の理由が先に並び、待機は launchd への
	// 再起動間隔としてだけ効く。
	s.delayFailedServeExit(ctx, failedCommand, launchd.New(), serveFailureExitDelay)
	return err
}

func (s *state) writeHelp(command *cobra.Command, _ []string) {
	hint := s.renderHelp(command)
	if s.json {
		_ = encodeJSON(s.out, map[string]string{"hint": hint})
		return
	}
	_, _ = io.WriteString(s.out, hint)
}

func (s *state) renderHelp(command *cobra.Command) string {
	var buffer bytes.Buffer
	previousOut := command.OutOrStdout()
	command.SetOut(&buffer)
	s.standardHelp(command, nil)
	command.SetOut(previousOut)
	return buffer.String()
}

// preScanOutputFlags は Cobra がコマンド解決を完了できないときにも、明示された出力形式の
// 指定を保つ。値を取ると分かっているフラグは次の引数を消費するので、値としての
// --json をフラグと取り違えない。
func (s *state) preScanOutputFlags(root *cobra.Command, args []string) {
	current := root
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			return
		}
		if strings.HasPrefix(arg, "--") {
			name, rawValue, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
			switch name {
			case "json":
				value := true
				if hasValue {
					parsed, err := strconv.ParseBool(rawValue)
					if err != nil {
						continue
					}
					value = parsed
				}
				s.json = value
			default:
				if !hasValue && longFlagTakesValue(current, name) && index+1 < len(args) {
					index++
				}
			}
			continue
		}
		if len(arg) == 2 && arg[0] == '-' && shortFlagTakesValue(current, string(arg[1])) && index+1 < len(args) {
			index++
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			if child := directChild(current, arg); child != nil {
				current = child
			}
		}
	}
}

func longFlagTakesValue(command *cobra.Command, name string) bool {
	flag := command.Flag(name)
	return flag != nil && flag.NoOptDefVal == ""
}

func shortFlagTakesValue(command *cobra.Command, shorthand string) bool {
	for current := command; current != nil; current = current.Parent() {
		if flag := current.LocalNonPersistentFlags().ShorthandLookup(shorthand); flag != nil {
			return flag.NoOptDefVal == ""
		}
		if flag := current.PersistentFlags().ShorthandLookup(shorthand); flag != nil {
			return flag.NoOptDefVal == ""
		}
	}
	return false
}

func directChild(command *cobra.Command, name string) *cobra.Command {
	for _, child := range command.Commands() {
		if child.Name() == name || child.HasAlias(name) {
			return child
		}
	}
	return nil
}

func domainUsageError(err error) error {
	if err == nil {
		return nil
	}
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return err
	}
	var usageErr *usageError
	if errors.As(err, &usageErr) {
		return err
	}
	return &usageError{err: err}
}

type usageError struct {
	err error
}

func (e *usageError) Error() string { return e.err.Error() }

func (e *usageError) Unwrap() error { return e.err }
