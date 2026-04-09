package licensetemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"text/template"

	"github.com/distr-sh/distr/internal/types"
	"github.com/stripe/stripe-go/v85"
)

func RenderPayload(tmpl types.LicenseTemplate, sub stripe.Subscription) (json.RawMessage, error) {
	t, err := template.New("payload").Funcs(templateFuncMap(sub)).Parse(tmpl.PayloadTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, sub); err != nil {
		return nil, fmt.Errorf("failed to render payload template: %w", err)
	}

	raw := json.RawMessage(buf.Bytes())
	if !json.Valid(raw) {
		return nil, fmt.Errorf("rendered payload template is not valid JSON")
	}
	return raw, nil
}

func templateFuncMap(sub stripe.Subscription) template.FuncMap {
	return template.FuncMap{
		"hasItem": func(keys ...string) bool {
			return slices.ContainsFunc(sub.Items.Data, func(item *stripe.SubscriptionItem) bool {
				return item != nil && item.Price != nil && slices.Contains(keys, item.Price.LookupKey)
			})
		},
		"itemQuantity": func(key string) int64 {
			for _, item := range sub.Items.Data {
				if item != nil && item.Price != nil && item.Price.LookupKey == key {
					return item.Quantity
				}
			}
			return 0
		},
	}
}
