import { useTranslation } from "react-i18next";
import { formatError } from "../i18n/domain";

type MutationErrorProps = {
  error: Error | null;
  taskTitle?: (id: string) => string | undefined;
};

export function MutationError({ error, taskTitle }: MutationErrorProps) {
  const { t } = useTranslation();
  if (!error) return null;
  return (
    <p className="form-error" role="alert">
      {formatError(error, t, taskTitle)}
    </p>
  );
}
