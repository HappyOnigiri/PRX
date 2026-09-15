package app

import (
	"context"
	"errors"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
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
	if update.PromptOverrides != nil {
		if err := applyPromptOverrides(&project.PromptOverrides, *update.PromptOverrides, "project"); err != nil {
			return domain.Project{}, err
		}
	}
	if update.TaskLabelOverrides != nil {
		if err := applyTaskLabelOverrides(
			&project.TaskLabelOverrides,
			*update.TaskLabelOverrides,
			"project",
		); err != nil {
			return domain.Project{}, err
		}
	}
	if update.Archived != nil {
		project.Archived = *update.Archived
	}
	if project.Title == "" {
		return domain.Project{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "project title is required")
	}
	return s.repository.UpdateProject(ctx, project)
}

// applyPromptOverrides は project と feature で共有する種類別の部分更新を適用する。
// 空文字列は DB の NULL に畳まれ、次の上位スコープを継承する。
func applyPromptOverrides(
	current *domain.PromptTemplateOverrides,
	update domain.PromptTemplateOverridesUpdate,
	scope string,
) error {
	for _, item := range []struct {
		kind   prompt.Kind
		value  *string
		target *string
		name   string
	}{
		{kind: prompt.KindDesign, value: update.Design, target: &current.Design, name: "design"},
		{
			kind: prompt.KindImplementation, value: update.Implementation,
			target: &current.Implementation, name: "implementation",
		},
		{kind: prompt.KindBatch, value: update.Batch, target: &current.Batch, name: "batch"},
	} {
		if item.value == nil {
			continue
		}
		if err := prompt.ValidateOverride(item.kind, *item.value); err != nil {
			var typed *prompt.Error
			if errors.As(err, &typed) {
				return domain.NewError(
					domain.DomainErrorCodeInvalidPromptTemplate,
					"%s prompt override %s: %s",
					scope,
					item.name,
					typed.Message,
				)
			}
			return domain.NewError(
				domain.DomainErrorCodeInvalidPromptTemplate,
				"%s prompt override %s: %s",
				scope, item.name, err,
			)
		}
		*item.target = *item.value
	}
	return nil
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
