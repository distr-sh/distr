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
	// entraHost issues multi-tenant discovery documents whose issuer is the literal template
	// "https://login.microsoftonline.com/{tenantid}/v2.0", which can never be verified.
	entraHost = "login.microsoftonline.com"
)

// entraMultiTenantPaths are the Entra ID endpoints that are not bound to a single tenant.
var entraMultiTenantPaths = []string{"common", "organizations", "consumers"}

// DiscoveryResult is the outcome of resolving an issuer URL to an identity provider.
type DiscoveryResult struct {
	// Issuer is the canonical issuer as stated by the discovery document. It may differ from the
	// URL that was entered (Auth0 states a trailing slash, Entra ID a tenant GUID for a domain
	// name), and it is what has to be stored and used for verification.
	Issuer   string
	Provider *oidc.Provider
	// PKCESupported reports whether the provider announced S256 code challenges, so an
	// administrator does not have to know.
	PKCESupported bool
}

// Discover resolves an issuer URL entered by an organization administrator to its identity
// provider. The URL is attacker-controlled input that this server fetches, so it is validated
// before and after the request: only https, no query or fragment, no multi-tenant Entra endpoint,
// and (unless CUSTOM_OIDC_ALLOW_PRIVATE_ISSUERS is set) no private network target.
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

	// go-oidc verifies that the document's issuer matches the one it was asked for, which is why
	// the canonical value has to be passed here instead of the entered one.
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

// ParseIssuerURL validates an issuer URL as entered by an administrator. It does not contact the
// provider, so it can be used to reject input before any request is made.
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
	// Azure AD B2C identifies a user flow with a query parameter, which the discovery URL and the
	// issuer comparison cannot carry. Such a tenant needs a user-flow-specific issuer instead.
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

// fetchCanonicalIssuer reads the issuer the provider states for itself. The document has to be
// served by the host that was entered: otherwise a provider could claim an issuer belonging to
// somebody else, and identities of two organizations would collide on (issuer, subject).
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

// discoveryHTTPClient is the client used for every request to an organization-configured provider,
// including the JWKS fetch during token verification. Its dialer rejects private network targets
// after DNS resolution, which also covers redirects and DNS rebinding.
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
