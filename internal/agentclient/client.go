package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentclient/useragent"
	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/deploymentlogs"
	"github.com/distr-sh/distr/internal/deploymenttargetlogs"
	"github.com/distr-sh/distr/internal/httpstatus"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type clientData struct {
	authTarget                   string
	authSecret                   string
	loginEndpoint                string
	manifestEndpoint             string
	resourceEndpoint             string
	statusEndpoint               string
	metricsEndpoint              string
	deploymentLogsEndpoint       string
	deploymentTargetLogsEndpoint string
}

type Client struct {
	clientData
	httpClient *http.Client
	logger     *zap.Logger
	token      jwt.Token
	rawToken   string
	mutex      sync.Mutex
}

func (c *Client) Resource(ctx context.Context) (*api.AgentResource, error) {
	var result api.AgentResource
	if req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resourceEndpoint, nil); err != nil {
		return nil, err
	} else {
		req.Header.Set("Content-Type", "application/json")
		if resp, err := c.doAuthenticated(ctx, req, true); err != nil {
			return nil, err
		} else if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		} else {
			return &result, nil
		}
	}
}

func (c *Client) Manifest(ctx context.Context) ([]byte, error) {
	if req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.manifestEndpoint, nil); err != nil {
		return nil, err
	} else if resp, err := c.doAuthenticated(ctx, req, true); err != nil {
		return nil, err
	} else if data, err := io.ReadAll(resp.Body); err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func (c *Client) StatusWithError(ctx context.Context, revisionID uuid.UUID, err error) error {
	return c.Status(ctx, revisionID, types.DeploymentStatusTypeError, err.Error())
}

func (c *Client) Status(
	ctx context.Context,
	revisionID uuid.UUID,
	statusType types.DeploymentStatusType,
	message string,
) error {
	deploymentStatus := api.AgentDeploymentStatus{
		RevisionID: revisionID,
		Message:    message,
		Type:       statusType,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(deploymentStatus); err != nil {
		return err
	} else if req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.statusEndpoint, &buf); err != nil {
		return err
	} else {
		req.Header.Set("Content-Type", "application/json")
		if _, err := c.doAuthenticated(ctx, req, true); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (c *Client) ExportDeploymentLogs(ctx context.Context, records []api.DeploymentLogRecord) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(records); err != nil {
		return err
	} else if req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.deploymentLogsEndpoint, &buf); err != nil {
		return err
	} else {
		req.Header.Set("Content-Type", "application/json")
		_, err := c.doAuthenticated(ctx, req, true)
		return err
	}
}

func (c *Client) ExportDeploymentTargetLogs(records ...api.DeploymentTargetLogRecord) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(records); err != nil {
		return err
	} else if req, err := http.NewRequest(http.MethodPut, c.deploymentTargetLogsEndpoint, &buf); err != nil {
		return err
	} else {
		req.Header.Set("Content-Type", "application/json")
		_, err := c.doAuthenticated(context.TODO(), req, false)
		return err
	}
}

func (c *Client) Login(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.loginEndpoint, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.authTarget, c.authSecret)
	if resp, err := c.do(req); err != nil {
		return err
	} else {
		var loginResponse api.AuthLoginResponse
		if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
			return err
		}
		if parsedToken, err := jwt.ParseInsecure([]byte(loginResponse.Token)); err != nil {
			return err
		} else {
			c.rawToken = loginResponse.Token
			c.token = parsedToken
			return nil
		}
	}
}

func (c *Client) EnsureToken(ctx context.Context, loggingEnabled bool) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.HasTokenExpiredAfter(time.Now().Add(30 * time.Second)) {
		if loggingEnabled {
			c.logger.Info("token has expired or is about to expire")
		}
		if err := c.Login(ctx); err != nil {
			if c.HasTokenExpired() {
				return fmt.Errorf("login failed: %w", err)
			} else if loggingEnabled {
				c.logger.Warn("token refresh failed but previous token is still valid", zap.Error(err))
			}
		} else if loggingEnabled {
			c.logger.Info("token refreshed")
		}
	}
	return nil
}

