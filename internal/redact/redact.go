// Package redact masks secrets in text that a customer environment sends to Distr, such as
// deployment logs and support bundle resources, before it is stored or shown to a vendor.
package redact

import (
	"regexp"
	"slices"
	"strings"
)

const Mask = "********"

// Every pattern is replaced with the template below, so a pattern keeps the text around the value
// it masks in the optional groups prefix and suffix.
const replacement = "${prefix}" + Mask + "${suffix}"

const (
	identifierChars = `[A-Za-z0-9_.-]*`
	// jsonStringContent matches the content of a double-quoted string up to the closing quote,
	// consuming escape sequences so that an escaped quote does not end it early.
	jsonStringContent = `(?:[^"\\\r\n]|\\.)+`
	// embeddedJSONStringContent is jsonStringContent for a string whose quotes are escaped once
	// more, so an escape sequence inside it starts with two backslashes and \" ends it.
	embeddedJSONStringContent = `(?:[^"\\\r\n]|\\\\(?:\\.|[^\\]))+`
)

// credentialWords are the lowercase words that name a credential. They are the hints of every
// pattern built from sensitiveKey or sensitiveName, which match them literally.
var credentialWords = []string{
	"password", "passwd", "passphrase", "secret", "token", "api_key", "api-key", "apikey",
	"access_key", "access-key", "accesskey", "private_key", "private-key", "privatekey", "credential",
}

var (
	credentialWordsExpr = alternation(credentialWords)
	// sensitiveKey matches a key ending in a word that names a credential (DB_PASSWORD,
	// client_secret, AWS_SECRET_ACCESS_KEY). It must not match a key that merely refers to one,
	// such as secretName or passwordFile in a Kubernetes resource, or PWD in an environment dump.
	sensitiveKey = identifierChars + `(?:` + credentialWordsExpr + `|credentials)`
	// sensitiveName matches the name of an environment variable containing such a word anywhere.
	sensitiveName = identifierChars + `(?:` + credentialWordsExpr + `)` + identifierChars
)

