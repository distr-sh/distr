package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func UserAccountsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Users"))
	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", getUserAccountsHandler).
			With(option.Description("List all user accounts")).
			With(option.Response(http.StatusOK, []api.UserAccountResponse{}))
		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", createUserAccountHandler).
			With(option.Description("Create a new user account")).
			With(option.Request(api.CreateUserAccountRequest{})).
			With(option.Response(http.StatusOK, api.CreateUserAccountResponse{}))
		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Route("/{userId}", func(r chiopenapi.Router) {
			type UserAccountRequest struct {
				UserId string `json:"-" path:"userId"`
			}

			r.Use(userAccountMiddleware)
			r.With(middleware.ProFeature).
				Patch("/", patchUserAccountHandler()).
				With(option.Description("Partially update a user account")).
				With(option.Request(struct {
					UserAccountRequest
					api.PatchUserAccountRequest
				}{})).
				With(option.Response(http.StatusOK, api.UserAccountResponse{}))
			r.Delete("/", deleteUserAccountHandler).
				With(option.Description("Delete a user account")).
				With(option.Request(UserAccountRequest{}))
			r.Patch("/image", patchImageUserAccount).
				With(option.Description("Update user account image")).
				With(option.Request(struct {
					UserAccountRequest
					api.PatchImageRequest
				}{})).
				With(option.Response(http.StatusOK, api.UserAccountResponse{}))
			r.With(inviteUserRateLimiter).
				Post("/invite", resendUserInviteHandler()).
				With(option.Description("Resend user invite")).
				With(option.Request(UserAccountRequest{})).
				With(option.Response(http.StatusOK, api.CreateUserAccountResponse{}))
		})
	})
	r.Get("/status", getUserAccountStatusHandler).With(option.Hidden(true))
}

func getUserAccountsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	var userAccounts []types.UserAccountWithUserRole
	var err error

	if customerOrgID := auth.CurrentCustomerOrgID(); customerOrgID != nil {
		userAccounts, err = db.GetUserAccountsByCustomerOrgID(ctx, *customerOrgID)
	} else {
		userAccounts, err = db.GetUserAccountsByOrgID(ctx, *auth.CurrentOrgID())
	}

	if err != nil {
		log.Error("failed to get user accounts", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.List(userAccounts, mapping.UserAccountToAPI))
	}
}

func getUserAccountStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	userAccount := auth.CurrentUser()
	RespondJSON(w, map[string]any{
		"active": userAccount.PasswordHash != nil,
	})
}

func createUserAccountHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	body, err := JsonBody[api.CreateUserAccountRequest](w, r)
	if err != nil {
		return
	}

	if customerOrgID := auth.CurrentCustomerOrgID(); customerOrgID != nil {
		if *auth.CurrentUserRole() != types.UserRoleAdmin {
			http.Error(w, "must be admin to create users", http.StatusForbidden)
			return
		}

		body.CustomerOrganizationID = customerOrgID
	} else if *auth.CurrentUserRole() != types.UserRoleAdmin && body.CustomerOrganizationID == nil {
		http.Error(w, "user must be admin to create non-customer users", http.StatusForbidden)
		return
	}

	var customerOrganization *types.CustomerOrganizationWithUsage
	if body.CustomerOrganizationID != nil {
		if co, err := db.GetCustomerOrganizationByID(
			ctx,
			*body.CustomerOrganizationID,
		); errors.Is(err, apierrors.ErrNotFound) || (err == nil && co.OrganizationID != *auth.CurrentOrgID()) {
			http.Error(w, "customer does not exist", http.StatusBadRequest)
			return
		} else if err != nil {
			err = fmt.Errorf("failed to get customer: %w", err)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			customerOrganization = co
		}
	}

	userAccount := types.UserAccount{
		Email: body.Email,
		Name:  body.Name,
	}
	var inviteURL string

	if err := db.RunTx(ctx, func(ctx context.Context) error {
		organization, err := db.GetOrganizationWithBranding(ctx, *auth.CurrentOrgID())
		if err != nil {
			err = fmt.Errorf("failed to get org with branding: %w", err)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		}

		if body.UserRole != types.UserRoleAdmin && !organization.SubscriptionType.IsPro() {
			err = errors.New("creating non-admin users requires a pro subscription")
			http.Error(w, err.Error(), http.StatusForbidden)
			return err
		}

		var limitReached bool
		if customerOrganization != nil {
			limitReached, err = subscription.IsCustomerUserAccountLimitReached(
				organization.Organization,
				*customerOrganization,
			)
			if err != nil {
				err = fmt.Errorf("failed to check customer user account limit: %w", err)
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			}
		} else {
			limitReached, err = subscription.IsVendorUserAccountLimitReached(ctx, organization.Organization)
			if err != nil {
				err = fmt.Errorf("failed to check vendor user account limit: %w", err)
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			}
		}

		if limitReached {
			err = errors.New("user limit reached")
			http.Error(w, err.Error(), http.StatusForbidden)
			return err
		}

		userHasExisted := false
		if existingUA, err := db.GetUserAccountByEmail(ctx, body.Email); errors.Is(err, apierrors.ErrNotFound) {
			if err := db.CreateUserAccount(ctx, &userAccount); err != nil {
				err = fmt.Errorf("failed to create user account: %w", err)
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			}
		} else if err != nil {
			err = fmt.Errorf("failed to get existing user account: %w", err)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		} else {
			userHasExisted = true
			userAccount = *existingUA
		}

		if err := db.CreateUserAccountOrganizationAssignment(
			ctx,
			userAccount.ID,
			organization.ID,
			body.UserRole,
			body.CustomerOrganizationID,
		); errors.Is(err, apierrors.ErrAlreadyExists) {
			http.Error(w, "user is already part of this organization", http.StatusBadRequest)
			return err
		} else if err != nil {
			err = fmt.Errorf("failed to create user org assignment: %w", err)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if !userHasExisted || userAccount.EmailVerifiedAt == nil {
			if inviteURL, err = generateUserInviteUrl(userAccount, organization.Organization); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		}

		if err := mailsending.SendUserInviteMail(
			ctx,
			userAccount,
			*organization,
			body.CustomerOrganizationID,
			inviteURL,
		); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		return nil
	}); err != nil {
		log.Warn("could not create user", zap.Error(err))
		return
	}

	RespondJSON(w, api.CreateUserAccountResponse{
		User:      userAccount.AsUserAccountWithRole(body.UserRole, body.CustomerOrganizationID, time.Now()),
		InviteURL: inviteURL,
	})
}

func patchUserAccountHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		userAccount := internalctx.GetUserAccount(ctx)

		if *auth.CurrentUserRole() != types.UserRoleAdmin {
			if auth.CurrentCustomerOrgID() != nil {
				http.Error(w, "admin role needed to patch user", http.StatusForbidden)
				return
			}

			if userAccount.CustomerOrganizationID == nil {
				http.Error(w, "admin role needed to patch non-customer user", http.StatusForbidden)
				return
			}
		}

		body, err := JsonBody[api.PatchUserAccountRequest](w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		isUpdateNeeded := false

		if body.Name != nil && *body.Name != userAccount.Name {
			userAccount.Name = *body.Name
			isUpdateNeeded = true
		}

		if body.UserRole != nil && *body.UserRole != userAccount.UserRole {
			if userAccount.ID == auth.CurrentUserID() {
				http.Error(w, "users cannot change their own role", http.StatusForbidden)
				return
			}
			err = db.UpdateUserAccountOrganizationAssignment(
				ctx,
				userAccount.ID,
				*auth.CurrentOrgID(),
				*body.UserRole,
				userAccount.CustomerOrganizationID,
			)
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
				return
			} else if err != nil {
				log.Info("user update failed", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else {
				userAccount.UserRole = *body.UserRole
			}
		}

		if isUpdateNeeded {
			user := userAccount.AsUserAccount()
			if err := db.UpdateUserAccount(ctx, &user); err != nil {
				log.Info("user update failed", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			*userAccount = user.AsUserAccountWithRole(
				userAccount.UserRole,
				userAccount.CustomerOrganizationID,
				userAccount.JoinedOrgAt,
			)
		}

		RespondJSON(w, mapping.UserAccountToAPI(*userAccount))
	}
}

func resendUserInviteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		userAccountIDStr := r.PathValue("userId")
		userAccountID, err := uuid.Parse(userAccountIDStr)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		userAccount, err := db.GetUserAccountWithRole(ctx, userAccountID, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			err = fmt.Errorf("failed to get org with branding: %w", err)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if userAccount.EmailVerified {
			http.Error(w, "UserAccount is already verified", http.StatusBadRequest)
			return
		}

		organization, err := db.GetOrganizationWithBranding(ctx, *auth.CurrentOrgID())
		if err != nil {
			err = fmt.Errorf("failed to get org with branding: %w", err)
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		inviteURL, err := generateUserInviteUrl(userAccount.AsUserAccount(), organization.Organization)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := mailsending.SendUserInviteMail(
			ctx,
			userAccount.AsUserAccount(),
			*organization,
			userAccount.CustomerOrganizationID,
			inviteURL,
		); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, api.CreateUserAccountResponse{User: *userAccount, InviteURL: inviteURL})
	}
}

func deleteUserAccountHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	userAccount := internalctx.GetUserAccount(ctx)
	auth := auth.Authentication.Require(ctx)

	if *auth.CurrentUserRole() != types.UserRoleAdmin {
		if auth.CurrentCustomerOrgID() != nil {
			http.Error(w, "admin role needed to delete user", http.StatusForbidden)
			return
		}

		if userAccount.CustomerOrganizationID == nil {
			http.Error(w, "admin role needed to delete non-customer user", http.StatusForbidden)
			return
		}
	}

	if userAccount.ID == auth.CurrentUserID() {
		http.Error(w, "UserAccount deleting themselves is not allowed", http.StatusForbidden)
	} else if err := db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.DeleteUserAccountFromOrganization(ctx, userAccount.ID, *auth.CurrentOrgID()); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return nil
			} else {
				return err
			}
		} else if err := db.DeleteAccessTokensOfUserInOrg(ctx, userAccount.ID, *auth.CurrentOrgID()); err != nil {
			return err
		} else if err := db.DeleteTutorialProgressesOfUserInOrg(ctx, userAccount.ID, *auth.CurrentOrgID()); err != nil {
			return err
		} else {
			w.WriteHeader(http.StatusNoContent)
			return nil
		}
	}); err != nil {
		log.Error("error removing user from org", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

var patchImageUserAccount = patchImageHandler(func(ctx context.Context, body api.PatchImageRequest) (any, error) {
	user := internalctx.GetUserAccount(ctx)
	if err := db.UpdateUserAccountImage(ctx, user, body.ImageID); err != nil {
		return nil, err
	} else {
		return mapping.UserAccountToAPI(*user), nil
	}
})

func userAccountMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		log := internalctx.GetLogger(ctx)
		if userId, err := uuid.Parse(r.PathValue("userId")); err != nil {
			http.NotFound(w, r)
		} else if userAccount, err := db.GetUserAccountWithRole(
			ctx,
			userId,
			*auth.CurrentOrgID(),
			auth.CurrentCustomerOrgID(),
		); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				log.Warn("error getting user", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			h.ServeHTTP(w, r.WithContext(internalctx.WithUserAccount(ctx, userAccount)))
		}
	})
}

func generateUserInviteUrl(userAccount types.UserAccount, organization types.Organization) (string, error) {
	// TODO: Should probably use a different mechanism for invite tokens but for now this should work OK
	if _, token, err := authjwt.GenerateVerificationTokenValidFor(userAccount); err != nil {
		return "", fmt.Errorf("failed to generate invite URL: %w", err)
	} else {
		return fmt.Sprintf(
			"%v/join?jwt=%v",
			customdomains.AppDomainOrDefault(organization),
			url.QueryEscape(token),
		), nil
	}
}
