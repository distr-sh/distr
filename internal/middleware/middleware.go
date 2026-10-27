package middleware

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/authkey"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/logstore"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/prometheus"
	"github.com/distr-sh/distr/internal/requestlog"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-mailx/mailx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func ContextInjectorMiddleware(
	db *pgxpool.Pool,
	dbReadonly *pgxpool.Pool,
	mailer *mailx.Mailer,
	oidcer *oidc.OIDCer,
	prometheusCollector *prometheus.DistrCollector,
	logStore logstore.LogStore,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = internalctx.WithDb(ctx, db)
			if dbReadonly != nil {
				ctx = internalctx.WithReadonlyDB(ctx, dbReadonly)
			}
			ctx = internalctx.WithMailer(ctx, mailer)
			ctx = internalctx.WithPrometheusCollector(ctx, prometheusCollector)
			ctx = internalctx.WithOIDCer(ctx, oidcer)
			if logStore != nil {
				ctx = logstore.NewContext(ctx, logStore)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UseReadonlyDB swaps the request's active db to the read-only database for the wrapped handlers.
// It is a noop when no read-only database is configured (the primary keeps being used).
//
// It must only be applied to routes that perform exclusively read-only queries and that are not part
// of an update-and-refetch loop in the frontend, since the read-only database may lag behind the
// primary. Place it after authentication/authorization middleware so those lookups keep using the
// primary.
func UseReadonlyDB(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if readonlyDB := internalctx.GetReadonlyDB(ctx); readonlyDB != nil {
			ctx = internalctx.WithDb(ctx, readonlyDB)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Chain composes middlewares into one, the first listed running outermost.
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return chi.Chain(middlewares...).Handler
}

// Split sends a request through ifTrue or ifFalse, both of which wrap the same handler.
func Split(
	predicate func(*http.Request) bool,
	ifTrue func(http.Handler) http.Handler,
	ifFalse func(http.Handler) http.Handler,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		yes, no := ifTrue(next), ifFalse(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if predicate(r) {
				yes.ServeHTTP(w, r)
			} else {
				no.ServeHTTP(w, r)
			}
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
		ctx, fields := requestlog.NewContext(r.Context())
		now := time.Now()
		handler.ServeHTTP(ww, r.WithContext(ctx))
		elapsed := time.Since(now)
		logFields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.Status()),
			zap.String("time", elapsed.String()),
		}
		if buildconfig.IsRelease() {
			logFields = append(
				logFields,
				zap.String("userAgent", r.UserAgent()),
				zap.String("ip", middleware.GetClientIP(ctx)),
			)
		}
		internalctx.GetLogger(ctx).Info("handling request", append(logFields, fields.ZapFields()...)...)
	}
	return http.HandlerFunc(fn)
}

// guard passes the request on unless check answers it with a Responder or fails like a [respond.Handler].
func guard(check func(r *http.Request) (respond.Responder, error)) func(http.Handler) http.Handler {
	return respond.Middleware(func(r *http.Request) (*http.Request, respond.Responder, error) {
		if resp, err := check(r); resp != nil || err != nil {
			return nil, resp, err
		}
		return r, nil, nil
	})
}

var insufficientPermissions = respond.Forbidden("insufficient permissions")

func isSuperAdmin(ctx context.Context) bool {
	if auth, err := auth.Authentication.Get(ctx); err == nil {
		return auth.IsSuperAdmin()
	}
	return false
}

func RequireAnyUserRole(userRoles ...types.UserRole) func(handler http.Handler) http.Handler {
	return guard(func(r *http.Request) (respond.Responder, error) {
		if auth, err := auth.Authentication.Get(r.Context()); err != nil {
			return nil, err
		} else if role := auth.CurrentUserRole(); !auth.IsSuperAdmin() &&
			(role == nil || !slices.Contains(userRoles, *role)) {
			return insufficientPermissions, nil
		}
		return nil, nil
	})
}

var (
	RequireReadWriteOrAdmin = RequireAnyUserRole(types.UserRoleReadWrite, types.UserRoleAdmin)
	RequireAdmin            = RequireAnyUserRole(types.UserRoleAdmin)
)

// ForbidSubscriptionTypes blocks the given subscription types. Gating is expressed as a
// denylist of the lower plans instead of an allowlist of the higher ones, so a newly
// introduced plan has access by default.
func ForbidSubscriptionTypes(forbidden ...types.SubscriptionType) func(http.Handler) http.Handler {
	names := make([]string, len(forbidden))
	for i, t := range forbidden {
		names[i] = string(t)
	}
	forbiddenSubscription := respond.Forbidden(
		"this operation can not be performed on an organization with one of the following subscription types: " +
			strings.Join(names, ", "),
	)

	return guard(func(r *http.Request) (respond.Responder, error) {
		if auth, err := auth.Authentication.Get(r.Context()); err != nil {
			return nil, err
		} else if auth.CurrentOrg() == nil {
			return respond.Forbidden("inadequate access token"), nil
		} else if slices.Contains(forbidden, auth.CurrentOrg().SubscriptionType) {
			return forbiddenSubscription, nil
		}
		return nil, nil
	})
}

var ProFeature = ForbidSubscriptionTypes(types.NonProSubscriptionTypes...)

var RequireVendor = guard(func(r *http.Request) (respond.Responder, error) {
	if auth, err := auth.Authentication.Get(r.Context()); err != nil {
		return nil, err
	} else if !auth.IsSuperAdmin() && (auth.CurrentCustomerOrgID() != nil || auth.CurrentPartnerOrgID() != nil) {
		return insufficientPermissions, nil
	}
	return nil, nil
})

var RequireVendorOrPartner = guard(func(r *http.Request) (respond.Responder, error) {
	if auth, err := auth.Authentication.Get(r.Context()); err != nil {
		return nil, err
	} else if !auth.IsSuperAdmin() && auth.CurrentCustomerOrgID() != nil {
		return insufficientPermissions, nil
	}
	return nil, nil
})

var Sentry = sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle

// SetSentryUserFromUserAuth sets the authenticated user's identity on the Sentry scope. It
// must run after auth.Authentication.Middleware so the user is available in the context; if
// there is no authenticated user it panics, since that is a wiring bug.
func SetSentryUserFromUserAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			auth := auth.Authentication.Require(ctx)
			hub.Scope().SetUser(sentry.User{
				ID:    auth.CurrentUserID().String(),
				Email: auth.CurrentUserEmail(),
			})
		}
		h.ServeHTTP(w, r)
	})
}

