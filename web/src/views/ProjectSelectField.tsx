import { useTranslation } from "react-i18next";
import type { Project } from "../gen/prx/v1/prx_pb";

interface ProjectSelectFieldProps {
  projects: Project[];
  currentProjectId?: string;
}

// The options are the union of the active projects and the current membership.
// The select is uncontrolled, and a defaultValue that is not among its options
// silently falls back to the first one, which would drop a feature out of an
// archived project the next time the form is saved.
export function ProjectSelectField({
  projects,
  currentProjectId = "",
}: ProjectSelectFieldProps) {
  const { t } = useTranslation();
  const options = projects.filter(
    (project) => !project.archived || project.id === currentProjectId,
  );
  return (
    <label>
      {t("project.membership")}
      <select name="projectId" defaultValue={currentProjectId}>
        <option value="">{t("project.noMembership")}</option>
        {options.map((project) => (
          <option key={project.id} value={project.id}>
            {project.title}
          </option>
        ))}
      </select>
    </label>
  );
}