func (c *Client) HasTokenExpired() bool {
	return c.HasTokenExpiredAfter(time.Now())
}

func (c *Client) HasTokenExpiredAfter(t time.Time) bool {
	return c.token == nil || c.token.Expiration().Before(t)
}

func (c *Client) ClearToken() {
	c.token = nil
	c.rawToken = ""
}

func (c *Client) RawToken() string {
	return c.rawToken
}

func (c *Client) ReportMetrics(ctx context.Context, metrics api.AgentDeploymentTargetMetrics) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(metrics); err != nil {
		return err
	} else if req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.metricsEndpoint, &buf); err != nil {
		return err
	} else {
		req.Header.Set("Content-Type", "application/json")
		if _, err := c.doAuthenticated(ctx, req, true); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (c *Client) doAuthenticated(ctx context.Context, r *http.Request, loggingEnabled bool) (*http.Response, error) {
	if resp, err := c.doAuthenticatedNoRetry(ctx, r, loggingEnabled); resp == nil || resp.StatusCode != 401 {
		return resp, err
	} else {
		if loggingEnabled {
			c.logger.Warn("got 401 response, try to regenerate token")
		}
		c.ClearToken()
		resp, err1 := c.doAuthenticatedNoRetry(ctx, r, loggingEnabled)
		if err1 != nil {
			return resp, multierr.Append(err, err1)
		} else {
			return resp, nil
		}
	}
}

func (c *Client) doAuthenticatedNoRetry(
	ctx context.Context,
	r *http.Request,
	loggingEnabled bool,
) (*http.Response, error) {
	if err := c.EnsureToken(ctx, loggingEnabled); err != nil {
		return nil, err
	} else {
		r.Header.Set("Authorization", "Bearer "+c.rawToken)
		return c.do(r)
	}
}

func (c *Client) do(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", fmt.Sprintf("%v/%v", useragent.DistrAgentUserAgent, buildconfig.Version()))
	return httpstatus.CheckStatus(c.httpClient.Do(r))
}

func (c *Client) ReloadFromEnv() (changed bool, err error) {
	var d clientData
	if d.authTarget, err = readEnvVar("DISTR_TARGET_ID"); err != nil {
		return changed, err
	} else if d.authSecret, err = readEnvVar("DISTR_TARGET_SECRET"); err != nil {
		return changed, err
	} else if d.loginEndpoint, err = readEnvVar("DISTR_LOGIN_ENDPOINT"); err != nil {
		return changed, err
	} else if d.manifestEndpoint, err = readEnvVar("DISTR_MANIFEST_ENDPOINT"); err != nil {
		return changed, err
	} else if d.resourceEndpoint, err = readEnvVar("DISTR_RESOURCE_ENDPOINT"); err != nil {
		return changed, err
	} else if d.statusEndpoint, err = readEnvVar("DISTR_STATUS_ENDPOINT"); err != nil {
		return changed, err
	} else if d.metricsEndpoint, err = readEnvVar("DISTR_METRICS_ENDPOINT"); err != nil {
		return changed, err
	} else if d.deploymentLogsEndpoint, err = readEnvVar("DISTR_LOGS_ENDPOINT"); err != nil {
		return changed, err
	} else if d.deploymentTargetLogsEndpoint, err = readEnvVar("DISTR_AGENT_LOGS_ENDPOINT"); err != nil {
		return changed, err
	} else {
		changed = c.clientData != d
		if changed {
			c.clientData = d
			c.ClearToken()
		}
		return changed, err
	}
}

func NewFromEnv(logger *zap.Logger) (*Client, error) {
	client := Client{
		httpClient: &http.Client{},
		logger:     logger,
	}
	if _, err := client.ReloadFromEnv(); err != nil {
		return nil, err
	}
	return &client, nil
}

func readEnvVar(key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	} else {
		return "", fmt.Errorf("missing environment variable: %v", key)
	}
}

var (
	_ deploymenttargetlogs.Exporter = (*Client)(nil)
	_ deploymentlogs.Exporter       = (*Client)(nil)
)
