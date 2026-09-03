import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  DependencySchema,
  DocumentKind,
  DocumentSchema,
  FeatureSchema,
  FeatureStatus,
  ProjectSchema,
  PullRequestDisplayState,
  PullRequestSchema,
  SnapshotSchema,
  TaskDisplayState,
  TaskKind,
  TaskSchema,
  TaskStatus,
  type Dependency,
  type Document,
  type Feature,
  type Project,
  type PullRequest,
  type Snapshot,
  type Task,
} from "../src/gen/prx/v1/prx_pb";

const featureDefaults = {
  id: "feature-1",
  slug: "payments",
  title: "Payments rollout",
  status: FeatureStatus.AUTO,
  displayStatus: FeatureStatus.ACTIVE,
  taskCount: 1,
  readyCount: 1,
  finishedCount: 0,
} satisfies MessageInitShape<typeof FeatureSchema>;

const projectDefaults = {
  id: "project-1",
  slug: "delivery",
  title: "Delivery platform",
} satisfies MessageInitShape<typeof ProjectSchema>;

const taskDefaults = {
  id: "task-1",
  featureId: "feature-1",
  title: "Build API",
  kind: TaskKind.PULL_REQUEST,
  status: TaskStatus.AUTO,
  assignee: "Mika",
  ready: true,
  displayState: TaskDisplayState.NOT_STARTED,
} satisfies MessageInitShape<typeof TaskSchema>;

const dependencyDefaults = {
  blockerTaskId: "task-1",
  blockedTaskId: "task-2",
} satisfies MessageInitShape<typeof DependencySchema>;

const pullRequestDefaults = {
  taskId: "task-1",
  owner: "acme",
  repository: "prx",
  number: 42n,
  url: "https://github.com/acme/prx/pull/42",
  displayState: PullRequestDisplayState.OPEN,
} satisfies MessageInitShape<typeof PullRequestSchema>;

const documentDefaults = {
  id: "document-1",
  taskId: "task-1",
  kind: DocumentKind.URL,
  title: "Runbook",
  locator: "https://example.com/runbook",
} satisfies MessageInitShape<typeof DocumentSchema>;

// The server reports every archived feature as read-only, so a fixture that
// set only archived would describe a snapshot the server never sends. An
// explicit readOnly still wins, which is how a feature that is read-only only
// because its project is archived is expressed.
export function makeFeature(
  overrides: MessageInitShape<typeof FeatureSchema> = {},
): Feature {
  const readOnly = overrides.readOnly ?? overrides.archived ?? false;
  return create(FeatureSchema, { ...featureDefaults, ...overrides, readOnly });
}

export function makeProject(
  overrides: MessageInitShape<typeof ProjectSchema> = {},
): Project {
  return create(ProjectSchema, { ...projectDefaults, ...overrides });
}

export function makeTask(
  overrides: MessageInitShape<typeof TaskSchema> = {},
): Task {
  return create(TaskSchema, { ...taskDefaults, ...overrides });
}

export function makeDependency(
  overrides: MessageInitShape<typeof DependencySchema> = {},
): Dependency {
  return create(DependencySchema, { ...dependencyDefaults, ...overrides });
}

export function makePullRequest(
  overrides: MessageInitShape<typeof PullRequestSchema> = {},
): PullRequest {
  return create(PullRequestSchema, { ...pullRequestDefaults, ...overrides });
}

export function makeDocument(
  overrides: MessageInitShape<typeof DocumentSchema> = {},
): Document {
  return create(DocumentSchema, { ...documentDefaults, ...overrides });
}

export function makeSnapshot(
  overrides: MessageInitShape<typeof SnapshotSchema> = {},
): Snapshot {
  return create(SnapshotSchema, {
    features: [makeFeature()],
    tasks: [makeTask()],
    ...overrides,
  });
}
