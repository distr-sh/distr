package middleware

import (
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
)

func RateLimitUserIDKey(r *http.Request) (string, error) {
	if auth, err := auth.Authentication.Get(r.Context()); err != nil {
		return "", err
	} else {
		return getTokenIdKey(auth.Token(), auth.CurrentUserID()), nil
	}
}

func RateLimitPathValueKey(name string) func(r *http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		return r.PathValue(name), nil
	}
}

func RateLimitCurrentDeploymentTargetIdKeyFunc(r *http.Request) (string, error) {
	if auth, err := auth.AgentAuthentication.Get(r.Context()); err != nil {
		return "", err
	} else {
		return getTokenIdKey(auth.Token(), auth.CurrentDeploymentTargetID()), nil
	}
}
