package handlers

import (
	"context"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/db"
	"github.com/google/uuid"
)

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
