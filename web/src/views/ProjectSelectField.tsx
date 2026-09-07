import { useTranslation } from "react-i18next";
import type { Project } from "../gen/prx/v1/prx_pb";

interface ProjectSelectFieldProps {
  projects: Project[];
  currentProjectId: string;
}

// 選択肢はアクティブなプロジェクトと現在の所属先。defaultValue が候補にない
// 非制御 select は先頭の選択肢に落ちて feature を移動させてしまうため。所属は
// 必須なので「なし」は用意しない。
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
