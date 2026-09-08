package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/cli"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/daemon"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func newOpenService(_ io.Writer) cli.OpenService {
	return func(ctx context.Context, options cli.ServiceOptions) (cli.Service, io.Closer, error) {
		dbPath := options.DatabasePath
		fixturePath := options.FixturePath
		configPath := config.PathFromContext(ctx)
		var temporaryRoot string
		// 既定の場所を store 内ではなくここで解決するのは、オープンに失敗しても
		// どのデータベースを試したか報告できるようにするため。
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
			// resolver の構築は意図的に遅延させる。認証情報はリポジトリ同期時に
			// しか読まないので、ローカルの CRUD や serve の起動に Keychain や
			// gh セッションが動いている必要はない。
			if _, loadErr := configStore.Load(); loadErr != nil {
				_ = closer.Close()
				return nil, nil, loadErr
			}
		}
		service := app.NewWithConfig(database, provider, configStore)
		// 常駐の観測は launchd と稼働記録を束ねた実装で注入する。app はどちらも
		// import しないので、配線層だけが知る事実としてここで渡す。
		service.SetDaemonInspector(func(context.Context) domain.DebugDaemonInput {
			return daemon.Inspect(launchd.New(), prx.Version()).DebugInput(options.Demo)
		})
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
