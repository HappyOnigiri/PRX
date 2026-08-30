import { Save, Trash2 } from "lucide-react";
import { useEffect, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { useDomainMutation } from "../hooks";
import { formatError } from "../i18n/domain";
import { IconButton } from "./IconButton";
import { MarkdownContent } from "./MarkdownPreview";
import { MutationError } from "./MutationError";

export function ImplementationPlanSection({
  taskId,
  hasPlan,
}: {
  taskId: string;
  hasPlan: boolean;
}) {
  const { t } = useTranslation();
  const planKey = `${taskId}:${String(hasPlan)}`;
  const [planContent, setPlanContent] = useState("");
  const [loadedKey, setLoadedKey] = useState<string>();
  const [readError, setReadError] = useState<Error>();
  const [readErrorKey, setReadErrorKey] = useState<string>();
  const save = useDomainMutation(mutations.upsertImplementationPlan);
  const remove = useDomainMutation(mutations.deleteImplementationPlan);
  const content =
    loadedKey === planKey ? planContent : hasPlan ? undefined : "";
  const currentReadError = readErrorKey === planKey ? readError : undefined;
  useEffect(() => {
    let current = true;
    if (!hasPlan) {
      return () => {
        current = false;
      };
    }
    mutations.getImplementationPlan(taskId).then(
      (response) => {
        if (current) {
          setPlanContent(response.implementationPlan?.content ?? "");
          setLoadedKey(planKey);
        }
      },
      (reason: unknown) => {
        if (current) {
          setReadErrorKey(planKey);
          setReadError(
            reason instanceof Error ? reason : new Error(String(reason)),
          );
        }
      },
    );
    return () => {
      current = false;
    };
  }, [hasPlan, planKey, taskId]);

  async function savePlan(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextContent = formValue(new FormData(event.currentTarget), "content");
    try {
      const response = await save.mutateAsync({ taskId, content: nextContent });
      setPlanContent(response.implementationPlan?.content ?? nextContent);
      setLoadedKey(`${taskId}:true`);
    } catch {
      // MutationError renders the structured error beneath the form.
    }
  }

  async function deletePlan() {
    try {
      await remove.mutateAsync(taskId);
      setPlanContent("");
      setLoadedKey(`${taskId}:false`);
    } catch {
      // MutationError renders the structured error beneath the section.
    }
  }

  return (
    <section className="implementation-plan-section">
      <h3>{t("inspector.implementationPlan")}</h3>
      {!hasPlan && (
        <p className="plan-empty">{t("inspector.planNotRegistered")}</p>
      )}
      {currentReadError && (
        <p className="form-error" role="alert">
          {formatError(currentReadError, t)}
        </p>
      )}
      {content === undefined && !currentReadError ? (
        <div className="plan-loading" role="status">
          <div className="spinner" />
          <span>{t("inspector.loadingPlan")}</span>
        </div>
      ) : (
        <PlanEditor
          content={content ?? ""}
          hasPlan={hasPlan}
          onChange={(value) => {
            setPlanContent(value);
            setLoadedKey(planKey);
          }}
          onSubmit={savePlan}
          onDelete={() => {
            void deletePlan();
          }}
          savePending={save.isPending}
          deletePending={remove.isPending}
          saveError={save.error}
          deleteError={remove.error}
        />
      )}
      {content?.trim() && <PlanPreview content={content} />}
    </section>
  );
}

function PlanEditor({
  content,
  hasPlan,
  onChange,
  onSubmit,
  onDelete,
  savePending,
  deletePending,
  saveError,
  deleteError,
}: {
  content: string;
  hasPlan: boolean;
  onChange: (value: string) => void;
  onSubmit: (event: SyntheticEvent<HTMLFormElement>) => void;
  onDelete: () => void;
  savePending: boolean;
  deletePending: boolean;
  saveError: Error | null;
  deleteError: Error | null;
}) {
  const { t } = useTranslation();
  return (
    <form className="stack-form" onSubmit={onSubmit}>
      <label>
        <span className="sr-only">{t("inspector.implementationPlan")}</span>
        <textarea
          name="content"
          value={content}
          onChange={(event) => {
            onChange(event.currentTarget.value);
          }}
          rows={8}
          placeholder={t("inspector.planPlaceholder")}
        />
      </label>
      <div className="plan-actions">
        <IconButton
          icon={Save}
          label={t("inspector.savePlan")}
          variant="primary"
          type="submit"
          disabled={savePending}
        />
        {hasPlan && (
          <IconButton
            icon={Trash2}
            label={t("inspector.deletePlan")}
            variant="danger"
            size="compact"
            iconOnly
            type="button"
            disabled={deletePending}
            onClick={onDelete}
          />
        )}
      </div>
      <MutationError error={saveError} />
      <MutationError error={deleteError} />
    </form>
  );
}

function PlanPreview({ content }: { content: string }) {
  return (
    <article className="implementation-plan-preview markdown-content">
      <MarkdownContent content={content} />
    </article>
  );
}
