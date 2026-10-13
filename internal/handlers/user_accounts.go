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
	"github.com/distr-sh/distr/internal/blocklist"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
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
		r.Get("/", respond.Handler(getUserAccountsHandler)).
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

			inviteUserRateLimiter := httprate.LimitBy(
				3,
				10*time.Minute,
				httprate.JoinKeys(middleware.RateLimitUserIDKey, middleware.RateLimitPathValueKey("userId")),
			)

			r.Use(userAccountMiddleware)
			r.With(middleware.ProFeature).
				Patch("/", respond.Handler(patchUserAccountHandler)).
				With(option.Description("Partially update a user account")).
				With(option.Request(struct {
					UserAccountRequest
					api.PatchUserAccountRequest
				}{})).
				With(option.Response(http.StatusOK, api.UserAccountResponse{}))
			r.Delete("/", respond.Handler(deleteUserAccountHandler)).
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
				Post("/invite", respond.Handler(resendUserInviteHandler)).
				With(option.Description("Resend user invite")).
				With(option.Request(UserAccountRequest{})).
				With(option.Response(http.StatusOK, api.CreateUserAccountResponse{}))
		})
	})
}

var userAccountNotFound = respond.NotFound("user not found")

func getUserAccountsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
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
		return nil, fmt.Errorf("failed to get user accounts: %w", err)
	} else {
		return respond.JSON(mapping.List(userAccounts, mapping.UserAccountToAPI)), nil
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
	if err := body.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if blocklist.EmailBlocked(body.Email) {
		http.Error(w, blocklist.EmailBlockedMessage, http.StatusForbidden)
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
	// An invite link lets its holder set the account's password, so the one in the response is only for an account
	// created here.
	var emailInviteURL, responseInviteURL string

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

		created := false
		if existingUA, err := db.GetUserAccountByEmail(ctx, body.Email); errors.Is(err, apierrors.ErrNotFound) {
			created = true
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

		if !userAccount.Activated {
			if emailInviteURL, err = generateUserInviteUrl(
				ctx, userAccount, *organization, body.CustomerOrganizationID, true); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			}
		}
		if created && !env.UserEmailVerificationRequired() {
			if responseInviteURL, err = generateUserInviteUrl(
				ctx, userAccount, *organization, body.CustomerOrganizationID, false); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			}
		}

		if err := mailsending.SendUserInviteMail(
			ctx,
			userAccount,
			*organization,
			body.CustomerOrganizationID,
			emailInviteURL,
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
		User: mapping.UserAccountToAPI(userAccount.AsUserAccountWithRole(
			body.UserRole, body.CustomerOrganizationID, body.PartnerOrganizationID, time.Now())),
		InviteURL: responseInviteURL,
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

	limitReached, err := subscription.IsBillableUserAccountLimitReached(ctx, organization)
	if err != nil {
		return false, fmt.Errorf("failed to check vendor user account limit: %w", err)
	}
	return limitReached, nil
}

func patchUserAccountHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	userAccount := internalctx.GetUserAccount(ctx)

	if body, resp := decodeJSON[api.PatchUserAccountRequest](r); resp != nil {
		return resp, nil
	} else if err := checkUserAccountWritability(ctx, *userAccount); err != nil {
		if errors.Is(err, apierrors.ErrForbidden) {
			return respond.Forbidden(err.Error()), nil
		} else {
			return nil, fmt.Errorf("failed to check user account writability: %w", err)
		}
	} else {
		isUpdateNeeded := false

		if body.Name != nil && *body.Name != userAccount.Name {
			userAccount.Name = *body.Name
			isUpdateNeeded = true
		}

		if body.UserRole != nil && *body.UserRole != userAccount.UserRole {
			if userAccount.ID == auth.CurrentUserID() {
				return respond.Forbidden("users cannot change their own role"), nil
			} else if err := db.UpdateUserAccountOrganizationAssignment(
				ctx,
				userAccount.ID,
				*auth.CurrentOrgID(),
				*body.UserRole,
				userAccount.CustomerOrganizationID,
				userAccount.PartnerOrganizationID,
			); errors.Is(err, apierrors.ErrNotFound) {
				return userAccountNotFound, nil
			} else if err != nil {
				return nil, fmt.Errorf("user update failed: %w", err)
			} else {
				userAccount.UserRole = *body.UserRole
			}
		}

		if isUpdateNeeded {
			user := userAccount.AsUserAccount()
			if err := db.UpdateUserAccount(ctx, &user); err != nil {
				return nil, fmt.Errorf("user update failed: %w", err)
			} else {
				*userAccount = user.AsUserAccountWithRole(
					userAccount.UserRole,
					userAccount.CustomerOrganizationID,
					userAccount.PartnerOrganizationID,
					userAccount.JoinedOrgAt,
				)
			}
		}

		return respond.JSON(mapping.UserAccountToAPI(*userAccount)), nil
	}
}

func resendUserInviteHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if userAccountID, err := uuid.Parse(r.PathValue("userId")); err != nil {
		return respond.BadRequest(http.StatusText(http.StatusBadRequest)), nil
	} else if userAccount, err := db.GetUserAccountWithRole(
		ctx, userAccountID, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		return userAccountNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user account: %w", err)
	} else if userAccount.Activated {
		return respond.BadRequest("this user has already accepted an invitation"), nil
	} else if organization, err := db.GetOrganizationWithBranding(ctx, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get org with branding: %w", err)
	} else if emailInviteURL, err := generateUserInviteUrl(
		ctx, userAccount.AsUserAccount(), *organization, userAccount.CustomerOrganizationID, true,
	); err != nil {
		return nil, err
	} else if err := mailsending.SendUserInviteMail(
		ctx,
		userAccount.AsUserAccount(),
		*organization,
		userAccount.CustomerOrganizationID,
		emailInviteURL,
	); err != nil {
		return nil, fmt.Errorf("failed to send user invite mail: %w", err)
	} else {
		return respond.JSON(api.CreateUserAccountResponse{User: mapping.UserAccountToAPI(*userAccount)}), nil
	}
}

func deleteUserAccountHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	userAccount := internalctx.GetUserAccount(ctx)
	auth := auth.Authentication.Require(ctx)

	if err := checkUserAccountWritability(ctx, *userAccount); err != nil {
		if errors.Is(err, apierrors.ErrForbidden) {
			return respond.Forbidden(err.Error()), nil
		} else {
			return nil, fmt.Errorf("failed to check user account writability: %w", err)
		}
	} else if userAccount.ID == auth.CurrentUserID() {
		return respond.Forbidden("UserAccount deleting themselves is not allowed"), nil
	} else if err := db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.DeleteUserAccountFromOrganization(ctx, userAccount.ID, *auth.CurrentOrgID()); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				return nil
			} else {
				return err
			}
		} else if err := db.DeleteAccessTokensOfUserInOrg(ctx, userAccount.ID, *auth.CurrentOrgID()); err != nil {
			return err
		} else if err := db.DeleteTutorialProgressesOfUserInOrg(ctx, userAccount.ID, *auth.CurrentOrgID()); err != nil {
			return err
		} else if err := db.DeleteCustomOIDCIdentitiesOfUserInOrg(
			ctx, userAccount.ID, *auth.CurrentOrgID()); err != nil {
			return err
		} else {
			return nil
		}
	}); err != nil {
		return nil, fmt.Errorf("error removing user from org: %w", err)
	} else {
		return respond.NoContent(), nil
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

func generateUserInviteUrl(
	ctx context.Context,
	userAccount types.UserAccount,
	organization types.OrganizationWithBranding,
	customerOrgID *uuid.UUID,
	emailVerified bool,
) (string, error) {
	if _, token, err := authjwt.GenerateInviteToken(userAccount, emailVerified); err != nil {
		return "", fmt.Errorf("failed to generate invite URL: %w", err)
	} else {
		return fmt.Sprintf(
			"%v/join?jwt=%v",
			customdomains.CustomerPortalDomainOrDefault(ctx, organization.ID, customerOrgID, organization.Branding),
			url.QueryEscape(token),
		), nil
	}
}
