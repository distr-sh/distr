package billing

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/types"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/price"
)

const (
	PriceKeyProCustomerMonthly      = "distr_pro_customer_monthly"
	PriceKeyProCustomerYearly       = "distr_pro_customer_yearly"
	PriceKeyProUserMonthly          = "distr_pro_user_monthly"
	PriceKeyProUserYearly           = "distr_pro_user_yearly"
	PriceKeyBusinessCustomerMonthly = "distr_business_customer_monthly"
	PriceKeyBusinessCustomerYearly  = "distr_business_customer_yearly"
	PriceKeyBusinessUserMonthly     = "distr_business_user_monthly"
	PriceKeyBusinessUserYearly      = "distr_business_user_yearly"
)

var (
	CustomerPriceKeys = []string{
		PriceKeyProCustomerMonthly,
		PriceKeyProCustomerYearly,
		PriceKeyBusinessCustomerMonthly,
		PriceKeyBusinessCustomerYearly,
	}
	UserPriceKeys = []string{
		PriceKeyProUserMonthly,
		PriceKeyProUserYearly,
		PriceKeyBusinessUserMonthly,
		PriceKeyBusinessUserYearly,
	}
	ProPriceKeys = []string{
		PriceKeyProCustomerMonthly,
		PriceKeyProCustomerYearly,
		PriceKeyProUserMonthly,
		PriceKeyProUserYearly,
	}
	BusinessPriceKeys = []string{
		PriceKeyBusinessCustomerMonthly,
		PriceKeyBusinessCustomerYearly,
		PriceKeyBusinessUserMonthly,
		PriceKeyBusinessUserYearly,
	}
	MonthlyPriceKeys = []string{
		PriceKeyProCustomerMonthly,
		PriceKeyProUserMonthly,
		PriceKeyBusinessCustomerMonthly,
		PriceKeyBusinessUserMonthly,
	}
	YearlyPriceKeys = []string{
		PriceKeyProCustomerYearly,
		PriceKeyProUserYearly,
		PriceKeyBusinessCustomerYearly,
		PriceKeyBusinessUserYearly,
	}
)

type PriceIDs struct {
	CustomerPriceID string
	UserPriceID     string
}

func GetStripePrices(
	ctx context.Context,
	subscriptionType types.SubscriptionType,
	subscriptionPeriod types.SubscriptionPeriod,
) (*PriceIDs, error) {
	var customerPriceLookupKey string
	var userPriceLookupKey string

	switch subscriptionType {
	case types.SubscriptionTypePro:
		switch subscriptionPeriod {
		case types.SubscriptionPeriodMonthly:
			customerPriceLookupKey = PriceKeyProCustomerMonthly
			userPriceLookupKey = PriceKeyProUserMonthly
		case types.SubscriptionPeriodYearly:
			customerPriceLookupKey = PriceKeyProCustomerYearly
			userPriceLookupKey = PriceKeyProUserYearly
		default:
			return nil, fmt.Errorf("invalid subscription period: %v", subscriptionPeriod)
		}
	case types.SubscriptionTypeBusiness:
		switch subscriptionPeriod {
		case types.SubscriptionPeriodMonthly:
			customerPriceLookupKey = PriceKeyBusinessCustomerMonthly
			userPriceLookupKey = PriceKeyBusinessUserMonthly
		case types.SubscriptionPeriodYearly:
			customerPriceLookupKey = PriceKeyBusinessCustomerYearly
			userPriceLookupKey = PriceKeyBusinessUserYearly
		default:
			return nil, fmt.Errorf("invalid subscription period: %v", subscriptionPeriod)
		}
	default:
		return nil, fmt.Errorf("invalid subscription type: %v", subscriptionType)
	}

	lookupKeys := []string{customerPriceLookupKey, userPriceLookupKey}
	listPriceResult := price.List(&stripe.PriceListParams{
		ListParams: stripe.ListParams{Context: ctx},
		LookupKeys: stripe.StringSlice(lookupKeys),
	})

	var result PriceIDs
	for listPriceResult.Next() {
		price := listPriceResult.Price()
		switch price.LookupKey {
		case customerPriceLookupKey:
			result.CustomerPriceID = price.ID
		case userPriceLookupKey:
			result.UserPriceID = price.ID
		}
	}

	if err := listPriceResult.Err(); err != nil {
		return nil, err
	}

	if result.CustomerPriceID == "" || result.UserPriceID == "" {
		return nil, fmt.Errorf("failed to find prices for lookupKeys:  %v", lookupKeys)
	}

	return &result, nil
}
