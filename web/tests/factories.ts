import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  FeatureSchema,
  FeatureStatus,
  SnapshotSchema,
  TaskDisplayState,
  TaskKind,
  TaskSchema,
  TaskStatus,
  type Feature,
  type Snapshot,
  type Task,
} from "../src/gen/prx/v1/prx_pb";

const featureDefaults = {
  id: "feature-1",
  slug: "payments",
  title: "Payments rollout",
  status: FeatureStatus.ACTIVE,
  taskCount: 1,
  readyCount: 1,
} satisfies MessageInitShape<typeof FeatureSchema>;

const taskDefaults = {
  id: "task-1",
  featureId: "feature-1",
  title: "Build API",
  kind: TaskKind.PULL_REQUEST,
  status: TaskStatus.PLANNED,
  assignee: "Mika",
  ready: true,
  displayState: TaskDisplayState.UNLINKED,
} satisfies MessageInitShape<typeof TaskSchema>;

export function makeFeature(
  overrides: MessageInitShape<typeof FeatureSchema> = {},
): Feature {
  return create(FeatureSchema, { ...featureDefaults, ...overrides });
}

export function makeTask(
  overrides: MessageInitShape<typeof TaskSchema> = {},
): Task {
  return create(TaskSchema, { ...taskDefaults, ...overrides });
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
