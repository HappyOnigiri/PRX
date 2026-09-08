package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/runstate"
	"github.com/HappyOnigiri/PRX/internal/webui"
)

const (
	// defaultServePort は設定が auto のときの希望ポート。
	defaultServePort = 7331
	// demoServePort は demo の希望ポート。通常起動の常駐サーバーと衝突させない。
	demoServePort = 7332
	loopbackHost  = "127.0.0.1"
)

// executableCheckInterval はバイナリ置換を検査する間隔。置換は人の操作に伴うので、
// 検出が数十秒遅れても実害はない。
const executableCheckInterval = 30 * time.Second

// serveEndpointRecorder はアプリケーションサービス側で実装する。CLI はそのパッケージを
// import できないので、アドレスはプリミティブ型のまま境界を越える。
type serveEndpointRecorder interface {
	SetServeEndpoint(address string, startedAt time.Time)
}

// serveDatabaseReporter は開いたデータベースの位置を返す。run state の読み手が
// どのデータベースを見ているサーバーかを判断できるようにする。
type serveDatabaseReporter interface {
	DatabasePath() string
}

type listenFunc func(ctx context.Context, network, address string) (net.Listener, error)

// listenPlan は 1 回の起動での bind 先と、EADDRINUSE から逃げてよいかを表す。
type listenPlan struct {
	address  string
	fallback bool
}

func (s *state) serveCommand() *cobra.Command {
	var address string
	command := &cobra.Command{
		Use:   "serve",
		Short: "Start the local WebUI and ConnectRPC server",
		Long: fmt.Sprintf(
			"Start the local WebUI and ConnectRPC server.\n\n"+
				"Without --addr the port comes from server.port in the configuration, which defaults to %d "+
				"and falls back to an ephemeral port when that is in use.\n"+
				"--addr is the only way to listen outside loopback, and such a server is not recorded as "+
				"the running one.",
			defaultServePort,
		),
		Example: "prx serve --demo",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if s.serveLockRefused {
				// 望んだ状態（サーバーが 1 つ稼働）は満たされている。非 0 で終わると
				// KeepAlive{SuccessfulExit:false} の下で launchd が再起動ループに入る。
				_, _ = fmt.Fprintln(s.errOut, "another PRX server is already running; not starting a second one")
				return nil
			}
			return s.runServe(cmd, address)
		},
	}
	command.Flags().
		StringVar(&address, "addr", "", "listen address, overriding server.port and loopback (host:port)")
	command.Flags().BoolVar(&s.demo, "demo", false, "start with isolated temporary demo data")
	return command
}

