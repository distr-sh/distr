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
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
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

			inviteUserRateLimiter := httprate.Limit(
				3,
				10*time.Minute,
				httprate.WithKeyFuncs(middleware.RateLimitUserIDKey, middleware.RateLimitPathValueKey("userId")),
			)

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
}

func getUserAccountsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	var userAccounts []types.UserAccountWithUserRole
	var err error

	if customerOrgID := auth.CurrentCustomerOrgID(); customerOrgID != nil {
		userAccounts, err = db.GetUserAccountsByCustomerOrgID(ctx, *customerOrgID)
	} else if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		userAccounts, err = db.GetUserAccountsByPartnerOrgID(ctx, *partnerOrgID)
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

func createUserAccountHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	body, err := JsonBody[api.CreateUserAccountRequest](w, r)
	if err != nil {
		return
	}

	if err := validateCreateUserAccount(ctx, &body); err != nil {
		if errors.Is(err, apierrors.ErrBadRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if errors.Is(err, apierrors.ErrForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			log.Error("create user validation error", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	var customerOrganization *types.CustomerOrganizationWithUsage
	if body.CustomerOrganizationID != nil {
		customerOrganization, err = getVerifiedCustomerOrganization(ctx, *body.CustomerOrganizationID)
		if err != nil {
			if errors.Is(err, apierrors.ErrBadRequest) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				log.Error("create user getting customer organization error", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
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

		if limitReached, err := checkUserCreationLimits(ctx, organization.Organization, customerOrganization); err != nil {
			log.Error("limit check error", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		} else if limitReached {
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
			body.PartnerOrganizationID,
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
		User: userAccount.AsUserAccountWithRole(
			body.UserRole, body.CustomerOrganizationID, body.PartnerOrganizationID, time.Now()),
		InviteURL: inviteURL,
	})
}

func validateCreateUserAccount(ctx context.Context, body *api.CreateUserAccountRequest) error {
	auth := auth.Authentication.Require(ctx)

	if customerOrgID := auth.CurrentCustomerOrgID(); customerOrgID != nil {
		if *auth.CurrentUserRole() != types.UserRoleAdmin {
			return apierrors.NewForbidden("must be admin to create users")
		}
		body.CustomerOrganizationID = customerOrgID
	} else if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		// Partners can only create users in their own partner org or their assigned customers.
		if body.PartnerOrganizationID != nil {
			if *auth.CurrentUserRole() != types.UserRoleAdmin {
				return apierrors.NewForbidden("must be admin to create users in the given scope")
			}
			if *body.PartnerOrganizationID != *partnerOrgID {
				return apierrors.NewForbidden("cannot create users for a different partner organization")
			}
		}
		if body.CustomerOrganizationID == nil {
			if *auth.CurrentUserRole() != types.UserRoleAdmin {
				return apierrors.NewForbidden("must be admin to create users in the given scope")
			}
			body.PartnerOrganizationID = partnerOrgID
		}
	} else if *auth.CurrentUserRole() != types.UserRoleAdmin &&
		body.CustomerOrganizationID == nil && body.PartnerOrganizationID == nil {
		return apierrors.NewForbidden("must be admin to create users in the given scope")
	}

	if body.CustomerOrganizationID != nil && body.PartnerOrganizationID != nil {
		return apierrors.NewBadRequest("cannot assign user to both a customer and a partner organization")
	}

	if body.PartnerOrganizationID != nil && auth.CurrentPartnerOrgID() == nil {
		err := db.ValidatePartnerOrgBelongsToOrg(ctx, *body.PartnerOrganizationID, *auth.CurrentOrgID())
		if errors.Is(err, db.ErrPartnerOrgNotInOrg) {
			return apierrors.NewBadRequest("partner organization does not exist")
		} else if err != nil {
			return fmt.Errorf("failed to validate partner org: %w", err)
		}
	}

	return nil
}

func getVerifiedCustomerOrganization(ctx context.Context, id uuid.UUID) (*types.CustomerOrganizationWithUsage, error) {
	auth := auth.Authentication.Require(ctx)

	co, err := db.GetCustomerOrganizationByID(ctx, id)
	if errors.Is(err, apierrors.ErrNotFound) {
		return nil, apierrors.NewBadRequest("customer does not exist")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	} else if co.OrganizationID != *auth.CurrentOrgID() {
		return nil, apierrors.NewBadRequest("customer does not exist")
	}

	// Partner users can only create users in customers assigned to their partner org.
	partnerOrgID := auth.CurrentPartnerOrgID()
	if partnerOrgID != nil && !util.PtrEq(co.PartnerOrganizationID, partnerOrgID) {
		return nil, apierrors.NewForbidden("customer is not assigned to your partner organization")
	}

	return co, nil
}

func checkUserCreationLimits(
	ctx context.Context,
	organization types.Organization,
	customerOrganization *types.CustomerOrganizationWithUsage,
) (bool, error) {
	if customerOrganization != nil {
		limitReached, err := subscription.IsCustomerUserAccountLimitReached(organization, *customerOrganization)
		if err != nil {
			return false, fmt.Errorf("failed to check customer user account limit: %w", err)
		}
		return limitReached, nil
	}

	limitReached, err := subscription.IsVendorUserAccountLimitReached(ctx, organization)
	if err != nil {
		return false, fmt.Errorf("failed to check vendor user account limit: %w", err)
	}
	return limitReached, nil
}

func patchUserAccountHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		userAccount := internalctx.GetUserAccount(ctx)

		body, err := JsonBody[api.PatchUserAccountRequest](w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := checkUserAccountWritability(ctx, *userAccount); err != nil {
			if errors.Is(err, apierrors.ErrForbidden) {
				http.Error(w, err.Error(), http.StatusForbidden)
			} else {
				log.Error("failed to check user account writability", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
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
				userAccount.PartnerOrganizationID,
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
				userAccount.PartnerOrganizationID,
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

		userAccount, err := db.GetUserAccountWithRole(
			ctx, userAccountID, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
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

	if err := checkUserAccountWritability(ctx, *userAccount); err != nil {
		if errors.Is(err, apierrors.ErrForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			log.Error("failed to check user account writability", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
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
			auth.CurrentPartnerOrgID(),
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

func checkUserAccountWritability(ctx context.Context, userAccount types.UserAccountWithUserRole) error {
	auth := auth.Authentication.Require(ctx)

	if *auth.CurrentUserRole() != types.UserRoleAdmin && (auth.CurrentCustomerOrgID() != nil ||
		(auth.CurrentPartnerOrgID() != nil && userAccount.CustomerOrganizationID == nil) ||
		(userAccount.CustomerOrganizationID == nil && userAccount.PartnerOrganizationID == nil)) {
		return apierrors.NewForbidden("admin role needed to patch user in the given scope")
	}

	return nil
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
