package logstore

import "context"

type contextKey struct{}

// NewContext returns a copy of ctx carrying the given LogStore, retrievable via FromContext.
func NewContext(ctx context.Context, store LogStore) context.Context {
	return context.WithValue(ctx, contextKey{}, store)
}

// FromContext returns the LogStore carried by ctx and panics when there is none.
func FromContext(ctx context.Context) LogStore {
	if store, ok := ctx.Value(contextKey{}).(LogStore); ok && store != nil {
		return store
	}
	panic("log store not contained in context")
}
