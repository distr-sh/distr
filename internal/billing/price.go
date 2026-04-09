package billing

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/types"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/price"
)

const (
	PriceKeyStarterCustomerMonthly = "distr_starter_customer_monthly"
	PriceKeyStarterCustomerYearly  = "distr_starter_customer_yearly"
	PriceKeyStarterUserMonthly     = "distr_starter_user_monthly"
	PriceKeyStarterUserYearly      = "distr_starter_user_yearly"
	PriceKeyProCustomerMonthly     = "distr_pro_customer_monthly"
	PriceKeyProCustomerYearly      = "distr_pro_customer_yearly"
	PriceKeyProUserMonthly         = "distr_pro_user_monthly"
	PriceKeyProUserYearly          = "distr_pro_user_yearly"
)

var (
	CustomerPriceKeys = []string{
		PriceKeyStarterCustomerMonthly,
		PriceKeyStarterCustomerYearly,
		PriceKeyProCustomerMonthly,
		PriceKeyProCustomerYearly,
	}
	UserPriceKeys = []string{
		PriceKeyStarterUserMonthly,
		PriceKeyStarterUserYearly,
		PriceKeyProUserMonthly,
		PriceKeyProUserYearly,
	}
	StarterPriceKeys = []string{
		PriceKeyStarterCustomerMonthly,
		PriceKeyStarterCustomerYearly,
		PriceKeyStarterUserMonthly,
		PriceKeyStarterUserYearly,
	}
	ProPriceKeys = []string{
		PriceKeyProCustomerMonthly,
		PriceKeyProCustomerYearly,
		PriceKeyProUserMonthly,
		PriceKeyProUserYearly,
	}
	MonthlyPriceKeys = []string{
		PriceKeyStarterCustomerMonthly,
		PriceKeyStarterUserMonthly,
		PriceKeyProCustomerMonthly,
		PriceKeyProUserMonthly,
	}
	YearlyPriceKeys = []string{
		PriceKeyStarterCustomerYearly,
		PriceKeyStarterUserYearly,
		PriceKeyProCustomerYearly,
		PriceKeyProUserYearly,
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
	case types.SubscriptionTypeStarter:
		switch subscriptionPeriod {
		case types.SubscriptionPeriodMonthly:
			customerPriceLookupKey = PriceKeyStarterCustomerMonthly
			userPriceLookupKey = PriceKeyStarterUserMonthly
		case types.SubscriptionPeriodYearly:
			customerPriceLookupKey = PriceKeyStarterCustomerYearly
			userPriceLookupKey = PriceKeyStarterUserYearly
		default:
			return nil, fmt.Errorf("invalid subscription period: %v", subscriptionPeriod)
		}
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
