package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/HappyOnigiri/PRX/internal/db"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

const (
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

func (s *Store) CreateFeature(ctx context.Context, slug, title, description string) (domain.Feature, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Feature{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := db.New(tx)
	publicID, err := nextPublicID(ctx, q, "feature", featurePublicIDPrefix)
	if err != nil {
		return domain.Feature{}, err
	}
	now := timestamp(s.now())
	params := db.CreateFeatureParams{
		ID:          uuid.NewString(),
		PublicID:    publicID,
		Slug:        slug,
		Title:       title,
		Description: description,
		Status:      string(domain.FeatureStatusActive),
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
	return domainFeature(value), nil
}

func (s *Store) GetFeature(ctx context.Context, id string) (domain.Feature, error) {
	value, err := db.New(s.db).GetFeatureByPublicID(ctx, id)
	return domainFeature(value), mapNotFound(err, "feature", id)
}

func (s *Store) GetFeatureBySlug(ctx context.Context, slug string) (domain.Feature, error) {
	value, err := db.New(s.db).GetFeatureBySlug(ctx, slug)
	return domainFeature(value), mapNotFound(err, "feature", slug)
}

func (s *Store) UpdateFeature(ctx context.Context, feature domain.Feature) (domain.Feature, error) {
	storageID := feature.StorageID
	if storageID == "" {
		value, err := db.New(s.db).GetFeatureByPublicID(ctx, feature.ID)
		if err != nil {
			return domain.Feature{}, mapNotFound(err, "feature", feature.ID)
		}
		storageID = value.ID
	}
	params := db.UpdateFeatureParams{
		Slug:        feature.Slug,
		Title:       feature.Title,
		Description: feature.Description,
		Status:      string(feature.Status),
		Archived:    boolInt(feature.Archived),
		UpdatedAt:   timestamp(s.now()),
		ID:          storageID,
	}
	value, err := db.New(s.db).
		UpdateFeature(ctx, params)
	return domainFeature(value), mapNotFound(err, "feature", feature.ID)
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
		Status:    string(domain.TaskStatusPlanned),
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
				"task has dependencies, a pull request, or documents; pass --cascade",
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
	if len(tasks) > 0 && !cascade {
		return domain.NewError(domain.DomainErrorCodeReferencesExist, "feature has tasks; pass --cascade")
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

func (s *Store) UpsertPullRequest(ctx context.Context, value domain.PullRequest) (domain.PullRequest, error) {
	assignees, _ := jsonMarshal(value.Assignees)
	q := db.New(s.db)
	task, err := q.GetTaskByPublicID(ctx, value.TaskID)
	if err != nil {
		return domain.PullRequest{}, mapNotFound(err, "task", value.TaskID)
	}
	row, err := q.UpsertPullRequest(ctx, db.UpsertPullRequestParams{
		TaskID:        task.ID,
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

func (s *Store) CreateDocument(
	ctx context.Context,
	featureID, taskID string,
	kind domain.DocumentKind,
	title, value string,
) (domain.Document, error) {
	q := db.New(s.db)
	featureIDs := map[string]string{}
	taskIDs := map[string]string{}
	if featureID != "" {
		feature, err := q.GetFeatureByPublicID(ctx, featureID)
		if err != nil {
			return domain.Document{}, mapNotFound(err, "feature", featureID)
		}
		featureID = feature.ID
		featureIDs[feature.ID] = feature.PublicID
	}
	if taskID != "" {
		task, err := q.GetTaskByPublicID(ctx, taskID)
		if err != nil {
			return domain.Document{}, mapNotFound(err, "task", taskID)
		}
		taskID = task.ID
		taskIDs[task.ID] = task.PublicID
	}
	params := db.CreateDocumentParams{
		ID:        uuid.NewString(),
		FeatureID: nullString(featureID),
		TaskID:    nullString(taskID),
		Kind:      string(kind),
		Title:     title,
		Value:     value,
		CreatedAt: timestamp(s.now()),
	}
	row, err := db.New(s.db).
		CreateDocument(ctx, params)
	if err != nil {
		return domain.Document{}, err
	}
	return domainDocument(row, featureIDs, taskIDs), nil
}

func (s *Store) GetDocument(ctx context.Context, id string) (domain.Document, error) {
	q := db.New(s.db)
	row, err := q.GetDocument(ctx, id)
	if err != nil {
		return domain.Document{}, mapNotFound(err, "document", id)
	}
	featureIDs := map[string]string{}
	taskIDs := map[string]string{}
	if row.FeatureID.Valid {
		featureIDs[row.FeatureID.String] = publicFeatureID(ctx, q, row.FeatureID.String)
	}
	if row.TaskID.Valid {
		taskIDs[row.TaskID.String] = publicTaskID(ctx, q, row.TaskID.String)
	}
	return domainDocument(row, featureIDs, taskIDs), nil
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
	result := domain.Snapshot{
		Features:     make([]domain.Feature, len(features)),
		Tasks:        make([]domain.Task, len(tasks)),
		Dependencies: make([]domain.Dependency, len(deps)),
		PullRequests: make([]domain.PullRequest, len(prs)),
		Documents:    make([]domain.Document, len(docs)),
	}
	featureIDs := publicFeatureIDs(features)
	taskIDs := publicTaskIDs(tasks)
	for i, row := range features {
		result.Features[i] = domainFeature(row)
	}
	for i, row := range tasks {
		result.Tasks[i] = domainTask(row, featureIDs[row.FeatureID])
	}
	for i, row := range deps {
		result.Dependencies[i] = domainDependency(row, taskIDs)
	}
	for i, row := range prs {
		result.PullRequests[i] = domainPullRequest(row, taskIDs)
	}
	for i, row := range docs {
		result.Documents[i] = domainDocument(row, featureIDs, taskIDs)
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
