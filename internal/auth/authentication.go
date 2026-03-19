package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/authn/authkey"
	"github.com/distr-sh/distr/internal/authn/jwt"
	authnSupportBundle "github.com/distr-sh/distr/internal/authn/supportbundle"
	"github.com/distr-sh/distr/internal/authn/token"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// Authentication supports Bearer (classic JWT) and AccessToken (PAT) headers and uses authinfo.DbAuthenticator
// to verify the token against the database, thereby ensuring the user exists in the database.
var Authentication = authn.New(
	authn.Chain4(
		token.NewExtractor(token.WithExtractorFuncs(token.FromHeader("Bearer"))),
		jwt.Authenticator(authjwt.JWTAuth),
		authinfo.UserJWTAuthenticator(),
		authinfo.DbAuthenticator(),
	),
	authn.Chain4(
		token.NewExtractor(token.WithExtractorFuncs(token.FromHeader("AccessToken"))),
		authkey.Authenticator(),
		authinfo.AuthKeyAuthenticator(),
		authinfo.DbAuthenticator(),
	),
)

// AgentAuthentication supports only Bearer JWT tokens
var AgentAuthentication = authn.New(
	authn.Chain3(
		token.NewExtractor(token.WithExtractorFuncs(token.FromHeader("Bearer"))),
		jwt.Authenticator(authjwt.JWTAuth),
		authinfo.AgentJWTAuthenticator(),
		// for agents, db check is done in the agent auth middleware, therefore no DbAuthenticator here
	),
)

// ArtifactsAuthentication supports Basic auth login for OCI clients, where the password should be a PAT.
// The given PAT is verified against the database, to make sure that the user still exists.
var ArtifactsAuthentication = authn.New(
	authn.Chain(
		token.NewExtractor(
			token.WithExtractorFuncs(token.FromBasicAuth()),
			token.WithErrorHeaders(http.Header{"WWW-Authenticate": []string{"Basic realm=\"Distr\""}}),
		),
		authn.Alternative[string, authinfo.AuthInfoWithOrganization](
			// Authenticate UserAccount with PAT
			authn.Chain4(
				authkey.Authenticator(),
				authinfo.AuthKeyAuthenticator(),
				authinfo.DbAuthenticator(),
				authinfo.DropUser(),
			),
			// Authenticate with Agent JWT
			authn.Chain3(
				jwt.Authenticator(authjwt.JWTAuth),
				authinfo.AgentJWTAuthenticator(),
				authinfo.AgentDbAuthenticator(),
			),
		),
	),
)

// SupportBundleAuthentication authenticates collect script requests using
// a query-param token tied to a specific support bundle.
var SupportBundleAuthentication = authn.New[*types.SupportBundle](
	authnSupportBundle.Authenticator(),
)

func handleUnknownError(w http.ResponseWriter, r *http.Request, err error) {
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		internalctx.GetLogger(r.Context()).Error("error authenticating request", zap.Error(err))
		sentry.GetHubFromContext(r.Context()).CaptureException(err)
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func init() {
	Authentication.SetUnknownErrorHandler(handleUnknownError)
	AgentAuthentication.SetUnknownErrorHandler(handleUnknownError)
	ArtifactsAuthentication.SetUnknownErrorHandler(handleUnknownError)
	SupportBundleAuthentication.SetUnknownErrorHandler(handleUnknownError)
}
