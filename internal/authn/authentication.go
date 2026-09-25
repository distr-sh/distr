package authn

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/multierr"
)

type contextKey struct{}

type Authentication[T any] struct {
	authenticators      []RequestAuthenticator[T]
	contextKey          contextKey
	unknownErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
	unauthorizedHandler func(w http.ResponseWriter, r *http.Request)
}

func New[T any](authenticators ...RequestAuthenticator[T]) *Authentication[T] {
	return &Authentication[T]{authenticators: authenticators, contextKey: contextKey{}}
}

func (a *Authentication[T]) SetUnknownErrorHandler(handler func(w http.ResponseWriter, r *http.Request, err error)) {
	a.unknownErrorHandler = handler
}

// SetUnauthorizedHandler overrides the plain text 401, for the registry whose clients expect the
// error format of the distribution spec. The response headers of the authentication error, the
// challenge among them, are already set when the handler is called.
func (a *Authentication[T]) SetUnauthorizedHandler(handler func(w http.ResponseWriter, r *http.Request)) {
	a.unauthorizedHandler = handler
}

func (a *Authentication[T]) NewContext(ctx context.Context, auth T) context.Context {
	return context.WithValue(ctx, a.contextKey, auth)
}

func (a *Authentication[T]) Get(ctx context.Context) (result T, err error) {
	if auth, ok := ctx.Value(a.contextKey).(T); ok {
		return auth, nil
	} else {
		return result, ErrNoAuthentication
	}
}

func (a *Authentication[T]) Require(ctx context.Context) T {
	if auth, err := a.Get(ctx); err != nil {
		panic(err)
	} else {
		return auth
	}
}

func (a *Authentication[T]) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		for _, provider := range a.authenticators {
			var result T
			if result, err = provider.Authenticate(r.Context(), r); err != nil {
				if errors.Is(err, ErrNoAuthentication) {
					continue
				} else {
					break
				}
			} else {
				next.ServeHTTP(w, r.WithContext(a.NewContext(r.Context(), result)))
				return
			}
		}

		a.handleError(w, r, err)
	})
}

func (a *Authentication[T]) ValidatorMiddleware(fn func(value T) error) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := fn(a.Require(r.Context())); err != nil {
				a.handleError(w, r, err)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func (a *Authentication[T]) handleError(w http.ResponseWriter, r *http.Request, err error) {
	for _, err := range multierr.Errors(err) {
		var rh WithResponseHeaders
		if errors.As(err, &rh) {
			for key, value := range rh.ResponseHeaders() {
				for _, v := range value {
					w.Header().Add(key, v)
				}
			}
		}
	}

	statusCode := http.StatusInternalServerError

	if errors.Is(err, ErrBadAuthentication) || errors.Is(err, ErrNoAuthentication) {
		if a.unauthorizedHandler != nil {
			a.unauthorizedHandler(w, r)
			return
		}
		statusCode = http.StatusUnauthorized
	} else if a.unknownErrorHandler != nil {
		a.unknownErrorHandler(w, r, err)
		return
	}

	http.Error(w, http.StatusText(statusCode), statusCode)
}
