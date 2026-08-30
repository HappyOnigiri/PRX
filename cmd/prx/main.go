package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/cli"
	"github.com/HappyOnigiri/PRX/internal/config"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func newOpenService(_ io.Writer) cli.OpenService {
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
		}
		configStore, configErr := config.NewStore(config.PathFromContext(ctx))
		if configErr != nil {
			_ = database.Close()
			return nil, nil, configErr
		}
		if live && fixturePath == "" {
			// Resolver construction is deliberately lazy: credentials are read
			// only when a repository is synchronized, so local CRUD and serve
			// startup do not require a working Keychain or gh session.
			if _, loadErr := configStore.Load(); loadErr != nil {
				_ = database.Close()
				return nil, nil, loadErr
			}
		}
		return app.NewWithConfig(database, provider, configStore), database, nil
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr, newOpenService(os.Stderr)); err != nil {
		os.Exit(1)
	}
}