var patterns = []pattern{
	// PEM private key block: -----BEGIN RSA PRIVATE KEY----- ... -----END RSA PRIVATE KEY-----
	hinted(
		`(?s)(?P<prefix>-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----).*?(?P<suffix>-----END [A-Z0-9 ]*PRIVATE KEY-----)`,
		"private key-----",
	),
	// Credentials in a URL: postgres://user:password@host
	hinted(`(?i)(?P<prefix>\b[a-z][a-z0-9+.-]*://)[^\s:/@"'<>]*:[^\s/@"'<>]+(?P<suffix>@)`, "://"),
	// Kubernetes env entry in JSON: {"name":"DB_PASSWORD","value":"..."}
	hinted(
		`(?i)(?P<prefix>"name"\s*:\s*"`+sensitiveName+`"\s*,\s*"value"\s*:\s*")`+jsonStringContent+`(?P<suffix>")`,
		credentialWords...,
	),
	// Kubernetes env entry in YAML: "- name: DB_PASSWORD" followed by "value: ..." on the next line.
	// A block scalar (value: |) is left to yamlBlockScalarValues.
	hinted(
		`(?im)(?P<prefix>^[ \t]*-?[ \t]*name:[ \t]*["']?`+sensitiveName+`["']?[ \t]*\r?\n[ \t]*value:[ \t]*)[^\s|>].*`,
		credentialWords...,
	),
	// Double-quoted value of a sensitive key: "client_secret": "...", DB_PASSWORD="..."
	hinted(
		`(?i)(?P<prefix>\b`+sensitiveKey+`["']?[ \t]*[:=][ \t]*")`+jsonStringContent+`(?P<suffix>")`,
		credentialWords...,
	),
	// Single-quoted value of a sensitive key: password='...'
	hinted(`(?i)(?P<prefix>\b`+sensitiveKey+`["']?[ \t]*[:=][ \t]*')[^'\r\n]+(?P<suffix>')`, credentialWords...),
	// Value of a sensitive key in JSON that is itself embedded in a JSON string: {\"api_key\":\"...\"}
	hinted(
		`(?i)(?P<prefix>\b`+sensitiveKey+`\\"[ \t]*:[ \t]*\\")`+embeddedJSONStringContent+`(?P<suffix>\\")`,
		credentialWords...,
	),
	// Line of an environment dump or .env file assigning a sensitive variable, whose unquoted value
	// runs to the end of the line: DB_PASSWORD=..., export API_TOKEN=...
	hinted(`(?im)^(?P<prefix>[ \t]*(?:export[ \t]+)?`+sensitiveKey+`=)[^"'\r\n][^\r\n]*`, credentialWords...),
	// Unquoted value of a sensitive key: DB_PASSWORD=..., api_key: .... A YAML block scalar
	// (password: |) is left to yamlBlockScalarValues.
	hinted(
		`(?i)(?P<prefix>\b`+sensitiveKey+`\\?["']?[ \t]*[:=][ \t]*)[^\s"'\\,;&{}\[\]|>][^\s"'\\,;&{}\[\]]*`,
		credentialWords...,
	),
	// Password field of a connection string, which may contain spaces: Password=...;
	hinted(`(?i)(?P<prefix>\b(?:password|pwd)[ \t]*=[ \t]*)[^;\r\n]+(?P<suffix>;)`, "password", "pwd"),
	// Password field after another field of a connection string, where the trailing semicolon is
	// optional: Server=db;Pwd=.... Without a preceding field, Pwd=... is the shell's PWD variable.
	hinted(`(?i)(?P<prefix>;[ \t]*(?:password|pwd)[ \t]*=[ \t]*)[^;\r\n]+`, "password", "pwd"),
	// Authorization header: Authorization: Bearer ...
	hinted(
		`(?i)(?P<prefix>\bauthorization["']?[ \t]*[:=][ \t]*["']?(?:bearer|basic|token)?[ \t]*)[^\s"',;]+`,
		"authorization",
	),
	// Bearer token outside an Authorization header
	hinted(`(?P<prefix>\bBearer[ \t]+)[A-Za-z0-9._~+/-]{20,}=*`, "bearer"),
	// AWS access key ID
	hinted(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`, "akia", "asia"),
	// GitHub token
	hinted(`\bgh[pousr]_[A-Za-z0-9]{36,}`, "ghp_", "gho_", "ghu_", "ghs_", "ghr_"),
	// GitHub fine-grained personal access token
	hinted(`\bgithub_pat_[A-Za-z0-9_]{22,}`, "github_pat_"),
	// GitLab personal access token
	hinted(`\bglpat-[A-Za-z0-9_-]{20,}`, "glpat-"),
	// Slack token
	hinted(`\bxox[abprs]-[A-Za-z0-9-]{10,}`, "xoxa-", "xoxb-", "xoxp-", "xoxr-", "xoxs-"),
	// Stripe secret or restricted key
	hinted(`\b[rs]k_(?:live|test)_[A-Za-z0-9]{16,}`, "k_live_", "k_test_"),
	// Google API key
	hinted(`\bAIza[0-9A-Za-z_-]{35}`, "aiza"),
	// JWT
	hinted(`\beyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`, "eyj"),
	// Distr access token, current and legacy format
	hinted(`(?P<prefix>\bdistr-)(?:[0-9A-Za-z]{22}_[0-9A-Za-z]{49}|[0-9a-f]{32})\b`, "distr-"),
}

// pattern is a regular expression that only matches an input whose lowercase form contains one of
// hints, so that the expression, which costs several microseconds per log line even when it finds
// nothing, only runs on the few lines that may contain a credential. A pattern without hints runs
// on every input. strings.ToLower keeps ſ, which (?i) matches with s, so a key spelled with it is
// not redacted.
type pattern struct {
	expr  *regexp.Regexp
	hints []string
}

func hinted(expr string, hints ...string) pattern {
	return pattern{expr: regexp.MustCompile(expr), hints: hints}
}

func (p pattern) mayMatch(lower string) bool {
	return len(p.hints) == 0 || slices.ContainsFunc(p.hints, func(hint string) bool {
		return strings.Contains(lower, hint)
	})
}

func alternation(words []string) string {
	quoted := make([]string, len(words))
	for i, word := range words {
		quoted[i] = regexp.QuoteMeta(word)
	}
	return strings.Join(quoted, "|")
}

type Redactor struct {
	secrets  []string
	replacer *strings.Replacer
}

// New returns a Redactor that masks the given secret values verbatim in addition to the patterns
// of common credentials.
func New(secrets ...string) *Redactor {
	r := &Redactor{secrets: make([]string, 0, len(secrets))}
	pairs := make([]string, 0, 2*len(secrets))
	for _, secret := range secrets {
		if secret != "" {
			r.secrets = append(r.secrets, secret)
			pairs = append(pairs, secret, Mask)
		}
	}
	r.replacer = strings.NewReplacer(pairs...)
	return r
}

// Redact returns s with every credential masked. Most inputs contain none, so it copies s only
// where something has to be masked: a strings.Replacer and ReplaceAllString copy their whole input
// even when they replace nothing.
func (r *Redactor) Redact(s string) string {
	if slices.ContainsFunc(r.secrets, func(secret string) bool { return strings.Contains(s, secret) }) {
		s = r.replacer.Replace(s)
	}
	lower := strings.ToLower(s)
	for _, p := range yamlBlockScalarValues {
		if p.mayMatch(lower) {
			if redacted := redactYAMLBlockScalar(s, p.expr); redacted != s {
				s, lower = redacted, strings.ToLower(redacted)
			}
		}
	}
	for _, p := range patterns {
		if p.mayMatch(lower) && p.expr.MatchString(s) {
			s = p.expr.ReplaceAllString(s, replacement)
			lower = strings.ToLower(s)
		}
	}
	return s
}

const yamlBlockScalarIndicator = `[|>][0-9+-]*[ \t]*(?:#.*)?\r?$`

// yamlBlockScalarValues match a sensitive YAML value that is a literal or folded block scalar, up to
// the end of the line that opens it. The indent group is the indentation that the content of the
// block scalar is deeper than.
var yamlBlockScalarValues = []pattern{
	// Kubernetes env entry: "- name: DB_PASSWORD" followed by "value: |-" on the next line
	hinted(
		`(?im)^[ \t]*-?[ \t]*name:[ \t]*["']?`+sensitiveName+`["']?[ \t]*\r?\n(?P<indent>[ \t]*)value:[ \t]*`+
			yamlBlockScalarIndicator,
		credentialWords...,
	),
	// Sensitive key: "password: |-", "- client_secret: >"
	hinted(
		`(?im)^(?P<indent>[ \t]*(?:-[ \t]+)?)["']?`+sensitiveKey+`["']?[ \t]*:[ \t]*`+yamlBlockScalarIndicator,
		credentialWords...,
	),
}

// redactYAMLBlockScalar replaces the content of every block scalar matched by pattern with a single
// masked line. The content is every following line indented deeper than the indent group, which a
// regular expression cannot tell apart from the next entry.
func redactYAMLBlockScalar(s string, pattern *regexp.Regexp) string {
	matches := pattern.FindAllStringSubmatchIndex(s, -1)
	if matches == nil {
		return s
	}
	indentGroup := pattern.SubexpIndex("indent")

	var b strings.Builder
	last := 0
	for _, match := range matches {
		if match[0] < last || match[1] == len(s) {
			continue
		}
		valueIndent := match[2*indentGroup+1] - match[2*indentGroup]
		contentStart := match[1] + 1
		contentEnd, contentIndent := -1, 0
		for pos := contentStart; pos < len(s); {
			lineEnd := strings.IndexByte(s[pos:], '\n')
			if lineEnd < 0 {
				lineEnd = len(s)
			} else {
				lineEnd += pos
			}
			line := strings.TrimRight(s[pos:lineEnd], "\r")
			if content := strings.TrimLeft(line, " \t"); content != "" {
				indent := len(line) - len(content)
				if indent <= valueIndent {
					break
				}
				if contentEnd < 0 {
					contentIndent = indent
				}
				contentEnd = pos + len(line)
			}
			pos = lineEnd + 1
		}
		if contentEnd < 0 {
			continue
		}
		b.WriteString(s[last:contentStart])
		b.WriteString(strings.Repeat(" ", contentIndent))
		b.WriteString(Mask)
		last = contentEnd
	}
	b.WriteString(s[last:])
	return b.String()
}
