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

	"github.com/HappyOnigiri/PRX/internal/app"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/webui"
)

func (s *state) serveCommand() *cobra.Command {
	var address string
	command := &cobra.Command{
		Use:     "serve",
		Short:   "Start the local WebUI and ConnectRPC server",
		Example: "prx serve --addr 127.0.0.1:7331",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if s.fixture == "" {
				if provider, err := githubprovider.NewLiveProvider(cmd.Context()); err == nil {
					s.service = app.New(s.store, provider)
				} else {
					_, _ = fmt.Fprintf(s.errOut, "warning: %v\n", err)
				}
			}
			rpcPath, rpcHandler := rpc.New(s.service)
			mux := http.NewServeMux()
			mux.Handle(rpcPath, rpcHandler)
			mux.Handle("/", webui.Handler())
			listener, err := (&net.ListenConfig{}).Listen(cmd.Context(), "tcp", address)
			if err != nil {
				return err
			}
			server := &http.Server{
				Addr:              address,
				Handler:           localOnly(listener.Addr(), mux),
				ReadHeaderTimeout: 5 * time.Second,
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
	return command
}

// localOnly rejects requests whose Host or Origin header does not belong to the
// address the server listens on. Without it a page on an attacker-controlled
// domain that resolves to the loopback address is same-origin from the
// browser's point of view and can drive every mutation on the local database.
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
