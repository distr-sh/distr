package requestlog

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Fields is filled in by the middlewares that authenticate a request and read by the request logger
// once the request has been handled. The logger runs outermost and cannot see the contexts derived
// further in, so this is shared through a pointer rather than stored as context values.
type Fields struct {
	OrganizationID         *uuid.UUID
	CustomerOrganizationID *uuid.UUID
	UserID                 *uuid.UUID
	DeploymentTargetID     *uuid.UUID
	TokenID                *string
	OrganizationScoped     *bool
}

type contextKey struct{}

func NewContext(ctx context.Context) (context.Context, *Fields) {
	fields := &Fields{}
	return context.WithValue(ctx, contextKey{}, fields), fields
}

// FromContext returns nil when the request is not logged, which Update accepts.
func FromContext(ctx context.Context) *Fields {
	fields, _ := ctx.Value(contextKey{}).(*Fields)
	return fields
}

func (f *Fields) Update(fn func(f *Fields)) {
	if f != nil {
		fn(f)
	}
}

func (f *Fields) ZapFields() []zap.Field {
	return []zap.Field{
		uuidField("organizationId", f.OrganizationID),
		uuidField("customerOrganizationId", f.CustomerOrganizationID),
		uuidField("userId", f.UserID),
		uuidField("deploymentTargetId", f.DeploymentTargetID),
		zap.Stringp("tokenId", f.TokenID),
		zap.Boolp("organizationScoped", f.OrganizationScoped),
	}
}

func uuidField(key string, value *uuid.UUID) zap.Field {
	if value == nil {
		return zap.Reflect(key, nil)
	}
	return zap.Stringer(key, value)
}
