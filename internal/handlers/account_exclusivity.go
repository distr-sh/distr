package handlers

import (
	"context"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/db"
	"github.com/google/uuid"
)

// An account that can sign in through an organization's own identity provider must be a member of
// that one organization and of nothing else. Otherwise the provider is a way into every other
// organization the account belongs to: whoever controls it can assert any email address, and
// nothing about the resulting session would be limited to the organization that configured it.
//
// The rule is enforced when the provider authenticates somebody (see resolveCustomOIDCUser) and
// kept true by the two guards below, which are the only ways an additional membership can appear.
// Both derive their answer from the account's current state rather than from a token claim, so they
// apply to a personal access token exactly like they apply to a browser session.

// checkOrganizationCreationAllowed reports whether the account may create another organization.
// Doing so would break the exclusivity of an account that signs in through an organization's
// identity provider, and the next login through it would be refused — so this protects the user as
// much as the organization.
func checkOrganizationCreationAllowed(ctx context.Context, userID uuid.UUID) error {
	governed, err := db.ExistsUserAccountCustomOIDCIdentity(ctx, userID)
	if err != nil {
		return err
	}
	if governed {
		return apierrors.NewBadRequest(
			"your account signs in through an organization's identity provider and can therefore only be a " +
				"member of that organization")
	}
	return nil
}

// checkMembershipAllowed reports whether an existing account may be added to the given
// organization. It refuses in both directions: an account that signs in through some organization's
// identity provider must not gain a second membership, and an organization that has an identity
// provider configured - including a disabled one, which can be re-enabled at any time - must not
// take in an account that already belongs elsewhere. Better a refused invite than a user whose
// single sign-on silently stops working.
func checkMembershipAllowed(ctx context.Context, userID, organizationID uuid.UUID) error {
	governed, err := db.ExistsUserAccountCustomOIDCIdentity(ctx, userID)
	if err != nil {
		return err
	}
	if governed {
		return apierrors.NewBadRequest(
			"this user signs in through an organization's identity provider and can therefore only be a member " +
				"of that organization")
	}

	hasConfiguration, err := db.ExistsCustomOIDCConfigurationForOrganization(ctx, organizationID)
	if err != nil {
		return err
	}
	if !hasConfiguration {
		return nil
	}
	otherOrganizations, err := db.CountUserAccountOrganizationsExcept(ctx, userID, organizationID)
	if err != nil {
		return err
	}
	if otherOrganizations > 0 {
		return apierrors.NewBadRequest(
			"this organization uses its own identity provider, so it can only contain users who are not a " +
				"member of another Distr organization, which this user already is")
	}
	return nil
}
