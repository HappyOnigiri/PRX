package app

import (
	"errors"
	"strconv"
	"strings"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func configDomainError(err error) error {
	var configErr *config.Error
	if errors.As(err, &configErr) {
		code := domain.DomainErrorCodeInvalidConfig
		switch configErr.Code {
		case config.ErrorCodeInvalid:
			code = domain.DomainErrorCodeInvalidConfig
		case config.ErrorCodeNotFound:
			code = domain.DomainErrorCodeNotFound
		case config.ErrorCodeReferences:
			code = domain.DomainErrorCodeReferencesExist
		}
		return domain.NewError(code, "%s", configErr.Message)
	}
	return domain.NewError(domain.DomainErrorCodeInvalidConfig, "%s", err)
}

func canonicalPullRequestURL(host config.Host, owner, repository string, number int64) string {
	return strings.TrimRight(host.WebURL, "/") + "/" + owner + "/" + repository + "/pull/" + formatNumber(number)
}

func formatNumber(value int64) string {
	return strconv.FormatInt(value, 10)
}
