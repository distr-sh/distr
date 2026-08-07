package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/distr-sh/distr/internal/env"
)

const SiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

const (
	requestTimeout   = 10 * time.Second
	maxResponseBytes = 64 * 1024
)

type Client struct {
	secret   string
	endpoint string
	client   *http.Client
}

// NewClient creates a client verifying tokens with the given secret. An empty endpoint means SiteverifyURL.
func NewClient(secret, endpoint string) *Client {
	if endpoint == "" {
		endpoint = SiteverifyURL
	}
	return &Client{secret: secret, endpoint: endpoint, client: &http.Client{Timeout: requestTimeout}}
}

// Tokens are only ever verified while serving a request, which is long after env.Initialize, so reading the
// secret on first use is safe.
var fromEnv = sync.OnceValue(func() *Client {
	var secret string
	if configured := env.TurnstileSecret(); configured != nil {
		secret = *configured
	}
	return NewClient(secret, SiteverifyURL)
})

func Verify(ctx context.Context, token, remoteIP string) error {
	return fromEnv().Verify(ctx, token, remoteIP)
}

// Verify fails closed: an unreachable or unexpectedly responding siteverify API is reported as a failed
// verification, never as a pass, so an outage cannot silently open up the endpoint it protects.
func (c *Client) Verify(ctx context.Context, token, remoteIP string) error {
	if c.secret == "" {
		return errors.New("no turnstile secret configured")
	}
	if token == "" {
		return errors.New("no turnstile token given")
	}

	form := url.Values{"secret": {c.secret}, "response": {token}}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("could not create siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("siteverify request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("siteverify returned unexpected status %v", resp.StatusCode)
	}

	var result siteverifyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
		return fmt.Errorf("could not decode siteverify response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("token rejected by siteverify: %v", strings.Join(result.ErrorCodes, ", "))
	}
	return nil
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}
