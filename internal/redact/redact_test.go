package redact_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/redact"
	. "github.com/onsi/gomega"
)

func TestRedact(t *testing.T) {
	g := NewWithT(t)
	r := redact.New("hunter2", "")

	for _, tc := range []struct{ input, expected string }{
		{"login with hunter2 failed", "login with ******** failed"},
		{"connecting to postgres://app:s3cr3t@db:5432", "connecting to postgres://********@db:5432"},
		{"connecting to redis://:s3cr3t@cache:6379", "connecting to redis://********@cache:6379"},
		{"level=info DB_PASSWORD=abc123 PORT=8080", "level=info DB_PASSWORD=******** PORT=8080"},
		{"HOME=/root\nDB_PASSWORD=abc,def ghi\nPORT=80", "HOME=/root\nDB_PASSWORD=********\nPORT=80"},
		{"export API_TOKEN=a;b\r\n", "export API_TOKEN=********\r\n"},
		{`DB_PASSWORD="a b"`, `DB_PASSWORD="********"`},
		{`{"client_secret": "a b c", "user": "x"}`, `{"client_secret": "********", "user": "x"}`},
		{`{"password":"head\"tailpiece"}`, `{"password":"********"}`},
		{`DB_PASSWORD="head'tailpiece"`, `DB_PASSWORD="********"`},
		{`password='head"tailpiece'`, `password='********'`},
		{`msg="{\"api_key\":\"k\\\\ey\"}"`, `msg="{\"api_key\":\"********\"}"`},
		{`msg="{\"api_key\":\"head\\\"tailpiece\"}"`, `msg="{\"api_key\":\"********\"}"`},
		{"Server=db;User ID=app;Password=p a;", "Server=db;User ID=app;Password=********;"},
		{"Server=db;Pwd=abc def", "Server=db;Pwd=********"},
		{"Server=db; Password=a b\nnext", "Server=db; Password=********\nnext"},
		{"Authorization: Bearer abc.def", "Authorization: Bearer ********"},
		{`{"name":"AWS_SECRET_ACCESS_KEY","value":"x"}`, `{"name":"AWS_SECRET_ACCESS_KEY","value":"********"}`},
		{`{"name":"DB_PASSWORD","value":"head\"tailpiece"}`, `{"name":"DB_PASSWORD","value":"********"}`},
		{"key AKIAABCDEFGHIJKLMNOP used", "key ******** used"},
		{"token ghp_0123456789abcdefghijklmnopqrstuvwxyz", "token ********"},
		{"jwt eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0In0.c2lnbmF0dXJlMTIz", "jwt ********"},
		{
			"distr-0123456789ABCDEFGHIJKL_0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLM",
			"distr-********",
		},
		{
			"env:\n  - name: GITHUB_TOKEN\n    value: abc def\n  - name: PORT\n    value: \"80\"\n",
			"env:\n  - name: GITHUB_TOKEN\n    value: ********\n  - name: PORT\n    value: \"80\"\n",
		},
		{
			"env:\n  - name: DB_PASSWORD\n    value: |-\n      line one\n\n      line two\n  - name: PORT\n    value: \"80\"\n",
			"env:\n  - name: DB_PASSWORD\n    value: |-\n      ********\n  - name: PORT\n    value: \"80\"\n",
		},
		{
			"- name: API_TOKEN\r\n  value: >2 # folded\r\n    abc\r\n    def",
			"- name: API_TOKEN\r\n  value: >2 # folded\r\n    ********",
		},
		{"- name: API_TOKEN\r\n  value: |\r\n    abc\r\n", "- name: API_TOKEN\r\n  value: |\r\n    ********\r\n"},
		{
			"-----BEGIN RSA PRIVATE KEY-----\nMIIE\nabcd\n-----END RSA PRIVATE KEY-----",
			"-----BEGIN RSA PRIVATE KEY-----********-----END RSA PRIVATE KEY-----",
		},
	} {
		g.Expect(r.Redact(tc.input)).To(Equal(tc.expected), tc.input)
	}
}

func TestRedact_keepsReferencesToCredentials(t *testing.T) {
	g := NewWithT(t)
	r := redact.New()

	for _, input := range []string{
		"secretName: registry-credentials",
		"passwordFile: /run/secrets/db",
		"PWD=/home/app",
		"credentials:\n  username: app",
		"refreshing token",
		"tokens: 1500",
		"Basic authentication failed",
		"listening on http://localhost:8080/path",
	} {
		g.Expect(r.Redact(input)).To(Equal(input), input)
	}
}
