package apitest_test

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func pullsWith(query url.Values) string {
	return apitest.ArtifactPulls + "?" + query.Encode()
}

func TestArtifactPulls(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	app := vendor.push(t, "app", "1.0.0", "app")
	tool := vendor.push(t, "tool", "1.0.0", "tool")
	customer := vendor.CreateCustomerOrganization("Customer")
	customerUser := vendor.Invite(types.UserRoleReadOnly, &customer)
	g := NewWithT(t)

	g.Expect(h.PullImage(vendor.key, "vendor/app", "1.0.0")).To(Succeed())
	g.Expect(h.PullImage(customerUser.CreateAccessToken(), "vendor/tool", "1.0.0")).To(Succeed())

	pulls := func(query url.Values) []api.ArtifactVersionPullResponse {
		t.Helper()
		return apitest.Expect[[]api.ArtifactVersionPullResponse](vendor.Get(pullsWith(query)), http.StatusOK)
	}
	artifactNames := func(pulls []api.ArtifactVersionPullResponse) []string {
		names := make([]string, len(pulls))
		for i, pull := range pulls {
			names[i] = pull.Artifact.Name
		}
		return names
	}
	g.Expect(artifactNames(pulls(url.Values{}))).To(ContainElements("app", "tool"))
	g.Expect(artifactNames(pulls(url.Values{"customerOrganizationId": {customer.String()}}))).To(HaveEach("tool"))
	g.Expect(artifactNames(pulls(url.Values{"artifactId": {app.String()}}))).To(HaveEach("app"))
	g.Expect(artifactNames(pulls(url.Values{"userAccountId": {customerUser.ID.String()}}))).To(HaveEach("tool"))
	g.Expect(pulls(url.Values{"customerOrganizationId": {customer.String()}})).
		To(HaveEach(HaveField("CustomerOrganizationName", HaveValue(Equal("Customer")))))
	g.Expect(pulls(url.Values{"anonymous": {"true"}})).To(BeEmpty())
	g.Expect(pulls(url.Values{"before": {time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)}})).To(BeEmpty())
	g.Expect(pulls(url.Values{"count": {"1"}})).To(HaveLen(1))

	options := apitest.Expect[api.ArtifactVersionPullFilterOptions](vendor.Get(apitest.ArtifactPulls+"/filter-options"),
		http.StatusOK)
	g.Expect(options.Artifacts).To(ConsistOf(
		api.ArtifactPullFilterOption{ID: app, Name: "app"},
		api.ArtifactPullFilterOption{ID: tool, Name: "tool"},
	))
	g.Expect(options.CustomerOrganizations).To(Equal([]api.ArtifactPullFilterOption{{ID: customer, Name: "Customer"}}))
	versionOptions := apitest.Expect[[]api.ArtifactPullFilterOption](
		vendor.Get(apitest.ArtifactPulls+"/filter-options/versions?artifactId="+app.String()), http.StatusOK)
	g.Expect(versionOptions).To(HaveExactElements(HaveField("Name", "1.0.0")))

	export := vendor.Get(apitest.ArtifactPulls + "/export?customerOrganizationId=" + customer.String())
	export.ExpectStatus(http.StatusOK)
	g.Expect(export.Header.Get("Content-Type")).To(HavePrefix("text/csv"))
	rows, err := csv.NewReader(strings.NewReader(string(export.Body))).ReadAll()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rows).To(HaveLen(len(pulls(url.Values{"customerOrganizationId": {customer.String()}}))+1),
		"the export has a header and the same rows as the list")
	g.Expect(rows[0][0]).To(Equal("Date"))
	g.Expect(rows[1]).To(ContainElements("Customer", "tool", "1.0.0"))
}

func TestArtifactPullErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	vendor.push(t, "app", "1.0.0", "app")
	g := NewWithT(t)

	query := func(name, key, value string) request {
		return request{name, http.MethodGet, pullsWith(url.Values{key: {value}}), nil}
	}
	requests := []request{
		query("malformed before", "before", "yesterday"),
		query("malformed after", "after", "tomorrow"),
		query("malformed count", "count", "many"),
		query("count of zero", "count", "0"),
		query("count above the maximum", "count", "1001"),
		query("malformed customer", "customerOrganizationId", "nope"),
		query("malformed user", "userAccountId", "nope"),
		query("malformed anonymous", "anonymous", "maybe"),
		query("malformed artifact", "artifactId", "nope"),
		query("malformed version", "artifactVersionId", "nope"),
		query("malformed deployment target", "deploymentTargetId", "nope"),
		{"export with a malformed filter", http.MethodGet, apitest.ArtifactPulls + "/export?count=many", nil},
		{"version options without artifact", http.MethodGet, apitest.ArtifactPulls + "/filter-options/versions", nil},
		{
			"version options of a malformed artifact", http.MethodGet,
			apitest.ArtifactPulls + "/filter-options/versions?artifactId=nope", nil,
		},
	}
	g.Expect(statuses(vendor.Client, requests)).To(Equal(allWithStatus(requests, http.StatusBadRequest)))
}

func TestArtifactPullAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	app := vendor.push(t, "app", "1.0.0", "app")
	customer := vendor.CreateCustomerOrganization("Customer")
	customerUser := vendor.Invite(types.UserRoleAdmin, &customer)
	g := NewWithT(t)
	g.Expect(h.PullImage(customerUser.CreateAccessToken(), "vendor/app", "1.0.0")).To(Succeed())

	versionOptions := apitest.ArtifactPulls + "/filter-options/versions?artifactId=" + app.String()
	requests := []request{
		{"list", http.MethodGet, apitest.ArtifactPulls, nil},
		{"filter options", http.MethodGet, apitest.ArtifactPulls + "/filter-options", nil},
		{"version options", http.MethodGet, versionOptions, nil},
		{"export", http.MethodGet, apitest.ArtifactPulls + "/export", nil},
	}
	g.Expect(statuses(customerUser.Client, requests)).To(Equal(allWithStatus(requests, http.StatusForbidden)))
	g.Expect(apitest.Expect[[]api.ArtifactVersionPullResponse](
		vendor.Invite(types.UserRoleReadOnly, nil).Get(apitest.ArtifactPulls), http.StatusOK)).NotTo(BeEmpty())

	other := h.RegisterVendor()
	g.Expect(apitest.Expect[[]api.ArtifactVersionPullResponse](other.Get(apitest.ArtifactPulls), http.StatusOK)).
		To(BeEmpty())
	g.Expect(apitest.Expect[[]api.ArtifactVersionPullResponse](
		other.Get(pullsWith(url.Values{"customerOrganizationId": {customer.String()}})), http.StatusOK)).To(BeEmpty())
	g.Expect(apitest.Expect[[]api.ArtifactPullFilterOption](other.Get(versionOptions), http.StatusOK)).To(BeEmpty())
	options := apitest.Expect[api.ArtifactVersionPullFilterOptions](other.Get(apitest.ArtifactPulls+"/filter-options"),
		http.StatusOK)
	g.Expect(options.Artifacts).To(BeEmpty())
	g.Expect(options.CustomerOrganizations).To(BeEmpty())
	g.Expect(options.UserAccounts).To(BeEmpty())
	rows, err := csv.NewReader(strings.NewReader(string(
		other.Get(apitest.ArtifactPulls + "/export").ExpectStatus(http.StatusOK).Body))).ReadAll()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rows).To(HaveLen(1), "only the header")
}
