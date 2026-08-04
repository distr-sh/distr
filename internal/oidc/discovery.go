package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/distr-sh/distr/internal/env"
)

const (
	discoveryPath    = "/.well-known/openid-configuration"
	discoveryTimeout = 10 * time.Second
	entraHost        = "login.microsoftonline.com"
)

var entraMultiTenantPaths = []string{"common", "organizations", "consumers"}

type DiscoveryResult struct {
	Issuer        string
	Provider      *oidc.Provider
	PKCESupported bool
}

func Discover(ctx context.Context, issuerURL string) (*DiscoveryResult, error) {
	parsed, err := ParseIssuerURL(issuerURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(oidc.ClientContext(ctx, discoveryHTTPClient()), discoveryTimeout)
	defer cancel()

	canonicalIssuer, err := fetchCanonicalIssuer(ctx, parsed)
	if err != nil {
		return nil, err
	}

	provider, err := oidc.NewProvider(ctx, canonicalIssuer)
	if err != nil {
		return nil, fmt.Errorf("could not read the OpenID configuration of %v: %w", canonicalIssuer, err)
	}

	var claims struct {
		CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
	}
	if err := provider.Claims(&claims); err != nil {
		return nil, fmt.Errorf("could not read the OpenID configuration of %v: %w", canonicalIssuer, err)
	}

	return &DiscoveryResult{
		Issuer:        canonicalIssuer,
		Provider:      provider,
		PKCESupported: containsFold(claims.CodeChallengeMethodsSupported, "S256"),
	}, nil
}

func ParseIssuerURL(issuerURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(issuerURL))
	if err != nil {
		return nil, fmt.Errorf("issuer is not a valid URL: %w", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && (scheme != "http" || !env.CustomOIDCAllowPrivateIssuers()) {
		return nil, fmt.Errorf("issuer must be an https URL")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("issuer must include a host")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("issuer must not contain credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("issuer must not contain a query string or fragment")
	}
	if err := validateNotEntraMultiTenant(parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func validateNotEntraMultiTenant(parsed *url.URL) error {
	if !strings.EqualFold(parsed.Hostname(), entraHost) {
		return nil
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) > 0 && containsFold(entraMultiTenantPaths, segments[0]) {
		return fmt.Errorf(
			"the Microsoft Entra ID endpoints %v cannot be used, because their OpenID configuration "+
				"does not name a tenant. Use the single-tenant issuer "+
				"https://login.microsoftonline.com/<tenant-id>/v2.0 instead",
			strings.Join(entraMultiTenantPaths, ", "),
		)
	}
	return nil
}

func fetchCanonicalIssuer(ctx context.Context, issuerURL *url.URL) (string, error) {
	discoveryURL := strings.TrimSuffix(issuerURL.String(), "/") + discoveryPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := discoveryHTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("could not read %v: %w", discoveryURL, err)
	}
	defer resp.Body.Close()
	if final := resp.Request.URL; !strings.EqualFold(final.Hostname(), issuerURL.Hostname()) {
		return "", fmt.Errorf("%v redirects to the different host %v", discoveryURL, final.Hostname())
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not read %v: %v", discoveryURL, resp.Status)
	}

	var document struct {
		Issuer string `json:"issuer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return "", fmt.Errorf("could not parse %v: %w", discoveryURL, err)
	}
	if document.Issuer == "" {
		return "", fmt.Errorf("%v does not state an issuer", discoveryURL)
	}

	canonical, err := ParseIssuerURL(document.Issuer)
	if err != nil {
		return "", fmt.Errorf("%v states an unusable issuer %q: %w", discoveryURL, document.Issuer, err)
	}
	if !strings.EqualFold(canonical.Hostname(), issuerURL.Hostname()) {
		return "", fmt.Errorf("%v states the issuer %q of a different host", discoveryURL, document.Issuer)
	}
	return document.Issuer, nil
}

func discoveryHTTPClient() *http.Client {
	dialer := &net.Dialer{Control: rejectPrivateAddress}
	return &http.Client{
		Timeout:   discoveryTimeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
	}
}

func rejectPrivateAddress(_, address string, _ syscall.RawConn) error {
	if env.CustomOIDCAllowPrivateIssuers() {
		return nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("could not parse address %v: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("could not parse address %v", address)
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return fmt.Errorf("the issuer resolves to the non-public address %v", ip)
	}
	return nil
}

func containsFold(values []string, search string) bool {
	for _, value := range values {
		if strings.EqualFold(value, search) {
			return true
		}
	}
	return false
}
