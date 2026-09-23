package authz

import "context"

type anonymousLimitKey struct{}

// WithAnonymousLimit attaches the rate limit check for a request without credentials. The authorizer
// runs it only once it has granted the request, because OCI clients send their first request without
// credentials even when they hold some and add them only after the 401 challenge. Counting refused
// requests would spend the budget on authenticated pulls of private artifacts, and once it is gone
// answer them with a 429 instead of the challenge, so the client would never authenticate.
func WithAnonymousLimit(ctx context.Context, exceeded func() bool) context.Context {
	return context.WithValue(ctx, anonymousLimitKey{}, exceeded)
}

// AnonymousLimitExceeded counts a granted anonymous request against its budget and reports whether
// the budget was already exhausted.
func AnonymousLimitExceeded(ctx context.Context) bool {
	if exceeded, ok := ctx.Value(anonymousLimitKey{}).(func() bool); ok {
		return exceeded()
	}
	return false
}
