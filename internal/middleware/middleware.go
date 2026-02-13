package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authkey"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func ContextInjectorMiddleware(
	db *pgxpool.Pool, mailer mail.Mailer, oidcer *oidc.OIDCer,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = internalctx.WithDb(ctx, db)
			ctx = internalctx.WithMailer(ctx, mailer)
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ctx = internalctx.WithRequestIPAddress(ctx, host)
			ctx = internalctx.WithOIDCer(ctx, oidcer)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func LoggerCtxMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logger.With(zap.String("requestId", middleware.GetReqID(r.Context())))
			ctx := internalctx.WithLogger(r.Context(), logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func LoggingMiddleware(handler http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		now := time.Now()
		handler.ServeHTTP(ww, r)
		elapsed := time.Since(now)
		logger := internalctx.GetLogger(r.Context())
		logger.Info("handling request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.Status()),
			zap.String("time", elapsed.String()))
	}
	return http.HandlerFunc(fn)
}

func isSuperAdmin(ctx context.Context) bool {
	if auth, err := auth.Authentication.Get(ctx); err == nil {
		return auth.IsSuperAdmin()
	}
	return false
}

func RequireAnyUserRole(userRoles ...types.UserRole) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if isSuperAdmin(ctx) {
				handler.ServeHTTP(w, r)
				return
			}
			if auth, err := auth.Authentication.Get(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
			} else if auth.CurrentUserRole() == nil || !slices.Contains(userRoles, *auth.CurrentUserRole()) {
				http.Error(w, "insufficient permissions", http.StatusForbidden)
			} else {
				handler.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}

var (
	RequireReadWriteOrAdmin = RequireAnyUserRole(types.UserRoleReadWrite, types.UserRoleAdmin)
	RequireAdmin            = RequireAnyUserRole(types.UserRoleAdmin)
)

func RequireAnySubscriptionType(types ...types.SubscriptionType) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if auth, err := auth.Authentication.Get(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
			} else if auth.CurrentOrg() == nil {
				http.Error(w, "inadequate access token", http.StatusForbidden)
			} else if !slices.Contains(types, auth.CurrentOrg().SubscriptionType) {
				typesStr := make([]string, 0, len(types))
				for _, t := range types {
					typesStr = append(typesStr, string(t))
				}
				http.Error(w, fmt.Sprintf(
					"this operation can only be performed on an organization with one of the following subscription types: %v",
					strings.Join(typesStr, ", "),
				), http.StatusForbidden)
			} else {
				handler.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}

var ProFeature = RequireAnySubscriptionType(
	types.SubscriptionTypePro,
	types.SubscriptionTypeTrial,
	types.SubscriptionTypeEnterprise,
)

func RequireVendor(handler http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if isSuperAdmin(ctx) {
			handler.ServeHTTP(w, r)
			return
		}
		if auth, err := auth.Authentication.Get(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else if auth.CurrentCustomerOrgID() != nil {
			http.Error(w, "insufficient permissions", http.StatusForbidden)
		} else {
			handler.ServeHTTP(w, r)
		}
	}
	return http.HandlerFunc(fn)
}

var Sentry = sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle

func SentryUser(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			if auth, err := auth.Authentication.Get(ctx); err == nil {
				hub.Scope().SetUser(sentry.User{
					ID:    auth.CurrentUserID().String(),
					Email: auth.CurrentUserEmail(),
				})
			}
		}
		h.ServeHTTP(w, r)
	})
}

func AgentSentryUser(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			if auth, err := auth.AgentAuthentication.Get(ctx); err == nil {
				hub.Scope().SetUser(sentry.User{
					ID: auth.CurrentDeploymentTargetID().String(),
				})
			}
		}
		h.ServeHTTP(w, r)
	})
}

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

func getTokenIdKey(token any, id uuid.UUID) string {
	prefix := ""
	switch token.(type) {
	case jwt.Token:
		prefix = "jwt"
	case authkey.Key:
		prefix = "authkey"
	default:
		panic("unknown token type")
	}
	return fmt.Sprintf("%v-%v", prefix, id)
}

var RequireOrgAndRole = auth.Authentication.ValidatorMiddleware(
	func(value authinfo.AuthInfoWithUserAndOrganization) error {
		if value.IsSuperAdmin() {
			// Super admins still need org context, but don't need a role
			if value.CurrentOrgID() == nil || value.CurrentOrg() == nil {
				return authn.ErrBadAuthentication
			}
			return nil
		}
		if value.CurrentOrgID() == nil || value.CurrentOrg() == nil || value.CurrentUserRole() == nil {
			return authn.ErrBadAuthentication
		}
		return nil
	},
)

func BlockSuperAdmin(handler http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if isSuperAdmin(r.Context()) {
			http.Error(w, "super admins cannot modify resources", http.StatusForbidden)
			return
		}
		handler.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func FeatureFlagMiddleware(feature types.Feature) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if auth, err := auth.Authentication.Get(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
			} else {
				org := auth.CurrentOrg()
				if !org.HasFeature(feature) {
					http.Error(w, fmt.Sprintf("%v not enabled for organization", feature), http.StatusForbidden)
				} else {
					handler.ServeHTTP(w, r)
				}
			}
		}
		return http.HandlerFunc(fn)
	}
}

var LicensingFeatureFlagEnabledMiddleware = FeatureFlagMiddleware(types.FeatureLicensing)

func SetRequestPattern(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if r.Pattern == "" {
			r.Pattern = chi.RouteContext(r.Context()).RoutePattern()
		}
	})
}

func OTEL(provider trace.TracerProvider) func(next http.Handler) http.Handler {
	mw := otelhttp.NewMiddleware(
		"",
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithSpanNameFormatter(
			func(operation string, r *http.Request) string {
				var b strings.Builder
				if operation != "" {
					b.WriteString(operation)
					b.WriteString(" ")
				}
				b.WriteString(r.Method)
				if r.Pattern != "" {
					b.WriteString(" ")
					b.WriteString(r.Pattern)
				}
				return b.String()
			},
		),
	)
	return func(next http.Handler) http.Handler {
		return mw(SetRequestPattern(next))
	}
}
