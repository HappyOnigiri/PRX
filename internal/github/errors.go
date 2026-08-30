package github

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	gh "github.com/google/go-github/v80/github"
)

type ErrorClass string

const (
	ErrorClassAuthUnavailable ErrorClass = "auth_unavailable"
	ErrorClassUnauthorized    ErrorClass = "unauthorized"
	ErrorClassPermission      ErrorClass = "permission"
	ErrorClassNotFound        ErrorClass = "not_found"
	ErrorClassRateLimit       ErrorClass = "rate_limit"
	ErrorClassTransient       ErrorClass = "transient"
	ErrorClassOther           ErrorClass = "other"
)

type ProviderError struct {
	Class      ErrorClass
	StatusCode int
	Operation  string
	Err        error
}

func (e *ProviderError) Error() string {
	if e.Operation == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Operation, e.Err)
}

func (e *ProviderError) Unwrap() error { return e.Err }

func ClassOf(err error) ErrorClass {
	if err == nil {
		return ErrorClassOther
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Class
	}
	return classifyError(err, statusCode(err, nil), nil)
}

func StatusCodeOf(err error) int {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.StatusCode
	}
	return statusCode(err, nil)
}

func wrapProviderError(operation string, err error, response *gh.Response) error {
	if err == nil {
		return nil
	}
	status := statusCode(err, response)
	return &ProviderError{
		Class:      classifyError(err, status, response),
		StatusCode: status,
		Operation:  operation,
		Err:        err,
	}
}

func unavailableError(message string) error {
	return &ProviderError{
		Class:     ErrorClassAuthUnavailable,
		Operation: "resolve GitHub authentication",
		Err:       errors.New(message),
	}
}

func duplicateCredentialError() error {
	return unavailableError("the same credential was already tried for this host")
}

func statusCode(err error, response *gh.Response) int {
	if response != nil && response.Response != nil {
		return response.StatusCode
	}
	var errorResponse *gh.ErrorResponse
	if errors.As(err, &errorResponse) && errorResponse.Response != nil {
		return errorResponse.Response.StatusCode
	}
	var rateLimit *gh.RateLimitError
	if errors.As(err, &rateLimit) && rateLimit.Response != nil {
		return rateLimit.Response.StatusCode
	}
	var abuseLimit *gh.AbuseRateLimitError
	if errors.As(err, &abuseLimit) && abuseLimit.Response != nil {
		return abuseLimit.Response.StatusCode
	}
	return 0
}

func classifyError(err error, status int, response *gh.Response) ErrorClass {
	var rateLimit *gh.RateLimitError
	if errors.As(err, &rateLimit) {
		return ErrorClassRateLimit
	}
	var abuseLimit *gh.AbuseRateLimitError
	if errors.As(err, &abuseLimit) {
		return ErrorClassRateLimit
	}
	if status == http.StatusTooManyRequests {
		return ErrorClassRateLimit
	}
	if status == http.StatusForbidden {
		if rateLimitRemainingZero(err, response) {
			return ErrorClassRateLimit
		}
		return ErrorClassPermission
	}
	switch {
	case status == http.StatusUnauthorized:
		return ErrorClassUnauthorized
	case status == http.StatusNotFound:
		return ErrorClassNotFound
	case status >= http.StatusInternalServerError:
		return ErrorClassTransient
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorClassTransient
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return ErrorClassTransient
	}
	var urlError *url.Error
	if errors.As(err, &urlError) {
		return ErrorClassTransient
	}
	return ErrorClassOther
}

func rateLimitRemainingZero(err error, response *gh.Response) bool {
	if response != nil && response.Response != nil {
		if response.Header.Get("X-RateLimit-Remaining") == "0" {
			return true
		}
	}
	var errorResponse *gh.ErrorResponse
	if errors.As(err, &errorResponse) && errorResponse.Response != nil {
		if errorResponse.Response.Header.Get("X-RateLimit-Remaining") == "0" {
			return true
		}
	}
	return false
}
