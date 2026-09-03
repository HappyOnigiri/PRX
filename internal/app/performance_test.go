package app_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func BenchmarkSnapshot5000Tasks(b *testing.B) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(b.TempDir(), "benchmark.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	provider, _ := githubprovider.NewFixtureProvider("demo")
	service := app.New(database, provider)
	for featureIndex := 0; featureIndex < 100; featureIndex++ {
		feature, err := service.CreateFeature(
			ctx,
			fmt.Sprintf("feature-%03d", featureIndex),
			fmt.Sprintf("Feature %03d", featureIndex),
			"",
			"",
		)
		if err != nil {
			b.Fatal(err)
		}
		var previous string
		for taskIndex := 0; taskIndex < 50; taskIndex++ {
			task, err := service.CreateTask(
				ctx,
				feature.ID,
				fmt.Sprintf("Task %03d/%02d", featureIndex, taskIndex),
				"",
				domain.TaskKindManual,
				"",
			)
			if err != nil {
				b.Fatal(err)
			}
			if previous != "" {
				if _, err := service.AddDependency(ctx, previous, task.ID); err != nil {
					b.Fatal(err)
				}
			}
			previous = task.ID
		}
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		snapshot, err := service.Snapshot(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if len(snapshot.Tasks) != 5000 {
			b.Fatalf("tasks=%d", len(snapshot.Tasks))
		}
	}
}
