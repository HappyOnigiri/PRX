import { Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { Dependency, Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";

interface DependencySectionProps {
  taskId: string;
  tasks: Task[];
  dependencies: Dependency[];
  readOnly?: boolean;
}

export function DependencySection({
  taskId,
  tasks,
  dependencies,
  readOnly = false,
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
          {!readOnly && (
            <IconButton
              icon={Trash2}
              label={t("inspector.removeDependency")}
              variant="danger"
              size="compact"
              iconOnly
              onClick={() => {
                removeDependency.mutate({
                  blocker: dependency.blockerTaskId,
                  blocked: taskId,
                });
              }}
            />
          )}
        </div>
      ))}
      {blockers.length === 0 && readOnly && (
        <p className="read-only-empty">{t("inspector.noDependencies")}</p>
      )}
      <MutationError error={removeDependency.error} />
    </section>
  );
}
