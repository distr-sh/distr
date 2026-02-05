package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/httpstatus"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func (c *Client) DeploymentTargets() *DeploymentTargets {
	return &DeploymentTargets{config: c.config}
}

type DeploymentTargets struct {
	config *Config
}

func (c *DeploymentTargets) url(elem ...string) string {
	return c.config.apiUrl(append([]string{"api", "v1", "deployment-targets"}, elem...)...).String()
}

func (c *DeploymentTargets) List(ctx context.Context) ([]types.DeploymentTargetFull, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(), nil)
	if err != nil {
		return nil, err
	}
	return JsonResponse[[]types.DeploymentTargetFull](c.config.httpClient.Do(req))
}

func (c *DeploymentTargets) Get(ctx context.Context, id uuid.UUID) (*types.DeploymentTargetFull, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(id.String()), nil)
	if err != nil {
		return nil, err
	}
	return JsonResponse[*types.DeploymentTargetFull](c.config.httpClient.Do(req))
}

func (c *DeploymentTargets) Create(
	ctx context.Context,
	req types.DeploymentTargetFull,
) (*types.DeploymentTargetFull, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(), &buf)
	if err != nil {
		return nil, err
	}
	return JsonResponse[*types.DeploymentTargetFull](c.config.httpClient.Do(httpReq))
}

func (c *DeploymentTargets) Update(
	ctx context.Context,
	req types.DeploymentTargetFull,
) (*types.DeploymentTargetFull, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(req.ID.String()), &buf)
	if err != nil {
		return nil, err
	}
	return JsonResponse[*types.DeploymentTargetFull](c.config.httpClient.Do(httpReq))
}

func (c *DeploymentTargets) Delete(ctx context.Context, id uuid.UUID) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.url(id.String()), nil)
	if err != nil {
		return err
	}
	_, err = httpstatus.CheckStatus(c.config.httpClient.Do(req))
	return err
}

func (c *DeploymentTargets) Connect(
	ctx context.Context,
	id uuid.UUID,
) (*api.DeploymentTargetAccessTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(id.String(), "access-request"), nil)
	if err != nil {
		return nil, err
	}
	return JsonResponse[*api.DeploymentTargetAccessTokenResponse](c.config.httpClient.Do(req))
}
