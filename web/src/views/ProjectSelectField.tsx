import { useTranslation } from "react-i18next";
import type { Project } from "../gen/prx/v1/prx_pb";

interface ProjectSelectFieldProps {
  projects: Project[];
  currentProjectId: string;
  value: string;
  onChange: (projectId: string) => void;
}

// 選択肢はアクティブなプロジェクトと現在の所属先。候補にない値を選ぶと feature
// を意図せず移動させてしまうため。所属は必須なので「なし」は用意しない。
export function ProjectSelectField({
  projects,
  currentProjectId,
  value,
  onChange,
}: ProjectSelectFieldProps) {
  const { t } = useTranslation();
  const options = projects.filter(
    (project) => !project.archived || project.id === currentProjectId,
  );
  return (
    <label>
      {t("project.membership")}
      <select
        name="projectId"
        value={value}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      >
        {options.map((project) => (
          <option key={project.id} value={project.id}>
            {project.title}
          </option>
        ))}
      </select>
    </label>
  );
}
