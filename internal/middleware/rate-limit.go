package middleware

import (
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/google/uuid"
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
	if auth, err := auth.ControllerAuthentication.Get(r.Context()); err != nil {
		return "", err
	} else {
		return getTokenIdKey(auth.Token(), auth.CurrentDeploymentTargetID()), nil
	}
}

func RateLimitBasicAuthDeploymentTargetIDKey(r *http.Request) (string, error) {
	targetID, _, _ := r.BasicAuth()
	return deploymentTargetIDKey(targetID), nil
}

func RateLimitQueryDeploymentTargetIDKey(r *http.Request) (string, error) {
	return deploymentTargetIDKey(r.URL.Query().Get("targetId")), nil
}

// deploymentTargetIDKey normalizes id, since uuid.Parse accepts several spellings of the same id that
// would otherwise get a bucket each. Anything else is kept for the handler to reject.
func deploymentTargetIDKey(id string) string {
	if parsed, err := uuid.Parse(id); err == nil {
		return parsed.String()
	}
	return id
}
