// Package redact masks secrets in text that a customer environment sends to Distr, such as
// deployment logs and support bundle resources, before it is stored or shown to a vendor.
package redact

import (
	"regexp"
	"strings"
)

const Mask = "********"

// Every pattern is replaced with the template below, so a pattern keeps the text around the value
// it masks in the optional groups prefix and suffix.
const replacement = "${prefix}" + Mask + "${suffix}"

const (
	identifierChars = `[A-Za-z0-9_.-]*`
	credentialWords = `password|passwd|passphrase|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|credential`
	// sensitiveKey matches a key ending in a word that names a credential (DB_PASSWORD,
	// client_secret, AWS_SECRET_ACCESS_KEY). It must not match a key that merely refers to one,
	// such as secretName or passwordFile in a Kubernetes resource, or PWD in an environment dump.
	sensitiveKey = identifierChars + `(?:` + credentialWords + `|credentials)`
	// sensitiveName matches the name of an environment variable containing such a word anywhere.
	sensitiveName = identifierChars + `(?:` + credentialWords + `)` + identifierChars
)

var patterns = compile(
	// PEM private key block: -----BEGIN RSA PRIVATE KEY----- ... -----END RSA PRIVATE KEY-----
	`(?s)(?P<prefix>-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----).*?(?P<suffix>-----END [A-Z0-9 ]*PRIVATE KEY-----)`,
	// Credentials in a URL: postgres://user:password@host
	`(?i)(?P<prefix>\b[a-z][a-z0-9+.-]*://)[^\s:/@"'<>]+:[^\s/@"'<>]+(?P<suffix>@)`,
	// Kubernetes env entry in JSON: {"name":"DB_PASSWORD","value":"..."}
	`(?i)(?P<prefix>"name"\s*:\s*"`+sensitiveName+`"\s*,\s*"value"\s*:\s*")[^"]*(?P<suffix>")`,
	// Kubernetes env entry in YAML: "- name: DB_PASSWORD" followed by "value: ..." on the next line
	`(?im)(?P<prefix>^[ \t]*-?[ \t]*name:[ \t]*["']?`+sensitiveName+`["']?[ \t]*\r?\n[ \t]*value:[ \t]*)[^\s].*`,
	// Quoted value of a sensitive key: "client_secret": "...", password='...'. The optional
	// backslashes cover JSON that is itself embedded in a JSON string.
	`(?i)(?P<prefix>\b`+sensitiveKey+`\\?["']?[ \t]*[:=][ \t]*\\?["'])(?:[^"'\\\r\n]|\\[^"'])+(?P<suffix>\\?["'])`,
	// Unquoted value of a sensitive key: DB_PASSWORD=..., api_key: ...
	`(?i)(?P<prefix>\b`+sensitiveKey+`\\?["']?[ \t]*[:=][ \t]*)[^\s"'\\,;&{}\[\]]+`,
	// Password field of a connection string, which may contain spaces: Password=...;
	`(?i)(?P<prefix>\b(?:password|pwd)[ \t]*=[ \t]*)[^;\r\n]+(?P<suffix>;)`,
	// Authorization header: Authorization: Bearer ...
	`(?i)(?P<prefix>\bauthorization["']?[ \t]*[:=][ \t]*["']?(?:bearer|basic|token)?[ \t]*)[^\s"',;]+`,
	// Bearer token outside an Authorization header
	`(?P<prefix>\bBearer[ \t]+)[A-Za-z0-9._~+/-]{20,}=*`,
	// AWS access key ID
	`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`,
	// GitHub token
	`\bgh[pousr]_[A-Za-z0-9]{36,}`,
	// GitHub fine-grained personal access token
	`\bgithub_pat_[A-Za-z0-9_]{22,}`,
	// GitLab personal access token
	`\bglpat-[A-Za-z0-9_-]{20,}`,
	// Slack token
	`\bxox[abprs]-[A-Za-z0-9-]{10,}`,
	// Stripe secret or restricted key
	`\b[rs]k_(?:live|test)_[A-Za-z0-9]{16,}`,
	// Google API key
	`\bAIza[0-9A-Za-z_-]{35}`,
	// JWT
	`\beyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`,
	// Distr access token, current and legacy format
	`(?P<prefix>\bdistr-)(?:[0-9A-Za-z]{22}_[0-9A-Za-z]{49}|[0-9a-f]{32})\b`,
)

func compile(exprs ...string) []*regexp.Regexp {
	result := make([]*regexp.Regexp, len(exprs))
	for i, expr := range exprs {
		result[i] = regexp.MustCompile(expr)
	}
	return result
}

type Redactor struct {
	secrets *strings.Replacer
}

// New returns a Redactor that masks the given secret values verbatim in addition to the patterns
// of common credentials.
func New(secrets ...string) *Redactor {
	pairs := make([]string, 0, 2*len(secrets))
	for _, secret := range secrets {
		if secret != "" {
			pairs = append(pairs, secret, Mask)
		}
	}
	return &Redactor{secrets: strings.NewReplacer(pairs...)}
}

func (r *Redactor) Redact(s string) string {
	s = r.secrets.Replace(s)
	for _, pattern := range patterns {
		s = pattern.ReplaceAllString(s, replacement)
	}
	return s
}
