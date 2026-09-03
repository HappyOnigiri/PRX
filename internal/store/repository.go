package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/HappyOnigiri/PRX/internal/db"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

const (
	projectPublicIDPrefix = "P-"
	featurePublicIDPrefix = "F-"
	taskPublicIDPrefix    = "T-"
)

func nextPublicID(ctx context.Context, q *db.Queries, entity, prefix string) (string, error) {
	nextValue, err := q.IncrementIDSequence(ctx, entity)
	if err != nil {
		return "", fmt.Errorf("allocate %s ID: %w", entity, err)
	}
	return fmt.Sprintf("%s%d", prefix, nextValue-1), nil
}

func publicFeatureID(ctx context.Context, q *db.Queries, storageID string) string {
	value, err := q.GetFeature(ctx, storageID)
	if err != nil {
		return ""
	}
	return value.PublicID
}

func publicTaskID(ctx context.Context, q *db.Queries, storageID string) string {
	value, err := q.GetTask(ctx, storageID)
	if err != nil {
		return ""
	}
	return value.PublicID
}

func publicProjectID(ctx context.Context, q *db.Queries, storageID string) string {
	value, err := q.GetProject(ctx, storageID)
	if err != nil {
		return ""
	}
	return value.PublicID
}

// projectStorageID translates a public project ID into the storage UUID the
// foreign key needs. An empty public ID means no project, which the schema
// records as NULL.
func projectStorageID(ctx context.Context, q *db.Queries, publicID string) (sql.NullString, error) {
	if publicID == "" {
		return sql.NullString{}, nil
	}
	project, err := q.GetProjectByPublicID(ctx, publicID)
	if err != nil {
		return sql.NullString{}, mapNotFound(err, "project", publicID)
	}
	return sql.NullString{String: project.ID, Valid: true}, nil
}

