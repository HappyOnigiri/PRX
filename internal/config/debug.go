package config

import (
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

// DebugInput は診断レポートが提示する設定情報を集める。他の呼び出し元と
// まったく同じ手順でファイルを読み込み、公開ビューだけを複製するため、
// 資格情報がレポートに漏れることはない。
func (s *Store) DebugInput() domain.DebugConfigInput {
	value, warnings, err := s.LoadWithWarnings()
	result := domain.DebugConfigInput{Warnings: warnings}
	if err != nil {
		result.LoadError = err.Error()
		return result
	}
	public := value.Public()
	result.Version = public.Version
	result.AutoSyncIntervalSeconds = public.GitHub.AutoSyncIntervalSeconds
	for _, host := range public.GitHub.Hosts {
		result.Hosts = append(result.Hosts, domain.DebugConfigHost{
			Host:       host.Host,
			APIURL:     host.APIURL,
			GraphQLURL: host.GraphQLURL,
		})
	}
	for _, method := range public.GitHub.AuthMethods {
		result.AuthMethods = append(result.AuthMethods, domain.DebugConfigAuthMethod{
			ID:               method.ID,
			Host:             method.Host,
			Type:             string(method.Type),
			SecretConfigured: method.SecretConfigured,
		})
	}
	// テンプレート本体は長くユーザーが書き換えるため公開ビューには含めない。
	// コピーしたプロンプトが変に見えるとき、読み手は文言が編集されたか
	// どうかだけ分かればよく、ここの情報がそれに答える。
	defaults := prompt.DefaultTemplates()
	result.Prompts = domain.DebugConfigPrompts{
		Design:         debugPrompt(value.Prompts.Design, defaults.Design),
		Implementation: debugPrompt(value.Prompts.Implementation, defaults.Implementation),
		Batch:          debugPrompt(value.Prompts.Batch, defaults.Batch),
	}
	return result
}

func debugPrompt(stored, builtIn string) domain.DebugConfigPrompt {
	return domain.DebugConfigPrompt{Customized: stored != builtIn, Bytes: len(stored)}
}
