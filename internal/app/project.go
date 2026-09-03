package app

import (
	"context"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *Service) CreateProject(ctx context.Context, slug, title, description string) (domain.Project, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	title = strings.TrimSpace(title)
	if !slugPattern.MatchString(slug) {
		return domain.Project{}, domain.NewError(
			domain.DomainErrorCodeInvalidSlug,
			"slug must contain lowercase letters, numbers, and single hyphens",
		)
	}
	if title == "" {
		return domain.Project{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "project title is required")
	}
	return s.repository.CreateProject(ctx, slug, title, strings.TrimSpace(description))
}

// UpdateProject applies every field the caller supplied. A nil pointer means
// the field was omitted; an empty string is a request to clear it.
func (s *Service) UpdateProject(
	ctx context.Context,
	id string,
	slug, title, description *string,
	archived *bool,
) (domain.Project, error) {
	project, err := s.ResolveProject(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	if !unarchiveOnly(archived, slug, title, description) {
		if err := s.guardProject(project); err != nil {
			return domain.Project{}, err
		}
	}
	if slug != nil {
		project.Slug = strings.TrimSpace(strings.ToLower(*slug))
	}
	if title != nil {
		project.Title = strings.TrimSpace(*title)
	}
	if description != nil {
		project.Description = *description
	}
	if archived != nil {
		project.Archived = *archived
	}
	if !slugPattern.MatchString(project.Slug) {
		return domain.Project{}, domain.NewError(domain.DomainErrorCodeInvalidSlug, "invalid project slug")
	}
	if project.Title == "" {
		return domain.Project{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "project title is required")
	}
	return s.repository.UpdateProject(ctx, project)
}

// ResolveProject only falls through to the next lookup when the previous one
// reported a missing row, so a storage failure such as a locked database keeps
// its own cause instead of being reported as a missing project.
func (s *Service) ResolveProject(ctx context.Context, idOrSlug string) (domain.Project, error) {
	project, err := s.repository.GetProject(ctx, idOrSlug)
	if err == nil {
		return project, nil
	}
	if domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		return domain.Project{}, err
	}
	project, err = s.repository.GetProjectBySlug(ctx, idOrSlug)
	if err == nil {
		return project, nil
	}
	if domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		return domain.Project{}, err
	}
	return domain.Project{}, domain.NewError(domain.DomainErrorCodeNotFound, "project %q was not found", idOrSlug)
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