func (s *Store) CreateProject(ctx context.Context, slug, title, description string) (domain.Project, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Project{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := db.New(tx)
	publicID, err := nextPublicID(ctx, q, "project", projectPublicIDPrefix)
	if err != nil {
		return domain.Project{}, err
	}
	now := timestamp(s.now())
	value, err := q.CreateProject(ctx, db.CreateProjectParams{
		ID:          uuid.NewString(),
		PublicID:    publicID,
		Slug:        slug,
		Title:       title,
		Description: description,
		Archived:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return domain.Project{}, fmt.Errorf("create project: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Project{}, err
	}
	return domainProject(value), nil
}

func (s *Store) GetProject(ctx context.Context, id string) (domain.Project, error) {
	value, err := db.New(s.db).GetProjectByPublicID(ctx, id)
	return domainProject(value), mapNotFound(err, "project", id)
}

func (s *Store) GetProjectBySlug(ctx context.Context, slug string) (domain.Project, error) {
	value, err := db.New(s.db).GetProjectBySlug(ctx, slug)
	return domainProject(value), mapNotFound(err, "project", slug)
}

func (s *Store) UpdateProject(ctx context.Context, project domain.Project) (domain.Project, error) {
	storageID := project.StorageID
	if storageID == "" {
		value, err := db.New(s.db).GetProjectByPublicID(ctx, project.ID)
		if err != nil {
			return domain.Project{}, mapNotFound(err, "project", project.ID)
		}
		storageID = value.ID
	}
	value, err := db.New(s.db).UpdateProject(ctx, db.UpdateProjectParams{
		Slug:        project.Slug,
		Title:       project.Title,
		Description: project.Description,
		Archived:    boolInt(project.Archived),
		UpdatedAt:   timestamp(s.now()),
		ID:          storageID,
	})
	return domainProject(value), mapNotFound(err, "project", project.ID)
}

// DeleteProject removes the container, not its contents. A cascade deletes the
// project's own documents and releases its features, because a feature has a
// longer life and a public ID of its own; `feature delete --cascade` is the
// operation that removes contained work.
func (s *Store) DeleteProject(ctx context.Context, id string, cascade bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	project, err := q.GetProjectByPublicID(ctx, id)
	if err != nil {
		return mapNotFound(err, "project", id)
	}
	storageID := sql.NullString{String: project.ID, Valid: true}
	references, err := q.CountProjectReferences(ctx, storageID)
	if err != nil {
		return err
	}
	if references > 0 && !cascade {
		return domain.NewError(
			domain.DomainErrorCodeReferencesExist,
			"project has features or documents; pass --cascade",
		)
	}
	if cascade {
		if err := q.DeleteDocumentsForProject(ctx, storageID); err != nil {
			return err
		}
		if err := q.DetachFeaturesFromProject(ctx, db.DetachFeaturesFromProjectParams{
			UpdatedAt: timestamp(s.now()),
			ProjectID: storageID,
		}); err != nil {
			return err
		}
	}
	if err := q.DeleteProject(ctx, project.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CreateFeature(
	ctx context.Context,
	slug, title, description, projectID string,
) (domain.Feature, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Feature{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := db.New(tx)
	project, err := projectStorageID(ctx, q, projectID)
	if err != nil {
		return domain.Feature{}, err
	}
	publicID, err := nextPublicID(ctx, q, "feature", featurePublicIDPrefix)
	if err != nil {
		return domain.Feature{}, err
	}
	now := timestamp(s.now())
	status, statusAuto := storedFeatureStatus(domain.FeatureStatusAuto)
	params := db.CreateFeatureParams{
		ID:          uuid.NewString(),
		PublicID:    publicID,
		Slug:        slug,
		Title:       title,
		Description: description,
		Status:      status,
		StatusAuto:  statusAuto,
		ProjectID:   project,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	value, err := q.CreateFeature(ctx, params)
	if err != nil {
		return domain.Feature{}, fmt.Errorf("create feature: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Feature{}, err
	}
	return domainFeature(value, projectID), nil
}

func (s *Store) GetFeature(ctx context.Context, id string) (domain.Feature, error) {
	q := db.New(s.db)
	value, err := q.GetFeatureByPublicID(ctx, id)
	if err != nil {
		return domain.Feature{}, mapNotFound(err, "feature", id)
	}
	return domainFeature(value, featureProjectPublicID(ctx, q, value)), nil
}

func (s *Store) GetFeatureBySlug(ctx context.Context, slug string) (domain.Feature, error) {
	q := db.New(s.db)
	value, err := q.GetFeatureBySlug(ctx, slug)
	if err != nil {
		return domain.Feature{}, mapNotFound(err, "feature", slug)
	}
	return domainFeature(value, featureProjectPublicID(ctx, q, value)), nil
}

func featureProjectPublicID(ctx context.Context, q *db.Queries, value db.Feature) string {
	if !value.ProjectID.Valid {
		return ""
	}
	return publicProjectID(ctx, q, value.ProjectID.String)
}

func (s *Store) UpdateFeature(ctx context.Context, feature domain.Feature) (domain.Feature, error) {
	q := db.New(s.db)
	storageID := feature.StorageID
	if storageID == "" {
		value, err := q.GetFeatureByPublicID(ctx, feature.ID)
		if err != nil {
			return domain.Feature{}, mapNotFound(err, "feature", feature.ID)
		}
		storageID = value.ID
	}
	project, err := projectStorageID(ctx, q, feature.ProjectID)
	if err != nil {
		return domain.Feature{}, err
	}
	status, statusAuto := storedFeatureStatus(feature.Status)
	params := db.UpdateFeatureParams{
		Slug:        feature.Slug,
		Title:       feature.Title,
		Description: feature.Description,
		Status:      status,
		StatusAuto:  statusAuto,
		Archived:    boolInt(feature.Archived),
		ProjectID:   project,
		UpdatedAt:   timestamp(s.now()),
		ID:          storageID,
	}
	value, err := q.UpdateFeature(ctx, params)
	return domainFeature(value, feature.ProjectID), mapNotFound(err, "feature", feature.ID)
}

func (s *Store) CreateTask(
	ctx context.Context,
	featureID, title, scope string,
	kind domain.TaskKind,
	assignee string,
) (domain.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	feature, err := q.GetFeatureByPublicID(ctx, featureID)
	if err != nil {
		return domain.Task{}, mapNotFound(err, "feature", featureID)
	}
	publicID, err := nextPublicID(ctx, q, "task", taskPublicIDPrefix)
	if err != nil {
		return domain.Task{}, err
	}
	now := timestamp(s.now())
	params := db.CreateTaskParams{
		ID:        uuid.NewString(),
		PublicID:  publicID,
		FeatureID: feature.ID,
		Title:     title,
		Scope:     scope,
		Kind:      string(kind),
		Status:    string(domain.TaskStatusAuto),
		Assignee:  assignee,
		CreatedAt: now,
		UpdatedAt: now,
	}
	value, err := q.
		CreateTask(ctx, params)
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Task{}, err
	}
	return domainTask(value, feature.PublicID), nil
}

func (s *Store) GetTask(ctx context.Context, id string) (domain.Task, error) {
	q := db.New(s.db)
	value, err := q.GetTaskByPublicID(ctx, id)
	if err != nil {
		return domain.Task{}, mapNotFound(err, "task", id)
	}
	return domainTask(value, publicFeatureID(ctx, q, value.FeatureID)), nil
}

func (s *Store) UpdateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	storageID := task.StorageID
	if storageID == "" {
		value, err := db.New(s.db).GetTaskByPublicID(ctx, task.ID)
		if err != nil {
			return domain.Task{}, mapNotFound(err, "task", task.ID)
		}
		storageID = value.ID
	}
	params := db.UpdateTaskParams{
		Title:     task.Title,
		Scope:     task.Scope,
		Status:    string(task.Status),
		Assignee:  task.Assignee,
		UpdatedAt: timestamp(s.now()),
		ID:        storageID,
	}
	q := db.New(s.db)
	value, err := q.
		UpdateTask(ctx, params)
	if err != nil {
		return domain.Task{}, mapNotFound(err, "task", task.ID)
	}
	return domainTask(value, publicFeatureID(ctx, q, value.FeatureID)), nil
}

func (s *Store) GetImplementationPlan(ctx context.Context, taskID string) (domain.Document, error) {
	q := db.New(s.db)
	task, err := q.GetTaskByPublicID(ctx, taskID)
	if err != nil {
		return domain.Document{}, mapNotFound(err, "task", taskID)
	}
	value, err := q.GetImplementationPlanDocument(ctx, sql.NullString{String: task.ID, Valid: true})
	return domainDocument(value, nil, nil, map[string]string{task.ID: task.PublicID}),
		mapNotFound(err, "implementation plan for task", taskID)
}

func (s *Store) UpsertImplementationPlan(
	ctx context.Context,
	taskID string,
	document domain.Document,
) (domain.Document, error) {
	q := db.New(s.db)
	task, err := q.GetTaskByPublicID(ctx, taskID)
	if err != nil {
		return domain.Document{}, mapNotFound(err, "task", taskID)
	}
	now := timestamp(s.now())
	value, err := q.UpsertImplementationPlanDocument(ctx, db.UpsertImplementationPlanDocumentParams{
		ID:        uuid.NewString(),
		TaskID:    sql.NullString{String: task.ID, Valid: true},
		Kind:      string(document.Kind),
		Title:     document.Title,
		Locator:   nullString(document.Locator),
		Content:   nullString(document.Content),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return domain.Document{}, err
	}
	return domainDocument(value, nil, nil, map[string]string{task.ID: task.PublicID}), nil
}

func (s *Store) DeleteImplementationPlan(ctx context.Context, taskID string) error {
	q := db.New(s.db)
	task, err := q.GetTaskByPublicID(ctx, taskID)
	if err != nil {
		return mapNotFound(err, "task", taskID)
	}
	affected, err := q.DeleteImplementationPlanDocument(ctx, sql.NullString{String: task.ID, Valid: true})
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewError(domain.DomainErrorCodeNotFound, "implementation plan for task %q was not found", taskID)
	}
	return nil
}

func (s *Store) AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Dependency{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	blockerTask, err := q.GetTaskByPublicID(ctx, blocker)
	if err != nil {
		return domain.Dependency{}, mapNotFound(err, "task", blocker)
	}
	blockedTask, err := q.GetTaskByPublicID(ctx, blocked)
	if err != nil {
		return domain.Dependency{}, mapNotFound(err, "task", blocked)
	}
	if blockerTask.FeatureID != blockedTask.FeatureID {
		return domain.Dependency{}, domain.NewError(
			domain.DomainErrorCodeCrossFeatureDependency,
			"dependencies must remain within one feature",
		)
	}
	taskRows, err := q.ListTasksByFeature(ctx, blockerTask.FeatureID)
	if err != nil {
		return domain.Dependency{}, err
	}
	depRows, err := q.ListDependenciesByFeature(ctx, blockerTask.FeatureID)
	if err != nil {
		return domain.Dependency{}, err
	}
	tasks := make([]domain.Task, len(taskRows))
	featureID := publicFeatureID(ctx, q, blockerTask.FeatureID)
	taskIDs := publicTaskIDs(taskRows)
	for i, row := range taskRows {
		tasks[i] = domainTask(row, featureID)
	}
	deps := make([]domain.Dependency, len(depRows))
	for i, row := range depRows {
		deps[i] = domainDependency(row, taskIDs)
		if row.BlockerTaskID == blockerTask.ID && row.BlockedTaskID == blockedTask.ID {
			return domain.Dependency{}, domain.NewError(
				domain.DomainErrorCodeDuplicateDependency,
				"dependency already exists",
			)
		}
	}
	if path := domain.CyclePath(tasks, deps, blockerTask.PublicID, blockedTask.PublicID); len(path) > 0 {
		return domain.Dependency{}, &domain.Error{
			Code:    domain.DomainErrorCodeCycle,
			Message: "dependency would create a cycle",
			Path:    path,
		}
	}
	value, err := q.AddDependency(
		ctx,
		db.AddDependencyParams{
			BlockerTaskID: blockerTask.ID,
			BlockedTaskID: blockedTask.ID,
			CreatedAt:     timestamp(s.now()),
		},
	)
	if err != nil {
		return domain.Dependency{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Dependency{}, err
	}
	return domainDependency(value, taskIDs), nil
}

func (s *Store) RemoveDependency(ctx context.Context, blocker, blocked string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	blockerTask, err := q.GetTaskByPublicID(ctx, blocker)
	if err != nil {
		return mapNotFound(err, "task", blocker)
	}
	blockedTask, err := q.GetTaskByPublicID(ctx, blocked)
	if err != nil {
		return mapNotFound(err, "task", blocked)
	}
	affected, err := q.RemoveDependency(ctx, db.RemoveDependencyParams{
		BlockerTaskID: blockerTask.ID,
		BlockedTaskID: blockedTask.ID,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewError(domain.DomainErrorCodeNotFound, "dependency %q → %q was not found", blocker, blocked)
	}
	return tx.Commit()
}

func (s *Store) DeleteTask(ctx context.Context, id string, cascade bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	task, err := q.GetTaskByPublicID(ctx, id)
	if err != nil {
		return mapNotFound(err, "task", id)
	}
	storageID := task.ID
	if !cascade {
		var references int
		query := `SELECT (SELECT COUNT(*) FROM dependencies WHERE blocker_task_id=? OR blocked_task_id=?) + ` +
			`(SELECT COUNT(*) FROM pull_requests WHERE task_id=?) + (SELECT COUNT(*) FROM documents WHERE task_id=?)`
		if err := tx.QueryRowContext(ctx, query, storageID, storageID, storageID, storageID).
			Scan(&references); err != nil {
			return err
		}
		if references > 0 {
			return domain.NewError(
				domain.DomainErrorCodeReferencesExist,
				"task has dependencies, a pull request, an implementation plan, or documents; pass --cascade",
			)
		}
	} else {
		if err := q.DeleteDependenciesForTask(
			ctx,
			db.DeleteDependenciesForTaskParams{BlockerTaskID: storageID, BlockedTaskID: storageID},
		); err != nil {
			return err
		}
		// Cascading removal does not require a pull request to be present.
		if _, err := q.DeletePullRequest(ctx, storageID); err != nil {
			return err
		}
		if err := q.DeleteDocumentsForTask(ctx, sql.NullString{String: storageID, Valid: true}); err != nil {
			return err
		}
	}
	if err := q.DeleteTask(ctx, storageID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteFeature(ctx context.Context, id string, cascade bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	feature, err := q.GetFeatureByPublicID(ctx, id)
	if err != nil {
		return mapNotFound(err, "feature", id)
	}
	storageID := feature.ID
	tasks, err := q.ListTasksByFeature(ctx, storageID)
	if err != nil {
		return err
	}
	var documentCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents WHERE feature_id=?`, storageID).
		Scan(&documentCount); err != nil {
		return err
	}
	if (len(tasks) > 0 || documentCount > 0) && !cascade {
		return domain.NewError(domain.DomainErrorCodeReferencesExist, "feature has tasks or documents; pass --cascade")
	}
	if cascade {
		if err := q.DeleteDependenciesForFeature(ctx, storageID); err != nil {
			return err
		}
		if err := q.DeletePullRequestsForFeature(ctx, storageID); err != nil {
			return err
		}
		if err := q.DeleteDocumentsForFeature(ctx, sql.NullString{String: storageID, Valid: true}); err != nil {
			return err
		}
		if err := q.DeleteTasksForFeature(ctx, storageID); err != nil {
			return err
		}
	}
	if err := q.DeleteFeature(ctx, storageID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetGitHubRepositoryAuthCache(
	ctx context.Context,
	host, owner, repository string,
) (string, bool, error) {
	row, err := db.New(s.db).GetGitHubRepositoryAuthCache(ctx, db.GetGitHubRepositoryAuthCacheParams{
		Host:       host,
		Owner:      owner,
		Repository: repository,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.AuthMethodID, true, nil
}

func (s *Store) UpsertGitHubRepositoryAuthCache(
	ctx context.Context,
	host, owner, repository, authMethodID string,
) error {
	return db.New(s.db).UpsertGitHubRepositoryAuthCache(ctx, db.UpsertGitHubRepositoryAuthCacheParams{
		Host:            host,
		Owner:           owner,
		Repository:      repository,
		AuthMethodID:    authMethodID,
		LastSucceededAt: timestamp(s.now()),
	})
}

func (s *Store) DeleteGitHubRepositoryAuthCache(ctx context.Context, host, owner, repository string) error {
	_, err := db.New(s.db).DeleteGitHubRepositoryAuthCache(ctx, db.DeleteGitHubRepositoryAuthCacheParams{
		Host:       host,
		Owner:      owner,
		Repository: repository,
	})
	return err
}

func (s *Store) UpsertPullRequest(ctx context.Context, value domain.PullRequest) (domain.PullRequest, error) {
	if value.Host == "" {
		value.Host = "github.com"
	}
	assignees, err := json.Marshal(value.Assignees)
	if err != nil {
		return domain.PullRequest{}, fmt.Errorf("marshal assignees: %w", err)
	}
	q := db.New(s.db)
	task, err := q.GetTaskByPublicID(ctx, value.TaskID)
	if err != nil {
		return domain.PullRequest{}, mapNotFound(err, "task", value.TaskID)
	}
	row, err := q.UpsertPullRequest(ctx, db.UpsertPullRequestParams{
		TaskID:        task.ID,
		Host:          value.Host,
		Owner:         value.Owner,
		Repository:    value.Repository,
		Number:        value.Number,
		Url:           value.URL,
		NodeID:        value.NodeID,
		Author:        value.Author,
		AssigneesJson: string(assignees),
		State:         string(value.State),
		Draft:         boolInt(value.Draft),
		ReviewState:   string(value.ReviewState),
		Mergeability:  string(value.Mergeability),
		GithubUpdatedAt: nullTime(
			value.GitHubUpdatedAt,
		),
		LastSyncedAt: nullTime(value.LastSyncedAt),
		SyncError:    value.SyncError,
		Stale:        boolInt(value.Stale),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.PullRequest{}, domain.NewError(
				domain.DomainErrorCodeDuplicatePullRequest,
				"pull request is already attached to another task",
			)
		}
		return domain.PullRequest{}, err
	}
	return domainPullRequest(row, map[string]string{task.ID: task.PublicID}), nil
}

func (s *Store) GetPullRequest(ctx context.Context, taskID string) (domain.PullRequest, error) {
	q := db.New(s.db)
	task, err := q.GetTaskByPublicID(ctx, taskID)
	if err != nil {
		return domain.PullRequest{}, mapNotFound(err, "task", taskID)
	}
	row, err := q.GetPullRequestByTask(ctx, task.ID)
	return domainPullRequest(
		row,
		map[string]string{task.ID: task.PublicID},
	), mapNotFound(err, "pull request for task", taskID)
}

func (s *Store) DeletePullRequest(ctx context.Context, taskID string) error {
	task, err := db.New(s.db).GetTaskByPublicID(ctx, taskID)
	if err != nil {
		return mapNotFound(err, "task", taskID)
	}
	affected, err := db.New(s.db).DeletePullRequest(ctx, task.ID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewError(domain.DomainErrorCodeNotFound, "pull request for task %q was not found", taskID)
	}
	return nil
}

func (s *Store) GitHubSyncState(ctx context.Context) (domain.GitHubSyncState, error) {
	value, err := db.New(s.db).GetGitHubSyncState(ctx)
	if err != nil {
		return domain.GitHubSyncState{}, err
	}
	return domain.GitHubSyncState{
		LastAttemptAt:   unixTime(value.LastAttemptUnix),
		LastCompletedAt: unixTime(value.LastCompletedUnix),
		Succeeded:       int(value.Succeeded),
		Failed:          int(value.Failed),
		Error:           value.RunError,
	}, nil
}

func (s *Store) AcquireGitHubAutoSync(
	ctx context.Context,
	runID string,
	attemptedAt time.Time,
	dueBeforeUnix int64,
) (bool, error) {
	affected, err := db.New(s.db).AcquireGitHubAutoSync(ctx, db.AcquireGitHubAutoSyncParams{
		RunID:             runID,
		LastAttemptUnix:   sql.NullInt64{Int64: attemptedAt.UTC().Unix(), Valid: true},
		LastAttemptUnix_2: sql.NullInt64{Int64: dueBeforeUnix, Valid: true},
	})
	return affected == 1, err
}

func (s *Store) StartGitHubSync(ctx context.Context, runID string, attemptedAt time.Time) error {
	return db.New(s.db).StartGitHubSync(ctx, db.StartGitHubSyncParams{
		RunID:           runID,
		LastAttemptUnix: sql.NullInt64{Int64: attemptedAt.UTC().Unix(), Valid: true},
	})
}

func (s *Store) CompleteGitHubSync(
	ctx context.Context,
	runID string,
	completedAt time.Time,
	succeeded, failed int,
	runError string,
) (bool, error) {
	affected, err := db.New(s.db).CompleteGitHubSync(ctx, db.CompleteGitHubSyncParams{
		LastCompletedUnix: sql.NullInt64{Int64: completedAt.UTC().Unix(), Valid: true},
		Succeeded:         int64(succeeded),
		Failed:            int64(failed),
		RunError:          runError,
		RunID:             runID,
	})
	return affected == 1, err
}

func unixTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	result := time.Unix(value.Int64, 0).UTC()
	return &result
}

func (s *Store) CreateDocument(
	ctx context.Context,
	parent domain.DocumentParent,
	kind domain.DocumentKind,
	title, locator, content string,
	isImplementationPlan bool,
) (domain.Document, error) {
	q := db.New(s.db)
	projectIDs := map[string]string{}
	featureIDs := map[string]string{}
	taskIDs := map[string]string{}
	stored := domain.DocumentParent{}
	if parent.ProjectID != "" {
		project, err := q.GetProjectByPublicID(ctx, parent.ProjectID)
		if err != nil {
			return domain.Document{}, mapNotFound(err, "project", parent.ProjectID)
		}
		stored.ProjectID = project.ID
		projectIDs[project.ID] = project.PublicID
	}
	if parent.FeatureID != "" {
		feature, err := q.GetFeatureByPublicID(ctx, parent.FeatureID)
		if err != nil {
			return domain.Document{}, mapNotFound(err, "feature", parent.FeatureID)
		}
		stored.FeatureID = feature.ID
		featureIDs[feature.ID] = feature.PublicID
	}
	if parent.TaskID != "" {
		task, err := q.GetTaskByPublicID(ctx, parent.TaskID)
		if err != nil {
			return domain.Document{}, mapNotFound(err, "task", parent.TaskID)
		}
		stored.TaskID = task.ID
		taskIDs[task.ID] = task.PublicID
	}
	params := db.CreateDocumentParams{
		ID:                   uuid.NewString(),
		ProjectID:            nullString(stored.ProjectID),
		FeatureID:            nullString(stored.FeatureID),
		TaskID:               nullString(stored.TaskID),
		Kind:                 string(kind),
		Title:                title,
		Locator:              nullString(locator),
		Content:              nullString(content),
		IsImplementationPlan: boolInt(isImplementationPlan),
		CreatedAt:            timestamp(s.now()),
		UpdatedAt:            timestamp(s.now()),
	}
	row, err := q.CreateDocument(ctx, params)
	if err != nil {
		return domain.Document{}, mapDocumentConstraint(err)
	}
	return domainDocument(row, projectIDs, featureIDs, taskIDs), nil
}

// UpdateDocument writes only the requested fields so that concurrent updates of
// independent fields cannot overwrite each other with stale values.
func (s *Store) UpdateDocument(
	ctx context.Context,
	id string,
	title *string,
	source *domain.Document,
	isImplementationPlan *bool,
) (domain.Document, error) {
	q := db.New(s.db)
	params := db.UpdateDocumentParams{UpdatedAt: timestamp(s.now()), ID: id}
	if title != nil {
		params.SetTitle = 1
		params.Title = *title
	}
	if source != nil {
		params.SetSource = 1
		params.Kind = string(source.Kind)
		params.Locator = nullString(source.Locator)
		params.Content = nullString(source.Content)
	}
	if isImplementationPlan != nil {
		params.SetImplementationPlan = 1
		params.IsImplementationPlan = boolInt(*isImplementationPlan)
	}
	row, err := q.UpdateDocument(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Document{}, mapNotFound(err, "document", id)
		}
		return domain.Document{}, mapDocumentConstraint(err)
	}
	projectIDs, featureIDs, taskIDs := documentPublicIDs(ctx, q, row)
	return domainDocument(row, projectIDs, featureIDs, taskIDs), nil
}

func (s *Store) GetDocument(ctx context.Context, id string) (domain.Document, error) {
	q := db.New(s.db)
	row, err := q.GetDocument(ctx, id)
	if err != nil {
		return domain.Document{}, mapNotFound(err, "document", id)
	}
	projectIDs, featureIDs, taskIDs := documentPublicIDs(ctx, q, row)
	return domainDocument(row, projectIDs, featureIDs, taskIDs), nil
}

// documentPublicIDs resolves the public identifier of whichever parent the
// document row carries, as the single-entry lookup maps domainDocument reads.
func documentPublicIDs(
	ctx context.Context,
	q *db.Queries,
	row db.Document,
) (projectIDs, featureIDs, taskIDs map[string]string) {
	projectIDs = map[string]string{}
	featureIDs = map[string]string{}
	taskIDs = map[string]string{}
	if row.ProjectID.Valid {
		projectIDs[row.ProjectID.String] = publicProjectID(ctx, q, row.ProjectID.String)
	}
	if row.FeatureID.Valid {
		featureIDs[row.FeatureID.String] = publicFeatureID(ctx, q, row.FeatureID.String)
	}
	if row.TaskID.Valid {
		taskIDs[row.TaskID.String] = publicTaskID(ctx, q, row.TaskID.String)
	}
	return projectIDs, featureIDs, taskIDs
}

func (s *Store) DeleteDocument(ctx context.Context, id string) error {
	affected, err := db.New(s.db).DeleteDocument(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewError(domain.DomainErrorCodeNotFound, "document %q was not found", id)
	}
	return nil
}

func (s *Store) Snapshot(ctx context.Context) (domain.Snapshot, error) {
	q := db.New(s.db)
	projects, err := q.ListProjects(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	features, err := q.ListFeatures(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	tasks, err := q.ListTasks(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	deps, err := q.ListDependencies(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	prs, err := q.ListPullRequests(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	planTaskIDs, err := q.ListImplementationPlanTaskIDs(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	docs, err := q.ListDocuments(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	result := domain.Snapshot{
		Projects:     make([]domain.Project, len(projects)),
		Features:     make([]domain.Feature, len(features)),
		Tasks:        make([]domain.Task, len(tasks)),
		Dependencies: make([]domain.Dependency, len(deps)),
		PullRequests: make([]domain.PullRequest, len(prs)),
		Documents:    make([]domain.Document, len(docs)),
	}
	projectIDs := publicProjectIDs(projects)
	featureIDs := publicFeatureIDs(features)
	taskIDs := publicTaskIDs(tasks)
	for i, row := range projects {
		result.Projects[i] = domainProject(row)
	}
	for i, row := range features {
		result.Features[i] = domainFeature(row, projectIDs[row.ProjectID.String])
	}
	for i, row := range tasks {
		result.Tasks[i] = domainTask(row, featureIDs[row.FeatureID])
	}
	planTasks := make(map[string]struct{}, len(planTaskIDs))
	for _, taskID := range planTaskIDs {
		planTasks[taskID.String] = struct{}{}
	}
	for i := range result.Tasks {
		_, result.Tasks[i].HasImplementationPlan = planTasks[result.Tasks[i].StorageID]
	}
	for i, row := range deps {
		result.Dependencies[i] = domainDependency(row, taskIDs)
	}
	for i, row := range prs {
		result.PullRequests[i] = domainPullRequest(row, taskIDs)
	}
	for i, row := range docs {
		result.Documents[i] = domainListedDocument(row, projectIDs, featureIDs, taskIDs)
	}
	return result, nil
}

func mapDocumentConstraint(err error) error {
	if strings.Contains(err.Error(), "documents.task_id") ||
		strings.Contains(err.Error(), "documents_one_plan_per_task_idx") {
		return domain.NewError(
			domain.DomainErrorCodeDuplicateImplementationPlan,
			"task already has an implementation plan",
		)
	}
	return err
}

func (s *Store) Validate(ctx context.Context) []string {
	errorsFound := s.integrityErrors(ctx)
	rows, err := s.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return append(errorsFound, err.Error())
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		errorsFound = append(errorsFound, "foreign key violation")
	}
	if err := rows.Err(); err != nil {
		errorsFound = append(errorsFound, err.Error())
	}
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return append(errorsFound, err.Error())
	}
	if _, err := domain.TopologicalOrder(snapshot.Tasks, snapshot.Dependencies); err != nil {
		errorsFound = append(errorsFound, err.Error())
	}
	return errorsFound
}

func (s *Store) integrityErrors(ctx context.Context) []string {
	// integrity_check reports corruption as result rows, not as an error, and
	// returns the single row "ok" when the database is sound.
	integrity, err := s.db.QueryContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		return []string{err.Error()}
	}
	defer func() { _ = integrity.Close() }()
	errorsFound := []string{}
	for integrity.Next() {
		var line string
		if err := integrity.Scan(&line); err != nil {
			errorsFound = append(errorsFound, err.Error())
			break
		}
		if line != "ok" {
			errorsFound = append(errorsFound, line)
		}
	}
	if err := integrity.Err(); err != nil {
		errorsFound = append(errorsFound, err.Error())
	}
	return errorsFound
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullTime(value *time.Time) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: timestamp(*value), Valid: true}
}
