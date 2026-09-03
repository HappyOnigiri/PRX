package config

import "github.com/HappyOnigiri/PRX/internal/domain"

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
	return result
}
