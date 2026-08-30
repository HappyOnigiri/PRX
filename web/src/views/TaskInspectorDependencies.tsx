import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { Dependency, Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { MutationError } from "./MutationError";

interface DependencySectionProps {
  taskId: string;
  tasks: Task[];
  dependencies: Dependency[];
}

export function DependencySection({
  taskId,
  tasks,
  dependencies,
}: DependencySectionProps) {
  const { t } = useTranslation();
  const removeDependency = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.removeDependency(blocker, blocked),
  );
  const blockers = dependencies.filter((dependency) => {
    return dependency.blockedTaskId === taskId;
  });

  return (
    <section>
      <h3>{t("inspector.blockedBy")}</h3>
      {blockers.map((dependency) => (
        <div className="dependency-chip" key={dependency.blockerTaskId}>
          <span>
            {tasks.find((task) => task.id === dependency.blockerTaskId)?.title}
          </span>
          <button
            aria-label={t("inspector.removeDependency")}
            onClick={() => {
              removeDependency.mutate({
                blocker: dependency.blockerTaskId,
                blocked: taskId,
              });
            }}
          >
            ×
          </button>
        </div>
      ))}
      <MutationError error={removeDependency.error} />
    </section>
  );
}