func (s *state) runServe(cmd *cobra.Command, address string) error {
	plan, err := s.serveListenPlan(address)
	if err != nil {
		return err
	}
	listener, err := listenServe(cmd.Context(), (&net.ListenConfig{}).Listen, plan)
	if err != nil {
		return err
	}
	// 実際に受け付けたアドレスはこの時点で初めて分かり、auto や --addr 127.0.0.1:0 では
	// 要求したものと異なる。診断レポートと稼働発見にはサーバーが実際に応答している
	// アドレスが必要。
	startedAt := time.Now().UTC()
	if recorder, ok := s.service.(serveEndpointRecorder); ok {
		recorder.SetServeEndpoint(listener.Addr().String(), startedAt)
	}
	if err := s.recordRunState(listener.Addr().String(), startedAt); err != nil {
		return err
	}
	rpcPath, rpcHandler := rpc.New(s.service)
	mux := http.NewServeMux()
	mux.Handle(rpcPath, rpcHandler)
	mux.Handle("/", webui.Handler(prx.Version(), s.demo))
	server := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           localOnly(listener.Addr(), mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if s.demo {
		_, _ = fmt.Fprintln(s.errOut, "PRX demo mode uses temporary data that resets on restart.")
	}
	_, _ = fmt.Fprintf(s.errOut, "PRX listening on http://%s\n", listener.Addr())
	go func() {
		<-cmd.Context().Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	if s.runLock != nil {
		go s.watchExecutable(cmd.Context(), launchd.New())
	}
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// serveListenPlan は起動形態から bind 先を決める。--addr は設定を無視するアドホックな
// 指定で、loopback 外へ出す唯一の手段でもある。demo は設定を読まない。
func (s *state) serveListenPlan(address string) (listenPlan, error) {
	if address != "" {
		return listenPlan{address: address}, nil
	}
	if s.demo {
		return listenPlan{address: loopbackAddress(demoServePort), fallback: true}, nil
	}
	store, err := s.configStore()
	if err != nil {
		return listenPlan{}, configCommandError(err)
	}
	settings, err := store.Load()
	if err != nil {
		return listenPlan{}, configCommandError(err)
	}
	return portListenPlan(settings.Server.Port), nil
}

// portListenPlan は設定されたポートを計画に写す。auto のときだけ EADDRINUSE から
// 逃げてよい。明示されたポートで別のポートに逃げると、意図と違う場所で待ち受ける。
func portListenPlan(port config.ServerPort) listenPlan {
	if port.Auto() {
		return listenPlan{address: loopbackAddress(defaultServePort), fallback: true}
	}
	return listenPlan{address: loopbackAddress(int(port))}
}

func loopbackAddress(port int) string {
	return net.JoinHostPort(loopbackHost, strconv.Itoa(port))
}

// listenServe は計画どおりに bind する。フォールバックは EADDRINUSE のときだけで、
// EACCES（特権ポート）や EADDRNOTAVAIL で :0 に落ちると意図と違うポートで黙って起動する。
func listenServe(ctx context.Context, listen listenFunc, plan listenPlan) (net.Listener, error) {
	listener, err := listen(ctx, "tcp", plan.address)
	if err == nil {
		return listener, nil
	}
	if !errors.Is(err, syscall.EADDRINUSE) {
		return nil, err
	}
	if !plan.fallback {
		return nil, domain.NewError(domain.DomainErrorCodeAddressInUse, "%s is already in use", plan.address)
	}
	return listen(ctx, "tcp", loopbackAddress(0))
}

// recordRunState は稼働情報をロック済みのファイルへ書く。ロックを持たない起動形態
// （--addr 指定と demo）は稼働中サーバーを名乗らないので何も書かない。
func (s *state) recordRunState(address string, startedAt time.Time) error {
	if s.runLock == nil {
		return nil
	}
	executable, _ := os.Executable()
	databasePath := s.dbPath
	if reporter, ok := s.service.(serveDatabaseReporter); ok {
		databasePath = reporter.DatabasePath()
	}
	err := s.runLock.Write(runstate.State{
		PID:            os.Getpid(),
		Address:        address,
		URL:            "http://" + address,
		StartedAt:      startedAt,
		Version:        prx.Version(),
		Executable:     executable,
		DatabasePath:   databasePath,
		LaunchdManaged: launchd.New().Managed(),
	})
	if err != nil {
		return domain.NewError(domain.DomainErrorCodeInternal, "%s", err)
	}
	return nil
}

// executableRestarter は置換を検出した後に必要な launchd の操作だけを表す。テストが実
// launchctl に触れずに監視の分岐を回せるようにする。
type executableRestarter interface {
	Managed() bool
	Kickstart(ctx context.Context) error
}

// watchExecutable は自分のバイナリが置き換えられたら launchd に自身の再起動を依頼する。
// これがないと、新しい CLI がデータベースを移行した後も古いサーバーが古い埋め込み
// スキーマで応答し続けるサイレントな版ずれが残る。
func (s *state) watchExecutable(ctx context.Context, manager *launchd.Manager) {
	path, err := os.Executable()
	if err != nil {
		s.warnExecutableWatchDisabled(err)
		return
	}
	s.watchExecutablePath(ctx, manager, path, executableCheckInterval)
}

// watchExecutablePath は監視対象と検査間隔を受け取る。間隔を引数にするのはテストが実時間を
// 待たずに 1 周期を回せるようにするためである。
func (s *state) watchExecutablePath(
	ctx context.Context, manager executableRestarter, path string, interval time.Duration,
) {
	// os.Executable は起動時のパスを返し続けるので、stat の失敗を「未変更の根拠」に
	// してはいけない。基準が取れないときは監視自体を無効にする。
	baseline, err := os.Stat(path)
	if err != nil {
		s.warnExecutableWatchDisabled(err)
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		current, err := os.Stat(path)
		if err != nil {
			// install(1) は一時ファイルを rename するので、パスは一時的に消える。
			// 置換と断定せず次の周期で再検査する。
			continue
		}
		if executableMatches(baseline, current) {
			continue
		}
		if !manager.Managed() {
			// 管理外のプロセスが自己 kickstart すると、置換側が flock を取れずに死んで
			// 手動で起動したプロセスだけが残る。
			_, _ = fmt.Fprintf(
				s.errOut,
				"PRX executable %s was replaced but this server is not managed by launchd; restart it manually\n",
				path,
			)
			return
		}
		if err := s.restartAfterReplacement(manager, path); err != nil {
			// 依頼の失敗で監視を終えると、以降の置換も検査されないまま古いサーバーが
			// 残る。基準はそのままなので、次の周期で同じ置換を再依頼できる。
			_, _ = fmt.Fprintf(s.errOut, "PRX could not restart itself after the replacement: %v\n", err)
			continue
		}
		return
	}
}

// restartAfterReplacement は launchd へ置換の反映を依頼する。kickstart -k はこのプロセスへ
// SIGTERM を送るので、サーバーの ctx から派生させると依頼の完了前に取り消される。
func (s *state) restartAfterReplacement(manager executableRestarter, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = fmt.Fprintf(s.errOut, "PRX executable %s was replaced; asking launchd to restart the server\n", path)
	return manager.Kickstart(ctx)
}

func (s *state) warnExecutableWatchDisabled(err error) {
	_, _ = fmt.Fprintf(
		s.errOut,
		"PRX cannot identify its own executable, so automatic restart after a replacement is disabled: %v\n",
		err,
	)
}

// executableMatches は inode・サイズ・mtime の 3 つを比べる。inode だけでは上書きを、
// サイズと mtime だけでは同値な置換を見逃す。
func executableMatches(baseline, current os.FileInfo) bool {
	return os.SameFile(baseline, current) && baseline.Size() == current.Size() &&
		baseline.ModTime().Equal(current.ModTime())
}

// localOnly は Host や Origin ヘッダーが listen 中のアドレスに属さないリクエストを拒否する。
// そうしないと、ループバックに解決される攻撃者所有のドメインが same-origin になる。
// docs/design/security.md を参照。
func localOnly(addr net.Addr, next http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	if host, port, err := net.SplitHostPort(addr.String()); err == nil {
		for _, name := range []string{host, "127.0.0.1", "localhost", "::1"} {
			allowed[strings.ToLower(net.JoinHostPort(name, port))] = struct{}{}
		}
	}
	permitted := func(hostPort string) bool {
		_, ok := allowed[strings.ToLower(hostPort)]
		return ok
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !permitted(r.Host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || !permitted(parsed.Host) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
