package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/cli"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func newOpenService(w io.Writer) cli.OpenService {
	return func(ctx context.Context, dbPath, fixturePath string, live bool) (cli.Service, io.Closer, error) {
		database, err := store.Open(ctx, dbPath)
		if err != nil {
			return nil, nil, err
		}

		var provider githubprovider.Provider
		if fixturePath != "" {
			provider, err = githubprovider.NewFixtureProvider(fixturePath)
			if err != nil {
				_ = database.Close()
				return nil, nil, err
			}
		} else if live {
			provider, err = githubprovider.NewLiveProvider(ctx)
			if err != nil {
				// Serving without a provider is useful for local CRUD and matches
				// the historical warning-only behavior when GitHub auth is absent.
				_, _ = fmt.Fprintf(w, "warning: %v\n", err)
				provider = nil
			}
		}

		return app.New(database, provider), database, nil
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr, newOpenService(os.Stderr)); err != nil {
		os.Exit(1)
	}
}
