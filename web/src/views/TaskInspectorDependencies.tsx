import { Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import type { Dependency, Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { IconButton } from "./IconButton";
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
  const addDependency = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.addDependency(blocker, blocked),
  );
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
        </div>
      ))}
      <form
        className="inline-form"
        onSubmit={(event) => {
          event.preventDefault();
          addDependency.mutate({
            blocker: formValue(new FormData(event.currentTarget), "blocker"),
            blocked: taskId,
          });
        }}
      >
        <select
          name="blocker"
          aria-label={t("inspector.blockerTask")}
          defaultValue=""
        >
          <option value="" disabled>
            {t("inspector.selectBlocker")}
          </option>
          {tasks
            .filter(
              (candidate) =>
                candidate.id !== taskId &&
                !blockers.some(
                  (dependency) => dependency.blockerTaskId === candidate.id,
                ),
            )
            .map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.title}
              </option>
            ))}
        </select>
        <IconButton
          icon={Plus}
          label={t("common.add")}
          variant="primary"
          type="submit"
        />
      </form>
      <MutationError
        error={addDependency.error}
        taskTitle={(id) => tasks.find((task) => task.id === id)?.title}
      />
      <MutationError error={removeDependency.error} />
    </section>
  );
}
