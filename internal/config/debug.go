package config

import (
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

// DebugInput collects the configuration facts a diagnostic report presents. It
// loads the file exactly as every other caller does, so the report describes the
// configuration PRX would actually use, and it copies only the public view so no
// credential material can reach the report.
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
	// The templates stay out of the public view because their bodies are long
	// and user-authored. What a reader needs when a copied prompt looks wrong is
	// whether the wording was edited at all, which these facts answer without
	// putting the text into the report.
	defaults := prompt.DefaultTemplates()
	result.Prompts = domain.DebugConfigPrompts{
		Design:         debugPrompt(value.Prompts.Design, defaults.Design),
		Implementation: debugPrompt(value.Prompts.Implementation, defaults.Implementation),
	}
	return result
}

func debugPrompt(stored, builtIn string) domain.DebugConfigPrompt {
	return domain.DebugConfigPrompt{Customized: stored != builtIn, Bytes: len(stored)}
}
