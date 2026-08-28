package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/HappyOnigiri/PRX/internal/db"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/google/uuid"
)

func (s *Store) CreateFeature(ctx context.Context, slug, title, description string) (domain.Feature, error) {
	now := timestamp(s.now())
	value, err := db.New(s.db).CreateFeature(ctx, db.CreateFeatureParams{ID: uuid.NewString(), Slug: slug, Title: title, Description: description, Status: "active", CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return domain.Feature{}, fmt.Errorf("create feature: %w", err)
	}
	return domainFeature(value), nil
}

func (s *Store) GetFeature(ctx context.Context, id string) (domain.Feature, error) {
	value, err := db.New(s.db).GetFeature(ctx, id)
	return domainFeature(value), mapNotFound(err, "feature", id)
}

func (s *Store) GetFeatureBySlug(ctx context.Context, slug string) (domain.Feature, error) {
	value, err := db.New(s.db).GetFeatureBySlug(ctx, slug)
	return domainFeature(value), mapNotFound(err, "feature", slug)
}

func (s *Store) UpdateFeature(ctx context.Context, feature domain.Feature) (domain.Feature, error) {
	value, err := db.New(s.db).UpdateFeature(ctx, db.UpdateFeatureParams{Slug: feature.Slug, Title: feature.Title, Description: feature.Description, Status: feature.Status, Archived: boolInt(feature.Archived), UpdatedAt: timestamp(s.now()), ID: feature.ID})
	return domainFeature(value), mapNotFound(err, "feature", feature.ID)
}

func (s *Store) CreateTask(ctx context.Context, featureID, title, scope, kind, assignee string) (domain.Task, error) {
	now := timestamp(s.now())
	value, err := db.New(s.db).CreateTask(ctx, db.CreateTaskParams{ID: uuid.NewString(), FeatureID: featureID, Title: title, Scope: scope, Kind: kind, Status: domain.TaskPlanned, Assignee: assignee, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}
	return domainTask(value), nil
}

func (s *Store) GetTask(ctx context.Context, id string) (domain.Task, error) {
	value, err := db.New(s.db).GetTask(ctx, id)
	return domainTask(value), mapNotFound(err, "task", id)
}

func (s *Store) UpdateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	value, err := db.New(s.db).UpdateTask(ctx, db.UpdateTaskParams{Title: task.Title, Scope: task.Scope, Status: task.Status, Assignee: task.Assignee, UpdatedAt: timestamp(s.now()), ID: task.ID})
	return domainTask(value), mapNotFound(err, "task", task.ID)
}

func (s *Store) AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Dependency{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	blockerTask, err := q.GetTask(ctx, blocker)
	if err != nil {
		return domain.Dependency{}, mapNotFound(err, "task", blocker)
	}
	blockedTask, err := q.GetTask(ctx, blocked)
	if err != nil {
		return domain.Dependency{}, mapNotFound(err, "task", blocked)
	}
	if blockerTask.FeatureID != blockedTask.FeatureID {
		return domain.Dependency{}, domain.NewError("cross_feature_dependency", "dependencies must remain within one feature")
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
	for i, row := range taskRows {
		tasks[i] = domainTask(row)
	}
	deps := make([]domain.Dependency, len(depRows))
	for i, row := range depRows {
		deps[i] = domainDependency(row)
		if row.BlockerTaskID == blocker && row.BlockedTaskID == blocked {
			return domain.Dependency{}, domain.NewError("duplicate_dependency", "dependency already exists")
		}
	}
	if path := domain.CyclePath(tasks, deps, blocker, blocked); len(path) > 0 {
		return domain.Dependency{}, &domain.Error{Code: "cycle", Message: "dependency would create a cycle", Path: path}
	}
	value, err := q.AddDependency(ctx, db.AddDependencyParams{BlockerTaskID: blocker, BlockedTaskID: blocked, CreatedAt: timestamp(s.now())})
	if err != nil {
		return domain.Dependency{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Dependency{}, err
	}
	return domainDependency(value), nil
}

func (s *Store) RemoveDependency(ctx context.Context, blocker, blocked string) error {
	affected, err := db.New(s.db).RemoveDependency(ctx, db.RemoveDependencyParams{BlockerTaskID: blocker, BlockedTaskID: blocked})
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewError("not_found", "dependency %q → %q was not found", blocker, blocked)
	}
	return nil
}

func (s *Store) DeleteTask(ctx context.Context, id string, cascade bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := db.New(tx)
	if _, err := q.GetTask(ctx, id); err != nil {
		return mapNotFound(err, "task", id)
	}
	if !cascade {
		var references int
		if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM dependencies WHERE blocker_task_id=? OR blocked_task_id=?) + (SELECT COUNT(*) FROM pull_requests WHERE task_id=?) + (SELECT COUNT(*) FROM documents WHERE task_id=?)`, id, id, id, id).Scan(&references); err != nil {
			return err
		}
		if references > 0 {
			return domain.NewError("references_exist", "task has dependencies, a pull request, or documents; pass --cascade")
		}
	} else {
		if err := q.DeleteDependenciesForTask(ctx, db.DeleteDependenciesForTaskParams{BlockerTaskID: id, BlockedTaskID: id}); err != nil {
			return err
		}
		// Cascading removal does not require a pull request to be present.
		if _, err := q.DeletePullRequest(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteDocumentsForTask(ctx, sql.NullString{String: id, Valid: true}); err != nil {
			return err
		}
	}
	if err := q.DeleteTask(ctx, id); err != nil {
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
	if _, err := q.GetFeature(ctx, id); err != nil {
		return mapNotFound(err, "feature", id)
	}
	tasks, err := q.ListTasksByFeature(ctx, id)
	if err != nil {
		return err
	}
	if len(tasks) > 0 && !cascade {
		return domain.NewError("references_exist", "feature has tasks; pass --cascade")
	}
	if cascade {
		if err := q.DeleteDependenciesForFeature(ctx, id); err != nil {
			return err
		}
		if err := q.DeletePullRequestsForFeature(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteDocumentsForFeature(ctx, sql.NullString{String: id, Valid: true}); err != nil {
			return err
		}
		if err := q.DeleteTasksForFeature(ctx, id); err != nil {
			return err
		}
	}
	if err := q.DeleteFeature(ctx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UpsertPullRequest(ctx context.Context, value domain.PullRequest) (domain.PullRequest, error) {
	assignees, _ := jsonMarshal(value.Assignees)
	row, err := db.New(s.db).UpsertPullRequest(ctx, db.UpsertPullRequestParams{
		TaskID: value.TaskID, Owner: value.Owner, Repository: value.Repository, Number: value.Number, Url: value.URL,
		NodeID: value.NodeID, Author: value.Author, AssigneesJson: string(assignees), State: value.State,
		Draft: boolInt(value.Draft), ReviewState: value.ReviewState, Mergeability: value.Mergeability,
		GithubUpdatedAt: nullTime(value.GitHubUpdatedAt), LastSyncedAt: nullTime(value.LastSyncedAt), SyncError: value.SyncError, Stale: boolInt(value.Stale),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.PullRequest{}, domain.NewError("duplicate_pull_request", "pull request is already attached to another task")
		}
		return domain.PullRequest{}, err
	}
	return domainPullRequest(row), nil
}

func (s *Store) GetPullRequest(ctx context.Context, taskID string) (domain.PullRequest, error) {
	row, err := db.New(s.db).GetPullRequestByTask(ctx, taskID)
	return domainPullRequest(row), mapNotFound(err, "pull request for task", taskID)
}

func (s *Store) DeletePullRequest(ctx context.Context, taskID string) error {
	affected, err := db.New(s.db).DeletePullRequest(ctx, taskID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewError("not_found", "pull request for task %q was not found", taskID)
	}
	return nil
}

func (s *Store) CreateDocument(ctx context.Context, featureID, taskID, kind, title, value string) (domain.Document, error) {
	row, err := db.New(s.db).CreateDocument(ctx, db.CreateDocumentParams{ID: uuid.NewString(), FeatureID: nullString(featureID), TaskID: nullString(taskID), Kind: kind, Title: title, Value: value, CreatedAt: timestamp(s.now())})
	if err != nil {
		return domain.Document{}, err
	}
	return domainDocument(row), nil
}

func (s *Store) DeleteDocument(ctx context.Context, id string) error {
	affected, err := db.New(s.db).DeleteDocument(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewError("not_found", "document %q was not found", id)
	}
	return nil
}

func (s *Store) Snapshot(ctx context.Context) (domain.Snapshot, error) {
	q := db.New(s.db)
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
	docs, err := q.ListDocuments(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	result := domain.Snapshot{Features: make([]domain.Feature, len(features)), Tasks: make([]domain.Task, len(tasks)), Dependencies: make([]domain.Dependency, len(deps)), PullRequests: make([]domain.PullRequest, len(prs)), Documents: make([]domain.Document, len(docs))}
	for i, row := range features {
		result.Features[i] = domainFeature(row)
	}
	for i, row := range tasks {
		result.Tasks[i] = domainTask(row)
	}
	for i, row := range deps {
		result.Dependencies[i] = domainDependency(row)
	}
	for i, row := range prs {
		result.PullRequests[i] = domainPullRequest(row)
	}
	for i, row := range docs {
		result.Documents[i] = domainDocument(row)
	}
	return result, nil
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
