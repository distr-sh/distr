package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/distr-sh/distr/internal/httpstatus"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func (c *Client) ApplicationEntitlements() *ApplicationEntitlements {
	return &ApplicationEntitlements{config: c.config}
}

type ApplicationEntitlements struct {
	config *Config
}

func (c *ApplicationEntitlements) url(elem ...string) string {
	return c.config.apiUrl(append([]string{"api", "v1", "application-entitlements"}, elem...)...).String()
}

func (c *ApplicationEntitlements) List(ctx context.Context) ([]types.ApplicationEntitlement, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(), nil)
	if err != nil {
		return nil, err
	}
	return JsonResponse[[]types.ApplicationEntitlement](c.config.httpClient.Do(req))
}

func (c *ApplicationEntitlements) Get(ctx context.Context, id uuid.UUID) (*types.ApplicationEntitlement, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(id.String()), nil)
	if err != nil {
		return nil, err
	}
	return JsonResponse[*types.ApplicationEntitlement](c.config.httpClient.Do(req))
}

func (c *ApplicationEntitlements) Create(
	ctx context.Context,
	entitlement *types.ApplicationEntitlementWithVersions,
) (*types.ApplicationEntitlement, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(entitlement); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(), &buf)
	if err != nil {
		return nil, err
	}
	return JsonResponse[*types.ApplicationEntitlement](c.config.httpClient.Do(req))
}

func (c *ApplicationEntitlements) Update(
	ctx context.Context,
	entitlement *types.ApplicationEntitlementWithVersions,
) (*types.ApplicationEntitlement, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(entitlement); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.url(entitlement.ID.String()), &buf)
	if err != nil {
		return nil, err
	}
	return JsonResponse[*types.ApplicationEntitlement](c.config.httpClient.Do(req))
}

func (c *ApplicationEntitlements) Delete(ctx context.Context, id uuid.UUID) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.url(id.String()), nil)
	if err != nil {
		return err
	}
	_, err = httpstatus.CheckStatus(c.config.httpClient.Do(req))
	return err
}
