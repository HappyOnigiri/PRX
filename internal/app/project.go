package app

import (
	"context"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *Service) CreateProject(ctx context.Context, title, description string) (domain.Project, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Project{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "project title is required")
	}
	return s.repository.CreateProject(ctx, title, strings.TrimSpace(description))
}

// UpdateProject は呼び出し側が指定した全フィールドを適用する。nil ポインタは
// 未指定を意味し、空文字列はその値を消す要求を意味する。
func (s *Service) UpdateProject(
	ctx context.Context,
	id string,
	update domain.ProjectUpdate,
) (domain.Project, error) {
	project, err := s.ResolveProject(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	if !archivedProjectFlagOnly(update) {
		if err := s.guardProject(project); err != nil {
			return domain.Project{}, err
		}
	}
	if update.Title != nil {
		project.Title = strings.TrimSpace(*update.Title)
	}
	if update.Description != nil {
		project.Description = *update.Description
	}
	if update.Archived != nil {
		project.Archived = *update.Archived
	}
	if project.Title == "" {
		return domain.Project{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "project title is required")
	}
	return s.repository.UpdateProject(ctx, project)
}

// ResolveProject は公開 ID で project を引く。project を指すオペランドの入口を
// 1 つに定め、行がないときは常に呼び出し側が渡したオペランドで報告するために存在する。
func (s *Service) ResolveProject(ctx context.Context, id string) (domain.Project, error) {
	return s.repository.GetProject(ctx, id)
}

// DeleteProject はコンテナを削除する。削除はアーカイブ済み project でも受け付ける
// 操作の 1 つなので、意図的にガードしていない。
func (s *Service) DeleteProject(ctx context.Context, id string, cascade bool) error {
	project, err := s.ResolveProject(ctx, id)
	if err != nil {
		return err
	}
	return s.repository.DeleteProject(ctx, project.ID, cascade)
}
