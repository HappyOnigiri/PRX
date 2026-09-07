package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/webui"
)

// serveEndpointRecorder はアプリケーションサービス側で実装する。CLI はそのパッケージを
// import できないので、アドレスはプリミティブ型のまま境界を越える。
type serveEndpointRecorder interface {
	SetServeEndpoint(address string, startedAt time.Time)
}

func (s *state) serveCommand() *cobra.Command {
	var address string
	command := &cobra.Command{
		Use:     "serve",
		Short:   "Start the local WebUI and ConnectRPC server",
		Example: "prx serve --demo",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rpcPath, rpcHandler := rpc.New(s.service)
			mux := http.NewServeMux()
			mux.Handle(rpcPath, rpcHandler)
			mux.Handle("/", webui.Handler(prx.Version(), s.demo))
			listener, err := (&net.ListenConfig{}).Listen(cmd.Context(), "tcp", address)
			if err != nil {
				return err
			}
			// 実際に受け付けたアドレスはこの時点で初めて分かり、--addr 127.0.0.1:0 では
			// 要求したものと異なる。診断レポートにはサーバーが実際に応答している
			// アドレスが必要。
			if recorder, ok := s.service.(serveEndpointRecorder); ok {
				recorder.SetServeEndpoint(listener.Addr().String(), time.Now().UTC())
			}
			server := &http.Server{
				Addr:              address,
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
			err = server.Serve(listener)
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		},
	}
	command.Flags().StringVar(&address, "addr", "127.0.0.1:7331", "listen address")
	command.Flags().BoolVar(&s.demo, "demo", false, "start with isolated temporary demo data")
	return command
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
