package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/distr-sh/distr/internal/env"
	"golang.org/x/oauth2"
)

const (
	discoveryPath    = "/.well-known/openid-configuration"
	discoveryTimeout = 10 * time.Second
	entraHost        = "login.microsoftonline.com"
)

var entraMultiTenantPaths = []string{"common", "organizations", "consumers"}

// supportedSigningAlgorithms are the algorithms go-oidc can verify an ID token with. The list of the
// discovery document is filtered by it, the same way go-oidc's own discovery does, so that an identity
// provider cannot name an algorithm that ends up being passed to the JOSE library unchecked.
var supportedSigningAlgorithms = []string{
	oidc.RS256, oidc.RS384, oidc.RS512,
	oidc.ES256, oidc.ES384, oidc.ES512,
	oidc.PS256, oidc.PS384, oidc.PS512,
	oidc.EdDSA,
}

type DiscoveryResult struct {
	Issuer        string
	Provider      *oidc.Provider
	PKCESupported bool
}

// discoveryDocument is the part of the OpenID configuration we act on. It is parsed by us instead of
// go-oidc's discovery, so that reading the issuer for the anti-collision checks and building the provider
// need only one request to the identity provider.
type discoveryDocument struct {
	oidc.ProviderConfig
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

func Discover(ctx context.Context, issuerURL string) (*DiscoveryResult, error) {
	parsed, err := ParseIssuerURL(issuerURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(RestrictedClientContext(ctx), discoveryTimeout)
	defer cancel()

	document, err := fetchDiscoveryDocument(ctx, parsed)
	if err != nil {
		return nil, err
	}

	return &DiscoveryResult{
		Issuer:        document.IssuerURL,
		Provider:      document.NewProvider(ctx),
		PKCESupported: containsFold(document.CodeChallengeMethodsSupported, "S256"),
	}, nil
}

func ParseIssuerURL(issuerURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(issuerURL))
	if err != nil {
		return nil, fmt.Errorf("issuer is not a valid URL: %w", err)
	}
	scheme := env.URLScheme(strings.ToLower(parsed.Scheme))
	if scheme != env.SchemeHTTPS && (scheme != env.SchemeHTTP || !nonPublicIssuersAllowed()) {
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

func fetchDiscoveryDocument(ctx context.Context, issuerURL *url.URL) (*discoveryDocument, error) {
	discoveryURL := strings.TrimSuffix(issuerURL.String(), "/") + discoveryPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := restrictedHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not read %v: %w", discoveryURL, err)
	}
	defer resp.Body.Close()
	final := resp.Request.URL
	if !strings.EqualFold(final.Hostname(), issuerURL.Hostname()) {
		return nil, fmt.Errorf("%v redirects to the different host %v", discoveryURL, final.Hostname())
	}
	if !strings.EqualFold(final.Scheme, issuerURL.Scheme) {
		return nil, fmt.Errorf("%v redirects to %v, which is not the scheme of the issuer",
			discoveryURL, final.Scheme)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not read %v: %v", discoveryURL, resp.Status)
	}

	var document discoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return nil, fmt.Errorf("could not parse %v: %w", discoveryURL, err)
	}
	if document.IssuerURL == "" {
		return nil, fmt.Errorf("%v does not state an issuer", discoveryURL)
	}
	if document.AuthURL == "" || document.TokenURL == "" || document.JWKSURL == "" {
		return nil, fmt.Errorf("%v does not state an authorization, token and jwks endpoint", discoveryURL)
	}

	canonical, err := ParseIssuerURL(document.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("%v states an unusable issuer %q: %w", discoveryURL, document.IssuerURL, err)
	}
	if !strings.EqualFold(canonical.Hostname(), issuerURL.Hostname()) {
		return nil, fmt.Errorf("%v states the issuer %q of a different host", discoveryURL, document.IssuerURL)
	}
	document.Algorithms = slices.DeleteFunc(document.Algorithms, func(algorithm string) bool {
		return !slices.Contains(supportedSigningAlgorithms, algorithm)
	})
	return &document, nil
}

// RestrictedClientContext returns a context whose OpenID Connect and OAuth2 requests all go through the
// client that refuses non-public addresses, so that neither the endpoints of a discovery document nor a
// redirect can be used to reach into the network the hub runs in.
func RestrictedClientContext(ctx context.Context) context.Context {
	client := restrictedHTTPClient()
	return context.WithValue(oidc.ClientContext(ctx, client), oauth2.HTTPClient, client)
}

func restrictedHTTPClient() *http.Client {
	dialer := &net.Dialer{Control: rejectPrivateAddress}
	return &http.Client{
		Timeout:   discoveryTimeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
	}
}

// nonPublicIssuersAllowed reports whether an issuer may be reached over http or at a non-public
// address. That is the case exactly on an instance which is not served over https itself: it is a
// local or internal installation, where the identity provider to develop or test against runs next
// to the hub. On every other instance the issuer is administrator input this server fetches, so both
// are refused - an operator with an identity provider inside their own network configures it as the
// instance-wide generic provider instead.
func nonPublicIssuersAllowed() bool {
	return env.HostScheme() != env.SchemeHTTPS
}

func rejectPrivateAddress(_, address string, _ syscall.RawConn) error {
	if nonPublicIssuersAllowed() {
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
