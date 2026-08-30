-- name: CreateFeature :one
INSERT INTO features (id, public_id, slug, title, description, status, archived, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: IncrementIDSequence :one
UPDATE id_sequences SET next_value = next_value + 1 WHERE entity = ? RETURNING next_value;

-- name: GetFeature :one
SELECT * FROM features WHERE id = ?;

-- name: GetFeatureByPublicID :one
SELECT * FROM features WHERE public_id = ?;

-- name: GetFeatureBySlug :one
SELECT * FROM features WHERE slug = ?;

-- name: ListFeatures :many
SELECT * FROM features ORDER BY archived, updated_at DESC, slug;

-- name: UpdateFeature :one
UPDATE features SET slug=?, title=?, description=?, status=?, archived=?, updated_at=? WHERE id=? RETURNING *;

-- name: DeleteFeature :exec
DELETE FROM features WHERE id=?;

-- name: CreateTask :one
INSERT INTO tasks (id, public_id, feature_id, title, scope, kind, status, assignee, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id=?;

-- name: GetTaskByPublicID :one
SELECT * FROM tasks WHERE public_id=?;

-- name: ListTasks :many
SELECT * FROM tasks ORDER BY created_at, id;

-- name: ListTasksByFeature :many
SELECT * FROM tasks WHERE feature_id=? ORDER BY created_at, id;

-- name: UpdateTask :one
UPDATE tasks SET title=?, scope=?, status=?, assignee=?, updated_at=? WHERE id=? RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id=?;

-- name: AddDependency :one
INSERT INTO dependencies (blocker_task_id, blocked_task_id, created_at) VALUES (?, ?, ?) RETURNING *;

-- name: RemoveDependency :execrows
DELETE FROM dependencies WHERE blocker_task_id=? AND blocked_task_id=?;

-- name: ListDependencies :many
SELECT * FROM dependencies ORDER BY created_at, blocker_task_id, blocked_task_id;

-- name: ListDependenciesByFeature :many
SELECT d.* FROM dependencies d JOIN tasks t ON t.id=d.blocker_task_id
WHERE t.feature_id=? ORDER BY d.created_at, d.blocker_task_id, d.blocked_task_id;

-- name: DeleteDependenciesForTask :exec
DELETE FROM dependencies WHERE blocker_task_id=? OR blocked_task_id=?;

-- name: DeleteDependenciesForFeature :exec
DELETE FROM dependencies WHERE blocker_task_id IN (SELECT id FROM tasks WHERE feature_id=?);

-- name: UpsertPullRequest :one
INSERT INTO pull_requests (task_id, host, owner, repository, number, url, node_id, author, assignees_json, state, draft, review_state, mergeability, github_updated_at, last_synced_at, sync_error, stale)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(task_id) DO UPDATE SET host=excluded.host, owner=excluded.owner, repository=excluded.repository, number=excluded.number,
url=excluded.url, node_id=excluded.node_id, author=excluded.author, assignees_json=excluded.assignees_json,
state=excluded.state, draft=excluded.draft, review_state=excluded.review_state, mergeability=excluded.mergeability,
github_updated_at=excluded.github_updated_at, last_synced_at=excluded.last_synced_at,
sync_error=excluded.sync_error, stale=excluded.stale RETURNING *;

-- name: GetPullRequestByTask :one
SELECT * FROM pull_requests WHERE task_id=?;

-- name: ListPullRequests :many
SELECT * FROM pull_requests ORDER BY host, owner, repository, number;

-- name: GetGitHubRepositoryAuthCache :one
SELECT * FROM github_repository_auth_cache WHERE host=? AND owner=? AND repository=?;

-- name: UpsertGitHubRepositoryAuthCache :exec
INSERT INTO github_repository_auth_cache (host, owner, repository, auth_method_id, last_succeeded_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(host, owner, repository) DO UPDATE SET auth_method_id=excluded.auth_method_id,
last_succeeded_at=excluded.last_succeeded_at;

-- name: DeleteGitHubRepositoryAuthCache :execrows
DELETE FROM github_repository_auth_cache WHERE host=? AND owner=? AND repository=?;

-- name: DeletePullRequest :execrows
DELETE FROM pull_requests WHERE task_id=?;

-- name: DeletePullRequestsForFeature :exec
DELETE FROM pull_requests WHERE task_id IN (SELECT id FROM tasks WHERE feature_id=?);

-- name: CreateDocument :one
INSERT INTO documents (id, feature_id, task_id, kind, title, value, created_at) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: ListDocuments :many
SELECT * FROM documents ORDER BY created_at, id;

-- name: GetDocument :one
SELECT * FROM documents WHERE id=?;

-- name: DeleteDocument :execrows
DELETE FROM documents WHERE id=?;

-- name: DeleteDocumentsForTask :exec
DELETE FROM documents WHERE task_id=?;

-- name: DeleteDocumentsForFeature :exec
DELETE FROM documents WHERE documents.feature_id=sqlc.arg(feature_id) OR documents.task_id IN (SELECT tasks.id FROM tasks WHERE tasks.feature_id=sqlc.arg(feature_id));

-- name: DeleteTasksForFeature :exec
DELETE FROM tasks WHERE feature_id=?;
