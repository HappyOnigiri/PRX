package app

import (
	"context"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// Archiving is a write barrier, not a presentation flag. The guards below are
// the only place that decides what an archived project or feature refuses, so
// the CLI, the RPC handlers, and the WebUI all inherit the same rule.
//
// Each guard reads the container's state before the write and does not share a
// transaction with it, so a write that passed the barrier can still land just
// after another process archived the container. The barrier coordinates people
// and their agents rather than locking the database, and the store carries no
// constraint that would close that window.
//
// The barrier lifts for exactly three operations: changing nothing but the
// archived flag, deleting a project or a feature, and a GitHub refresh that
// names a feature or a task explicitly. Deletion stays available because it is
// how archived work is finally discarded, and a project cascade releases its
// features by clearing their project_id, which the guards must not read as a
// forbidden reassignment.

// archivedReadOnly is the single error every refused write returns, so a caller
// branches on one code without needing to know which container is archived.
func archivedReadOnly() error {
	return domain.NewError(
		domain.DomainErrorCodeArchivedReadOnly,
		"archived projects and features are read-only; unarchive before writing",
	)
}

// archivedFeatureFlagOnly and archivedProjectFlagOnly report whether a request
// carries the archived flag and changes nothing else. Moving that flag is the
// one update an archived record accepts, in either direction, so lifting the
// archive and re-archiving a record both stay possible while a request that
// also carries another change stays refused.
//
// Each clears the flag and compares what is left with the zero value, so the
// question stays structural: a field added to the update type is covered by the
// barrier without this rule being edited.
func archivedFeatureFlagOnly(update domain.FeatureUpdate) bool {
	if update.Archived == nil {
		return false
	}
	update.Archived = nil
	return update == domain.FeatureUpdate{}
}

func archivedProjectFlagOnly(update domain.ProjectUpdate) bool {
	if update.Archived == nil {
		return false
	}
	update.Archived = nil
	return update == domain.ProjectUpdate{}
}

// featureReadOnly derives, for a feature read outside a snapshot, the value
// Snapshot publishes as Feature.ReadOnly.
func (s *Service) featureReadOnly(ctx context.Context, feature domain.Feature) (bool, error) {
	if feature.Archived {
		return true, nil
	}
	if feature.ProjectID == "" {
		return false, nil
	}
	project, err := s.repository.GetProject(ctx, feature.ProjectID)
	if err != nil {
		return false, err
	}
	return project.Archived, nil
}

// withReadOnly returns the feature with ReadOnly derived, so a feature that
// leaves the application layer outside a snapshot carries the same value
// Snapshot publishes instead of the stored zero value.
func (s *Service) withReadOnly(ctx context.Context, feature domain.Feature) (domain.Feature, error) {
	readOnly, err := s.featureReadOnly(ctx, feature)
	if err != nil {
		return domain.Feature{}, err
	}
	feature.ReadOnly = readOnly
	return feature, nil
}

// guardFeature refuses a write that lands inside an archived feature or inside
// a feature whose project is archived.
func (s *Service) guardFeature(ctx context.Context, feature domain.Feature) error {
	readOnly, err := s.featureReadOnly(ctx, feature)
	if err != nil {
		return err
	}
	if readOnly {
		return archivedReadOnly()
	}
	return nil
}

// guardTask refuses a write to a task, its dependencies, its pull request, or
// its documents when the owning feature is read-only.
func (s *Service) guardTask(ctx context.Context, task domain.Task) error {
	feature, err := s.ResolveFeature(ctx, task.FeatureID)
	if err != nil {
		return err
	}
	return s.guardFeature(ctx, feature)
}

// guardTaskID guards a write aimed at a task the caller named by public ID, and
// reports the missing task when there is none.
func (s *Service) guardTaskID(ctx context.Context, taskID string) error {
	task, err := s.repository.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	return s.guardTask(ctx, task)
}

func (s *Service) guardProject(project domain.Project) error {
	if project.Archived {
		return archivedReadOnly()
	}
	return nil
}

// guardDocument refuses a write to a document whose parent is read-only,
// whichever of the three parents the document carries.
func (s *Service) guardDocument(ctx context.Context, document domain.Document) error {
	switch {
	case document.ProjectID != "":
		project, err := s.repository.GetProject(ctx, document.ProjectID)
		if err != nil {
			return err
		}
		return s.guardProject(project)
	case document.FeatureID != "":
		feature, err := s.ResolveFeature(ctx, document.FeatureID)
		if err != nil {
			return err
		}
		return s.guardFeature(ctx, feature)
	case document.TaskID != "":
		return s.guardTaskID(ctx, document.TaskID)
	}
	return nil
}

// resolveProjectAssignment turns the requested membership into the public
// project ID to store. An empty request detaches the feature, and an archived
// project refuses to take one in.
func (s *Service) resolveProjectAssignment(ctx context.Context, requested string) (string, error) {
	if requested == "" {
		return "", nil
	}
	project, err := s.ResolveProject(ctx, requested)
	if err != nil {
		return "", err
	}
	if err := s.guardProject(project); err != nil {
		return "", err
	}
	return project.ID, nil
}
