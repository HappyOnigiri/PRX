package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/cli"
	"github.com/HappyOnigiri/PRX/internal/config"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func newOpenService(_ io.Writer) cli.OpenService {
	return func(ctx context.Context, options cli.ServiceOptions) (cli.Service, io.Closer, error) {
		dbPath := options.DatabasePath
		fixturePath := options.FixturePath
		configPath := config.PathFromContext(ctx)
		var temporaryRoot string
		// The default location is resolved here rather than inside the store so
		// a failed open can still report which database was attempted.
		if dbPath == "" && !options.Demo {
			if resolved, resolveErr := store.DefaultPath(); resolveErr == nil {
				dbPath = resolved
			}
		}
		if options.Demo {
			var err error
			temporaryRoot, err = os.MkdirTemp("", "prx-demo-")
			if err != nil {
				return nil, nil, err
			}
			dbPath = filepath.Join(temporaryRoot, "prx.db")
			configPath = filepath.Join(temporaryRoot, "config.yaml")
			fixturePath = filepath.Join(temporaryRoot, "github-fixture.json")
			if err := app.WriteDemoFixture(fixturePath); err != nil {
				_ = os.RemoveAll(temporaryRoot)
				return nil, nil, err
			}
		}

		database, err := store.Open(ctx, dbPath)
		if err != nil {
			if temporaryRoot != "" {
				_ = os.RemoveAll(temporaryRoot)
			}
			return nil, nil, &cli.ServiceOpenError{DatabasePath: dbPath, Err: err}
		}
		closer := &serviceCloser{database: database, temporaryRoot: temporaryRoot}

		var provider githubprovider.Provider
		if fixturePath != "" {
			provider, err = githubprovider.NewFixtureProvider(fixturePath)
			if err != nil {
				_ = closer.Close()
				return nil, nil, err
			}
		}
		configStore, configErr := config.NewStore(configPath)
		if configErr != nil {
			_ = closer.Close()
			return nil, nil, configErr
		}
		if options.Live && fixturePath == "" {
			// Resolver construction is deliberately lazy: credentials are read
			// only when a repository is synchronized, so local CRUD and serve
			// startup do not require a working Keychain or gh session.
			if _, loadErr := configStore.Load(); loadErr != nil {
				_ = closer.Close()
				return nil, nil, loadErr
			}
		}
		service := app.NewWithConfig(database, provider, configStore)
		service.SetProcessInfo(app.ProcessInfo{
			Mode:               "cli",
			Demo:               options.Demo,
			GitHubFixture:      fixturePath != "",
			DatabasePath:       dbPath,
			DatabasePathSource: options.DatabasePathSource,
			ConfigPathSource:   options.ConfigPathSource,
		})
		if options.Demo {
			markdownPath := filepath.Join(temporaryRoot, "walkthrough.md")
			if err := service.InitializeDemo(ctx, markdownPath); err != nil {
				_ = closer.Close()
				return nil, nil, err
			}
		}
		return service, closer, nil
	}
}

type serviceCloser struct {
	database      io.Closer
	temporaryRoot string
}

func (c *serviceCloser) Close() error {
	closeErr := c.database.Close()
	if c.temporaryRoot == "" {
		return closeErr
	}
	removeErr := os.RemoveAll(c.temporaryRoot)
	return errors.Join(closeErr, removeErr)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr, newOpenService(os.Stderr)); err != nil {
		os.Exit(1)
	}
}
