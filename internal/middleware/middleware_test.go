package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

type fakeAuthInfo struct {
	authinfo.AuthInfoWithUserAndOrganization
	org           *types.Organization
	customerOrgID *uuid.UUID
	isSuperAdmin  bool
}

func (f fakeAuthInfo) CurrentOrg() *types.Organization  { return f.org }
func (f fakeAuthInfo) CurrentCustomerOrgID() *uuid.UUID { return f.customerOrgID }
func (f fakeAuthInfo) CurrentPartnerOrgID() *uuid.UUID  { return nil }
func (f fakeAuthInfo) IsSuperAdmin() bool               { return f.isSuperAdmin }

type countingReader struct {
	io.Reader
	read int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.read += n
	return n, err
}

func TestRequestSize(t *testing.T) {
	const payload = "longer than the limit"
	var readErr error
	handler := RequestSize(16)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))

	t.Run("declared length is rejected without reading the body", func(t *testing.T) {
		g := NewWithT(t)
		body := &countingReader{Reader: strings.NewReader(payload)}
		req := httptest.NewRequest(http.MethodPut, "/", body)
		req.ContentLength = int64(len(payload))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		g.Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
		g.Expect(body.read).To(BeZero())
	})

	t.Run("undeclared length is cut off while reading", func(t *testing.T) {
		g := NewWithT(t)
		req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(payload))
		req.ContentLength = -1
		handler.ServeHTTP(httptest.NewRecorder(), req)
		_, ok := errors.AsType[*http.MaxBytesError](readErr)
		g.Expect(ok).To(BeTrue())
	})
}

func TestRequireActiveSubscription(t *testing.T) {
	expired := &types.Organization{
		SubscriptionType:   types.SubscriptionTypePro,
		SubscriptionEndsAt: time.Now().Add(-time.Hour),
	}
	serve := func(info fakeAuthInfo) *httptest.ResponseRecorder {
		handler := RequireActiveSubscription(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(auth.Authentication.NewContext(req.Context(), info))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	for _, tt := range []struct {
		name     string
		info     fakeAuthInfo
		expected int
		body     string
		hidden   string
	}{
		{
			name:     "active subscription passes",
			info:     fakeAuthInfo{org: &types.Organization{SubscriptionEndsAt: time.Now().Add(time.Hour)}},
			expected: http.StatusOK,
		},
		{
			name: "community never expires",
			info: fakeAuthInfo{org: &types.Organization{
				SubscriptionType:   types.SubscriptionTypeCommunity,
				SubscriptionEndsAt: time.Now().Add(-time.Hour),
			}},
			expected: http.StatusOK,
		},
		{
			name:     "vendor of an expired organization is told why",
			info:     fakeAuthInfo{org: expired},
			expected: http.StatusForbidden,
			body:     "subscription",
		},
		{
			name:     "customer of an expired organization is not told why",
			info:     fakeAuthInfo{org: expired, customerOrgID: new(uuid.New())},
			expected: http.StatusForbidden,
			body:     "unavailable",
			hidden:   "subscription",
		},
		{
			name:     "super admin passes an expired organization",
			info:     fakeAuthInfo{org: expired, isSuperAdmin: true},
			expected: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			w := serve(tt.info)
			g.Expect(w.Code).To(Equal(tt.expected))
			g.Expect(w.Body.String()).To(ContainSubstring(tt.body))
			if tt.hidden != "" {
				g.Expect(w.Body.String()).NotTo(ContainSubstring(tt.hidden))
			}
		})
	}
}
