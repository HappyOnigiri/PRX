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

// UpdateProject applies every field the caller supplied. A nil pointer means
// the field was omitted; an empty string is a request to clear it.
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

// ResolveProject looks a project up by its public ID. It exists so callers
// name one entry point for an operand that identifies a project, and so a
// missing row is always reported with the operand the caller supplied.
func (s *Service) ResolveProject(ctx context.Context, id string) (domain.Project, error) {
	return s.repository.GetProject(ctx, id)
}

// DeleteProject removes the container. Deletion is one of the operations an
// archived project still accepts, so it is deliberately unguarded.
func (s *Service) DeleteProject(ctx context.Context, id string, cascade bool) error {
	project, err := s.ResolveProject(ctx, id)
	if err != nil {
		return err
	}
	return s.repository.DeleteProject(ctx, project.ID, cascade)
}
