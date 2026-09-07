package app

import (
	"context"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// 以下のガードが、アーカイブ済みの project や feature が何を拒むかを決める唯一の場所であり、
// CLI・RPC ハンドラ・WebUI は同じルールを引き継ぐ。
// 障壁と例外、その限界は docs/design/archive.md に記録している。

// archivedReadOnly は拒否された書き込みが必ず返す唯一のエラー。呼び出し側は
// どのコンテナがアーカイブ済みかを知らずに、1 つのコードだけで分岐できる。
func archivedReadOnly() error {
	return domain.NewError(
		domain.DomainErrorCodeArchivedReadOnly,
		"archived projects and features are read-only; unarchive before writing",
	)
}

// archivedFeatureFlagOnly と archivedProjectFlagOnly は、リクエストが archived フラグ
// だけを変えるかを判定する。フラグを消して残りをゼロ値と比較するため、
// フィールドが増えても自動的に対象になる。
func archivedFeatureFlagOnly(update domain.FeatureUpdate) bool {
	if update.Archived == nil {
		return false
	}
	update.Archived = nil
	return update == domain.FeatureUpdate{}
}

func archivedProjectFlagOnly(update domain.ProjectUpdate) bool {
	if update.Archived == nil {
		return false
	}
	update.Archived = nil
	return update == domain.ProjectUpdate{}
}

// featureReadOnly は、snapshot 外で読んだ feature について、Snapshot が
// Feature.ReadOnly として公開する値を導出する。
func (s *Service) featureReadOnly(ctx context.Context, feature domain.Feature) (bool, error) {
	if feature.Archived {
		return true, nil
	}
	project, err := s.repository.GetProject(ctx, feature.ProjectID)
	if err != nil {
		return false, err
	}
	return project.Archived, nil
}

// withReadOnly は ReadOnly を導出した feature を返す。snapshot 以外で application 層を
// 出る feature も、保存されたゼロ値ではなく Snapshot と同じ値を持つ。
func (s *Service) withReadOnly(ctx context.Context, feature domain.Feature) (domain.Feature, error) {
	readOnly, err := s.featureReadOnly(ctx, feature)
	if err != nil {
		return domain.Feature{}, err
	}
	feature.ReadOnly = readOnly
	return feature, nil
}

// guardFeature は、アーカイブ済みの feature 内、または project がアーカイブ済みの
// feature 内への書き込みを拒む。
func (s *Service) guardFeature(ctx context.Context, feature domain.Feature) error {
	readOnly, err := s.featureReadOnly(ctx, feature)
	if err != nil {
		return err
	}
	if readOnly {
		return archivedReadOnly()
	}
	return nil
}

// guardTask は、所属する feature が読み取り専用のとき、task とその依存・
// pull request・document への書き込みを拒む。
func (s *Service) guardTask(ctx context.Context, task domain.Task) error {
	feature, err := s.ResolveFeature(ctx, task.FeatureID)
	if err != nil {
		return err
	}
	return s.guardFeature(ctx, feature)
}

// guardTaskID は、呼び出し側が公開 ID で指定した task への書き込みをガードし、
// 該当がなければ task が存在しないことを報告する。
func (s *Service) guardTaskID(ctx context.Context, taskID string) error {
	task, err := s.repository.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	return s.guardTask(ctx, task)
}

func (s *Service) guardProject(project domain.Project) error {
	if project.Archived {
		return archivedReadOnly()
	}
	return nil
}

// guardDocument は、親が読み取り専用の document への書き込みを拒む。
// 3 種類の親のいずれを持つ場合も対象になる。
func (s *Service) guardDocument(ctx context.Context, document domain.Document) error {
	switch {
	case document.ProjectID != "":
		project, err := s.repository.GetProject(ctx, document.ProjectID)
		if err != nil {
			return err
		}
		return s.guardProject(project)
	case document.FeatureID != "":
		feature, err := s.ResolveFeature(ctx, document.FeatureID)
		if err != nil {
			return err
		}
		return s.guardFeature(ctx, feature)
	case document.TaskID != "":
		return s.guardTaskID(ctx, document.TaskID)
	}
	return nil
}

// resolveProjectAssignment は、要求された所属を保存用の公開 project ID に変換する。
// 所属は必須なので空の要求は拒否し、アーカイブ済みの project は feature を受け入れない。
func (s *Service) resolveProjectAssignment(ctx context.Context, requested string) (string, error) {
	if requested == "" {
		return "", domain.NewError(domain.DomainErrorCodeInvalidParent, "feature project is required")
	}
	project, err := s.ResolveProject(ctx, requested)
	if err != nil {
		return "", err
	}
	if err := s.guardProject(project); err != nil {
		return "", err
	}
	return project.ID, nil
}
