package types

import (
	"encoding/base64"
	"fmt"
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/limit"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
)

type Organization struct {
	ID                                  uuid.UUID          `db:"id" json:"id"`
	CreatedAt                           time.Time          `db:"created_at" json:"createdAt"`
	Name                                string             `db:"name" json:"name"`
	Slug                                *string            `db:"slug" json:"slug"`
	Features                            []Feature          `db:"features" json:"features"`
	AppDomain                           *string            `db:"app_domain" json:"appDomain"`
	RegistryDomain                      *string            `db:"registry_domain" json:"registryDomain"`
	EmailFromAddress                    *string            `db:"email_from_address" json:"emailFromAddress"`
	SubscriptionType                    SubscriptionType   `db:"subscription_type" json:"subscriptionType"`
	SubscriptionPeriod                  SubscriptionPeriod `db:"subscription_period" json:"subscriptionPeriod"`
	SubscriptionEndsAt                  time.Time          `db:"subscription_ends_at" json:"subscriptionEndsAt"`
	StripeCustomerID                    *string            `db:"stripe_customer_id" json:"stripeCustomerId"`
	StripeSubscriptionID                *string            `db:"stripe_subscription_id" json:"stripeSubscriptionId"`
	SubscriptionCustomerOrganizationQty limit.Limit        `db:"subscription_customer_organization_quantity" json:"subscriptionCustomerOrganizationQuantity"` //nolint:lll
	SubscriptionUserAccountQty          limit.Limit        `db:"subscription_user_account_quantity" json:"subscriptionUserAccountQuantity"`                   //nolint:lll
	PreConnectScript                    *string            `db:"pre_connect_script" json:"preConnectScript"`
	PostConnectScript                   *string            `db:"post_connect_script" json:"postConnectScript"`
	ConnectScriptIsSudo                 bool               `db:"connect_script_is_sudo" json:"connectScriptIsSudo"`
}

func (org *Organization) HasFeature(feature Feature) bool {
	return slices.Contains(org.Features, feature)
}

func (org *Organization) AddFeatures(features ...Feature) {
	for _, feature := range features {
		if !org.HasFeature(feature) {
			org.Features = append(org.Features, feature)
		}
	}
}

func (org *Organization) RemoveFeatures(features ...Feature) {
	org.Features = slices.DeleteFunc(org.Features, func(f Feature) bool { return slices.Contains(features, f) })
}

func (org *Organization) SetFeature(feature Feature, enabled bool) {
	if enabled {
		org.AddFeatures(feature)
	} else {
		org.RemoveFeatures(feature)
	}
}

func (org *Organization) HasActiveSubscription() bool {
	return org.SubscriptionType == SubscriptionTypeCommunity || org.SubscriptionEndsAt.After(time.Now())
}

func (org *Organization) HasActiveSubscriptionWithType(st SubscriptionType) bool {
	return org.HasActiveSubscription() && org.SubscriptionType == st
}

type OrganizationWithUserRole struct {
	Organization
	UserRole                 UserRole   `db:"user_role" json:"userRole"`
	CustomerOrganizationID   *uuid.UUID `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
	CustomerOrganizationName *string    `db:"customer_organization_name" json:"customerOrganizationName,omitempty"`
	JoinedOrgAt              time.Time  `db:"joined_org_at" json:"joinedOrgAt"`
}

type OrganizationWithBranding struct {
	Organization
	Branding *OrganizationBranding `db:"branding"`
}

type OrganizationBranding struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	CreatedAt              time.Time  `db:"created_at" json:"createdAt"`
	OrganizationID         uuid.UUID  `db:"organization_id" json:"-"`
	UpdatedAt              time.Time  `db:"updated_at" json:"updatedAt"`
	UpdatedByUserAccountID *uuid.UUID `db:"updated_by_user_account_id" json:"-"`
	Title                  *string    `db:"title" json:"title"`
	Description            *string    `db:"description" json:"description"`
	Logo                   []byte     `db:"logo" json:"logo"`
	LogoFileName           *string    `db:"logo_file_name" json:"logoFileName"`
	LogoContentType        *string    `db:"logo_content_type" json:"logoContentType"`
}

func (b *OrganizationBranding) LogoDataUrl() *string {
	if b.Logo != nil && b.LogoContentType != nil {
		return util.PtrTo(fmt.Sprintf(
			"data:%s;base64,%s",
			*b.LogoContentType,
			base64.StdEncoding.EncodeToString(b.Logo),
		))
	} else {
		return nil
	}
}
