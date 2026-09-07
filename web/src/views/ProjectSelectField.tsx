import { useTranslation } from "react-i18next";
import type { Project } from "../gen/prx/v1/prx_pb";

interface ProjectSelectFieldProps {
  projects: Project[];
  currentProjectId: string;
}

// The options are the active projects plus the current membership, because an
// uncontrolled select whose defaultValue is missing falls back to the first
// option and would move the feature. Membership is required, so none is empty.
export function ProjectSelectField({
  projects,
  currentProjectId,
}: ProjectSelectFieldProps) {
  const { t } = useTranslation();
  const options = projects.filter(
    (project) => !project.archived || project.id === currentProjectId,
  );
  return (
    <label>
      {t("project.membership")}
      <select name="projectId" defaultValue={currentProjectId}>
        {options.map((project) => (
          <option key={project.id} value={project.id}>
            {project.title}
          </option>
        ))}
      </select>
    </label>
  );
}