// SetSentryUserFromControllerAuth sets the authenticated controller's identity on the Sentry scope. It
// must run after auth.ControllerAuthentication.Middleware so the controller is available in the context;
// if there is no authenticated controller it panics, since that is a wiring bug.
func SetSentryUserFromControllerAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			auth := auth.ControllerAuthentication.Require(ctx)
			hub.Scope().SetUser(sentry.User{
				ID: auth.CurrentDeploymentTargetID().String(),
			})
		}
		h.ServeHTTP(w, r)
	})
}

func getTokenIdKey(token any, id uuid.UUID) string {
	prefix := ""
	switch token.(type) {
	case jwt.Token:
		prefix = "jwt"
	case authkey.Token:
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

// RequireTokenScope rejects the request unless the authenticated credential was minted with the
// given token scope. It restricts the password-setting endpoints to their dedicated special tokens:
// regular login tokens, PATs and controller tokens carry the empty scope and are therefore rejected.
func RequireTokenScope(scope authjwt.TokenScope) func(http.Handler) http.Handler {
	return auth.Authentication.ValidatorMiddleware(
		func(value authinfo.AuthInfoWithUserAndOrganization) error {
			if value.TokenScope() != scope {
				return authn.ErrBadAuthentication
			}
			return nil
		},
	)
}

// BlockCrossOrganizationAction rejects an action that would leave the organization the credential is
// confined to, or destroy it, for the credentials described by authinfo.AuthInfo.OrganizationScoped.
// A personal access token is one of them.
var BlockCrossOrganizationAction = guard(func(r *http.Request) (respond.Responder, error) {
	if auth.Authentication.Require(r.Context()).OrganizationScoped() {
		return respond.Forbidden("you are signed in with a credential that belongs to a single organization and can " +
			"therefore only act within that organization"), nil
	}
	return nil, nil
})

// The messages are exported for the endpoints that reject only part of their request body and
// therefore cannot apply the middleware.
const (
	OidcSessionBlockedMessage = "operation not permitted in SSO session. " +
		"Request a password reset to receive a link to your email address that lets you change them."
	AccessTokenBlockedMessage = "this cannot be done with an access token. Sign in to Distr to do it."
)

var RequireNonOidcToken = guard(func(r *http.Request) (respond.Responder, error) {
	if auth.Authentication.Require(r.Context()).IsCustomOIDCSession() {
		return respond.Forbidden(OidcSessionBlockedMessage), nil
	}
	return nil, nil
})

var RequireNonAccessToken = guard(func(r *http.Request) (respond.Responder, error) {
	if auth.Authentication.Require(r.Context()).IsAccessToken() {
		return respond.Forbidden(AccessTokenBlockedMessage), nil
	}
	return nil, nil
})

// RequireEmailVerified rejects requests with 403 when USER_EMAIL_VERIFICATION_REQUIRED is
// enabled and the authenticated user's DB record has no EmailVerifiedAt. It must run after
// auth.Authentication.Middleware so the DB-loaded user is available in the context; if
// there is no authenticated user it panics, since that is a wiring bug.
var RequireEmailVerified = guard(func(r *http.Request) (respond.Responder, error) {
	if !env.UserEmailVerificationRequired() {
		return nil, nil
	} else if user := auth.Authentication.Require(r.Context()).CurrentUser(); user == nil || user.EmailVerifiedAt == nil {
		return respond.Forbidden("email not verified"), nil
	}
	return nil, nil
})

var BlockSuperAdmin = guard(func(r *http.Request) (respond.Responder, error) {
	if isSuperAdmin(r.Context()) {
		return respond.Forbidden("super admins cannot modify resources"), nil
	}
	return nil, nil
})

var BlockSuperAdminUnlessOrganizationExpired = guard(func(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	if !isSuperAdmin(ctx) {
		return nil, nil
	} else if org := auth.Authentication.Require(ctx).CurrentOrg(); org == nil || org.HasActiveSubscription() {
		return respond.Forbidden("super admins cannot delete an active organization"), nil
	}
	return nil, nil
})

func FeatureFlagMiddleware(feature types.Feature) func(handler http.Handler) http.Handler {
	featureNotEnabled := respond.Forbidden(fmt.Sprintf("%v not enabled for organization", feature))

	return guard(func(r *http.Request) (respond.Responder, error) {
		if auth, err := auth.Authentication.Get(r.Context()); err != nil {
			return nil, err
		} else if !auth.CurrentOrg().HasFeature(feature) {
			return featureNotEnabled, nil
		}
		return nil, nil
	})
}

// CustomerFeatureMiddleware rejects a customer user whose customer organization has none of the
// features. Vendor and partner users pass, since they manage the features on behalf of their customers.
func CustomerFeatureMiddleware(features ...types.CustomerOrganizationFeature) func(handler http.Handler) http.Handler {
	names := make([]string, len(features))
	for i, feature := range features {
		names[i] = string(feature)
	}
	featureNotEnabled := respond.Forbidden(strings.Join(names, " or ") + " not enabled for customer organization")

	return guard(func(r *http.Request) (respond.Responder, error) {
		if auth, err := auth.Authentication.Get(r.Context()); err != nil {
			return nil, err
		} else if customerOrg := auth.CurrentCustomerOrg(); customerOrg != nil &&
			!slices.ContainsFunc(features, customerOrg.HasFeature) {
			return featureNotEnabled, nil
		}
		return nil, nil
	})
}

var (
	AlertsCustomerFeatureMiddleware              = CustomerFeatureMiddleware(types.CustomerOrganizationFeatureAlerts)
	UpdateNotificationsCustomerFeatureMiddleware = CustomerFeatureMiddleware(
		types.CustomerOrganizationFeatureUpdateNotifications,
	)
	NotificationHistoryCustomerFeatureMiddleware = CustomerFeatureMiddleware(
		types.CustomerOrganizationFeatureAlerts,
		types.CustomerOrganizationFeatureUpdateNotifications,
	)
	SupportBundlesCustomerFeatureMiddleware = CustomerFeatureMiddleware(types.CustomerOrganizationFeatureSupportBundles)
)

var (
	LicensingFeatureFlagEnabledMiddleware = FeatureFlagMiddleware(types.FeatureLicensing)
	VendorBillingFeatureMiddleware        = FeatureFlagMiddleware(types.FeatureVendorBilling)
	PartnerManagementFeatureMiddleware    = FeatureFlagMiddleware(types.FeaturePartnerManagement)
	CustomDomainsFeatureMiddleware        = FeatureFlagMiddleware(types.FeatureCustomDomains)
	VulnerabilitiesFeatureMiddleware      = FeatureFlagMiddleware(types.FeatureVulnerabilities)
	CustomEmailsFeatureMiddleware         = FeatureFlagMiddleware(types.FeatureCustomEmails)
	CustomOidcProvidersFeatureMiddleware  = FeatureFlagMiddleware(types.FeatureCustomOidcProviders)
	AutoUpdatesFeatureMiddleware          = FeatureFlagMiddleware(types.FeatureAutoUpdates)
)

// RequireCustomDomainsConfigured rejects requests unless the instance itself is set up for custom
// domain self-service. Without a CNAME target there is no proxy obtaining certificates for custom
// domains, so a registered domain would never be served.
var RequireCustomDomainsConfigured = guard(func(r *http.Request) (respond.Responder, error) {
	if !env.CustomDomainsConfigured() {
		return respond.Forbidden("custom domains are not configured on this instance"), nil
	}
	return nil, nil
})

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
