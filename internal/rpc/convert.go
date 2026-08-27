package rpc

import (
	prmapv1 "github.com/HappyOnigiri/PRX/gen/prmap/v1"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func protoFeature(v domain.Feature) *prmapv1.Feature {
	return &prmapv1.Feature{Id: v.ID, Slug: v.Slug, Title: v.Title, Description: v.Description, Status: v.Status, Archived: v.Archived, CreatedAt: v.CreatedAt.Format(timeFormat), UpdatedAt: v.UpdatedAt.Format(timeFormat), TaskCount: int32(v.TaskCount), ReadyCount: int32(v.ReadyCount), ReviewWaitingCount: int32(v.ReviewWaitingCount), ConflictCount: int32(v.ConflictCount), MergedCount: int32(v.MergedCount)}
}
func protoTask(v domain.Task) *prmapv1.Task {
	return &prmapv1.Task{Id: v.ID, FeatureId: v.FeatureID, Title: v.Title, Scope: v.Scope, Kind: v.Kind, Status: v.Status, Assignee: v.Assignee, CreatedAt: v.CreatedAt.Format(timeFormat), UpdatedAt: v.UpdatedAt.Format(timeFormat), Ready: v.Ready, DisplayState: v.DisplayState, BlockedReason: v.BlockedReason}
}
func protoDependency(v domain.Dependency) *prmapv1.Dependency {
	return &prmapv1.Dependency{BlockerTaskId: v.BlockerTaskID, BlockedTaskId: v.BlockedTaskID, CreatedAt: v.CreatedAt.Format(timeFormat)}
}
func protoPullRequest(v domain.PullRequest) *prmapv1.PullRequest {
	result := &prmapv1.PullRequest{TaskId: v.TaskID, Owner: v.Owner, Repository: v.Repository, Number: v.Number, Url: v.URL, NodeId: v.NodeID, Author: v.Author, Assignees: v.Assignees, State: v.State, Draft: v.Draft, ReviewState: v.ReviewState, Mergeability: v.Mergeability, SyncError: v.SyncError, Stale: v.Stale, DisplayState: v.DisplayState}
	if v.GitHubUpdatedAt != nil {
		result.GithubUpdatedAt = v.GitHubUpdatedAt.Format(timeFormat)
	}
	if v.LastSyncedAt != nil {
		result.LastSyncedAt = v.LastSyncedAt.Format(timeFormat)
	}
	return result
}
func protoDocument(v domain.Document) *prmapv1.Document {
	return &prmapv1.Document{Id: v.ID, FeatureId: v.FeatureID, TaskId: v.TaskID, Kind: v.Kind, Title: v.Title, Value: v.Value, CreatedAt: v.CreatedAt.Format(timeFormat)}
}

func protoSnapshot(v domain.Snapshot) *prmapv1.Snapshot {
	result := &prmapv1.Snapshot{}
	for _, item := range v.Features {
		result.Features = append(result.Features, protoFeature(item))
	}
	for _, item := range v.Tasks {
		result.Tasks = append(result.Tasks, protoTask(item))
	}
	for _, item := range v.Dependencies {
		result.Dependencies = append(result.Dependencies, protoDependency(item))
	}
	for _, item := range v.PullRequests {
		result.PullRequests = append(result.PullRequests, protoPullRequest(item))
	}
	for _, item := range v.Documents {
		result.Documents = append(result.Documents, protoDocument(item))
	}
	for _, item := range v.ReadyTasks {
		result.ReadyTasks = append(result.ReadyTasks, protoTask(item))
	}
	for _, item := range v.ReviewWaitingTasks {
		result.ReviewWaitingTasks = append(result.ReviewWaitingTasks, protoTask(item))
	}
	for _, item := range v.ConflictTasks {
		result.ConflictTasks = append(result.ConflictTasks, protoTask(item))
	}
	for _, item := range v.StaleTasks {
		result.StaleTasks = append(result.StaleTasks, protoTask(item))
	}
	return result
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"
