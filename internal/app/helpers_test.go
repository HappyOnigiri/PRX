package app_test

import (
	"context"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

// newFeature は feature を、所属先の project ごと作る。コンテナではなく feature 自体を
// 対象とするテスト向け。
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
