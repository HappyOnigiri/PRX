import { useTranslation } from "react-i18next";
import { TaskKind } from "../gen/prx/v1/prx_pb";
import { taskKindLabel } from "../i18n/domain";

export function CreateTaskFields() {
  const { t } = useTranslation();
  return (
    <>
      <label>
        {t("common.title")}
        <input
          name="title"
          required
          placeholder={t("taskCreate.titlePlaceholder")}
        />
      </label>
      <label>
        {t("common.scope")}
        <textarea name="scope" placeholder={t("taskCreate.scopePlaceholder")} />
      </label>
      <div className="form-row">
        <label>
          {t("taskCreate.kind")}
          <select name="kind">
            <option value={TaskKind.PULL_REQUEST}>
              {taskKindLabel(TaskKind.PULL_REQUEST, t)}
            </option>
            <option value={TaskKind.MANUAL}>
              {taskKindLabel(TaskKind.MANUAL, t)}
            </option>
          </select>
        </label>
        <label>
          {t("common.assignee")}
          <input
            name="assignee"
            placeholder={t("taskCreate.assigneePlaceholder")}
          />
        </label>
      </div>
    </>
  );
}
