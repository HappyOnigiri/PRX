package app_test

import (
	"context"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

// newFeature creates a feature together with the project it has to belong to,
// for the tests whose subject is the feature rather than its container.
func newFeature(
	t *testing.T,
	ctx context.Context,
	service *app.Service,
	title string,
) domain.Feature {
	t.Helper()
	project, err := service.CreateProject(ctx, title+" project", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err := service.CreateFeature(ctx, title, "", project.ID)
	if err != nil {
		t.Fatal(err)
	}
	return feature
}
