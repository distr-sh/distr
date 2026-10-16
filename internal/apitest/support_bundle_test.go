package apitest_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

const (
	supportBundleConfiguration = apitest.SupportBundles + "/configuration"
	supportBundleScripts       = supportBundleConfiguration + "/scripts"
)

// supportBundleFixture is a vendor with one customer whose admin opened a bundle.
type supportBundleFixture struct {
	vendor, customer *apitest.User
	customerOrg      uuid.UUID
	bundle           api.CreateSupportBundleResponse
}

func newSupportBundleFixture(h *apitest.Harness) supportBundleFixture {
	vendor := h.RegisterVendor()
	customerOrg := vendor.CreateCustomerOrganization("Acme")
	customer := vendor.Invite(types.UserRoleAdmin, &customerOrg)
	return supportBundleFixture{
		vendor:      vendor,
		customer:    customer,
		customerOrg: customerOrg,
		bundle:      createSupportBundle(customer, "Pods crash"),
	}
}

func createSupportBundle(customer *apitest.User, title string) api.CreateSupportBundleResponse {
	return apitest.Expect[api.CreateSupportBundleResponse](customer.Post(apitest.SupportBundles,
		api.CreateSupportBundleRequest{Title: title}), http.StatusOK)
}

func bundlePath(id uuid.UUID) string {
	return apitest.Path(apitest.SupportBundles, id)
}

// collectPath is the path of the collect script in command, with the bundle secret in its query.
func collectPath(command string) string {
	start := strings.Index(command, apitest.SupportBundleCollect)
	end := strings.Index(command[start:], "'")
	return command[start : start+end]
}

func collectRoute(id uuid.UUID, secret, route string) string {
	return apitest.SupportBundleCollect + "/" + id.String() + "/" + route + "?bundleSecret=" + secret
}

func bundleSecret(command string) string {
	_, secret, _ := strings.Cut(collectPath(command), "bundleSecret=")
	return secret
}

func resourceUpload(name, content string) apitest.Multipart {
	return apitest.Multipart{
		Values:          map[string]string{"name": name},
		Files:           map[string]string{"content": content},
		FileContentType: "text/plain",
	}
}

func getSupportBundle(client *apitest.Client, id uuid.UUID) api.SupportBundleDetail {
	return apitest.Expect[api.SupportBundleDetail](client.Get(bundlePath(id)), http.StatusOK)
}

func setBundleStatus(status string) api.UpdateSupportBundleStatusRequest {
	return api.UpdateSupportBundleStatusRequest{Status: status}
}

func TestSupportBundleLifecycle(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	f := newSupportBundleFixture(h)
	id := f.bundle.ID
	secret := bundleSecret(f.bundle.CollectCommand)
	collect := h.Anonymous()
	g := NewWithT(t)

	g.Expect(f.bundle.Status).To(Equal(string(types.SupportBundleStatusInitialized)))
	g.Expect(f.bundle.CustomerOrganizationID).To(Equal(f.customerOrg))
	g.Expect(f.bundle.CreatedByUserAccountID).To(HaveValue(Equal(f.customer.ID)))
	g.Expect(f.bundle.CollectCommand).To(HavePrefix("curl -fsSL '"))
	g.Expect(f.bundle.CollectCommand).To(HaveSuffix("' | sh"))
	g.Expect(secret).NotTo(BeEmpty())
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT bundle_secret IS NULL AND bundle_secret_enc IS NOT NULL FROM SupportBundle WHERE id = $1`, id)).
		To(BeTrue(), "the secret is stored encrypted only")
	g.Expect(getSupportBundle(f.vendor.Client, id).CollectCommand).To(HaveValue(Equal(f.bundle.CollectCommand)),
		"the vendor can hand the command on while the bundle collects")

	script := collect.Get(collectPath(f.bundle.CollectCommand)).ExpectStatus(http.StatusOK)
	g.Expect(script.Header.Get("Content-Type")).To(HavePrefix("text/plain"))
	g.Expect(string(script.Body)).To(ContainSubstring(secret))
	g.Expect(string(script.Body)).To(ContainSubstring(id.String()))

	logs := apitest.Expect[api.SupportBundleResourceSummary](
		collect.Post(collectRoute(id, secret, "resources"), resourceUpload(" logs ", "line 1\n")), http.StatusOK)
	g.Expect(logs.Name).To(Equal("logs"), "the multipart name is trimmed")
	collect.Post(collectRoute(id, secret, "resources"), resourceUpload("logs", "line 2\n")).
		ExpectStatus(http.StatusOK)
	collect.Post(collectRoute(id, secret, "resources"), resourceUpload("kube/pods", "pods\n")).
		ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT bool_and(content IS NULL AND content_enc IS NOT NULL) FROM SupportBundleResource`)).
		To(BeTrue(), "resource contents are stored encrypted only")

	collect.Post(collectRoute(id, secret, "finalize"), nil).ExpectStatus(http.StatusNoContent)
	detail := getSupportBundle(f.vendor.Client, id)
	g.Expect(detail.Status).To(Equal(string(types.SupportBundleStatusCreated)))
	g.Expect(detail.ResourceCount).To(BeEquivalentTo(3))
	g.Expect(detail.CollectCommand).To(BeNil())
	g.Expect(detail.Resources).To(HaveLen(3))
	g.Expect(detail.Resources[0].Content).To(Equal("line 1\n"))
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT bundle_secret_expires_at IS NULL FROM SupportBundle WHERE id = $1`, id)).To(BeTrue())
	g.Expect(statuses(collect, []request{
		{"script after finalize", http.MethodGet, collectRoute(id, secret, "collect-script"), nil},
		{"upload after finalize", http.MethodPost, collectRoute(id, secret, "resources"), resourceUpload("late", "x")},
		{"finalize twice", http.MethodPost, collectRoute(id, secret, "finalize"), nil},
	})).To(Equal(map[string]int{
		"script after finalize": http.StatusUnauthorized,
		"upload after finalize": http.StatusUnauthorized,
		"finalize twice":        http.StatusUnauthorized,
	}))

	download := f.vendor.Get(bundlePath(id) + "/download").ExpectStatus(http.StatusOK)
	g.Expect(download.Header.Get("Content-Type")).To(Equal("application/zip"))
	g.Expect(download.Header.Get("Content-Disposition")).
		To(Equal(`attachment; filename="distr-support-bundle-acme-podscrash-` + id.String()[:8] + `.zip"`))
	archive, err := zip.NewReader(bytes.NewReader(download.Body), int64(len(download.Body)))
	g.Expect(err).NotTo(HaveOccurred())
	entries := map[string]string{}
	for _, file := range archive.File {
		reader, err := file.Open()
		g.Expect(err).NotTo(HaveOccurred())
		content, err := io.ReadAll(reader)
		g.Expect(err).NotTo(HaveOccurred())
		entries[file.Name] = string(content)
	}
	g.Expect(entries).To(Equal(map[string]string{
		"logs.txt":      "line 1\n",
		"logs-2.txt":    "line 2\n",
		"kube-pods.txt": "pods\n",
	}), "entry names cannot escape the archive or collide")

	comment := apitest.Expect[api.SupportBundleComment](f.customer.Post(bundlePath(id)+"/comments",
		api.CreateSupportBundleCommentRequest{Content: "still crashing"}), http.StatusOK)
	g.Expect(comment.UserAccountID).To(HaveValue(Equal(f.customer.ID)))
	f.vendor.Post(bundlePath(id)+"/comments", api.CreateSupportBundleCommentRequest{Content: "looking into it"}).
		ExpectStatus(http.StatusOK)
	detail = getSupportBundle(f.customer.Client, id)
	g.Expect(detail.CommentCount).To(BeEquivalentTo(2))
	g.Expect(detail.Comments).To(HaveLen(2))
	g.Expect(detail.LastCommentAt).NotTo(BeNil())

	g.Expect(statuses(f.customer.Client, []request{
		{"customer resolves", http.MethodPatch, bundlePath(id) + "/status", setBundleStatus("resolved")},
	})).To(Equal(map[string]int{"customer resolves": http.StatusForbidden}))
	f.vendor.Patch(bundlePath(id)+"/status", setBundleStatus("canceled")).ExpectStatus(http.StatusBadRequest)
	f.vendor.Patch(bundlePath(id)+"/status", setBundleStatus("resolved")).ExpectStatus(http.StatusNoContent)
	detail = getSupportBundle(f.customer.Client, id)
	g.Expect(detail.Status).To(Equal(string(types.SupportBundleStatusResolved)))
	g.Expect(detail.StatusChangedByUserName).To(HaveValue(Equal("Vendor")))
	g.Expect(detail.StatusChangedAt).NotTo(BeNil())
	f.vendor.Patch(bundlePath(id)+"/status", setBundleStatus("resolved")).ExpectStatus(http.StatusBadRequest)

	f.vendor.Delete(bundlePath(id)).ExpectStatus(http.StatusNoContent)
	f.vendor.Get(bundlePath(id)).ExpectStatus(http.StatusNotFound)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM SupportBundleResource`)).To(BeZero())
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM SupportBundleComment`)).To(BeZero())
}

func TestSupportBundleCancelAndResolveWhileCollecting(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	f := newSupportBundleFixture(h)
	resolved := createSupportBundle(f.customer, "resolved while collecting")
	g := NewWithT(t)

	f.customer.Patch(bundlePath(f.bundle.ID)+"/status", setBundleStatus("canceled")).
		ExpectStatus(http.StatusNoContent)
	f.vendor.Patch(bundlePath(resolved.ID)+"/status", setBundleStatus("resolved")).
		ExpectStatus(http.StatusNoContent)
	for _, bundle := range []api.CreateSupportBundleResponse{f.bundle, resolved} {
		h.Anonymous().Get(collectPath(bundle.CollectCommand)).ExpectStatus(http.StatusUnauthorized)
		g.Expect(getSupportBundle(f.vendor.Client, bundle.ID).CollectCommand).To(BeNil())
	}
	g.Expect(getSupportBundle(f.vendor.Client, f.bundle.ID).Status).
		To(Equal(string(types.SupportBundleStatusCanceled)))
	g.Expect(statuses(f.vendor.Client, []request{
		{"cancel twice", http.MethodPatch, bundlePath(f.bundle.ID) + "/status", setBundleStatus("canceled")},
		{"resolve canceled", http.MethodPatch, bundlePath(f.bundle.ID) + "/status", setBundleStatus("resolved")},
	})).To(Equal(map[string]int{
		"cancel twice":     http.StatusBadRequest,
		"resolve canceled": http.StatusBadRequest,
	}))
}

func TestSupportBundleCollectAuthentication(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	f := newSupportBundleFixture(h)
	id := f.bundle.ID
	secret := bundleSecret(f.bundle.CollectCommand)
	other := createSupportBundle(f.customer, "other")
	otherSecret := bundleSecret(other.CollectCommand)
	expired := createSupportBundle(f.customer, "expired")
	h.Exec(`UPDATE SupportBundle SET bundle_secret_expires_at = now() - interval '1 minute' WHERE id = $1`, expired.ID)
	g := NewWithT(t)

	routes := []string{"collect-script", "resources", "finalize"}
	requests := make([]request, 0, 5*len(routes))
	withUserToken := make([]request, 0, len(routes))
	for _, route := range routes {
		method := http.MethodPost
		var body any = resourceUpload("logs", "x")
		if route == "collect-script" {
			method, body = http.MethodGet, nil
		}
		base := apitest.SupportBundleCollect + "/" + id.String() + "/" + route
		requests = append(requests,
			request{route + " without secret", method, base, body},
			request{route + " with a wrong secret", method, collectRoute(id, "wrong", route), body},
			request{route + " with the secret of another bundle", method, collectRoute(id, otherSecret, route), body},
			request{
				route + " of a malformed bundle id", method,
				apitest.SupportBundleCollect + "/nope/" + route + "?bundleSecret=" + secret, body,
			},
			request{
				route + " with an expired secret", method,
				collectRoute(expired.ID, bundleSecret(expired.CollectCommand), route), body,
			},
		)
		withUserToken = append(withUserToken, request{route, method, base, body})
	}
	g.Expect(statuses(h.Anonymous(), requests)).To(Equal(allWithStatus(requests, http.StatusUnauthorized)))
	g.Expect(statuses(f.customer.Client, withUserToken)).
		To(Equal(allWithStatus(withUserToken, http.StatusUnauthorized)))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM SupportBundleResource`)).To(BeZero())
	g.Expect(apitest.QueryValue[int](h,
		`SELECT count(*) FROM SupportBundle WHERE status = 'initialized'`)).To(Equal(3))

	g.Expect(statuses(h.Anonymous(), []request{
		{"upload without name", http.MethodPost, collectRoute(id, secret, "resources"), resourceUpload(" ", "x")},
		{
			"upload without content", http.MethodPost, collectRoute(id, secret, "resources"),
			apitest.Multipart{Values: map[string]string{"name": "logs"}},
		},
		{"upload as JSON", http.MethodPost, collectRoute(id, secret, "resources"), object{"name": "logs"}},
	})).To(Equal(map[string]int{
		"upload without name":    http.StatusBadRequest,
		"upload without content": http.StatusBadRequest,
		"upload as JSON":         http.StatusRequestEntityTooLarge,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM SupportBundleResource`)).To(BeZero())
}

func TestSupportBundleErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	f := newSupportBundleFixture(h)
	path := bundlePath(f.bundle.ID)
	unknown := bundlePath(uuid.New())
	g := NewWithT(t)

	g.Expect(statuses(f.customer.Client, []request{
		{"no title", http.MethodPost, apitest.SupportBundles, api.CreateSupportBundleRequest{}},
		{"blank title", http.MethodPost, apitest.SupportBundles, api.CreateSupportBundleRequest{Title: "  "}},
		{"malformed body", http.MethodPost, apitest.SupportBundles, "{"},
		{"empty comment", http.MethodPost, path + "/comments", api.CreateSupportBundleCommentRequest{Content: " "}},
		{"comment on unknown", http.MethodPost, unknown + "/comments", api.CreateSupportBundleCommentRequest{Content: "x"}},
		{"get malformed id", http.MethodGet, apitest.SupportBundles + "/nope", nil},
		{"get unknown", http.MethodGet, unknown, nil},
		{"download unknown", http.MethodGet, unknown + "/download", nil},
		{"delete unknown", http.MethodDelete, unknown, nil},
		{"unknown status", http.MethodPatch, path + "/status", setBundleStatus("created")},
		{"empty status", http.MethodPatch, path + "/status", setBundleStatus("")},
		{"cancel unknown", http.MethodPatch, unknown + "/status", setBundleStatus("canceled")},
	})).To(Equal(map[string]int{
		"no title":           http.StatusBadRequest,
		"blank title":        http.StatusBadRequest,
		"malformed body":     http.StatusBadRequest,
		"empty comment":      http.StatusBadRequest,
		"comment on unknown": http.StatusNotFound,
		"get malformed id":   http.StatusNotFound,
		"get unknown":        http.StatusNotFound,
		"download unknown":   http.StatusNotFound,
		"delete unknown":     http.StatusNotFound,
		"unknown status":     http.StatusBadRequest,
		"empty status":       http.StatusBadRequest,
		"cancel unknown":     http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM SupportBundle`)).To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM SupportBundleComment`)).To(BeZero())
	g.Expect(getSupportBundle(f.vendor.Client, f.bundle.ID).Status).
		To(Equal(string(types.SupportBundleStatusInitialized)))
}

func TestSupportBundleAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	f := newSupportBundleFixture(h)
	path := bundlePath(f.bundle.ID)
	colleague := f.vendor.Invite(types.UserRoleReadOnly, &f.customerOrg)
	reader := f.vendor.Invite(types.UserRoleReadOnly, nil)
	otherCustomerOrg := f.vendor.CreateCustomerOrganization("Other")
	otherCustomer := f.vendor.Invite(types.UserRoleAdmin, &otherCustomerOrg)
	otherBundle := createSupportBundle(otherCustomer, "other")
	otherVendor := h.RegisterVendor()
	g := NewWithT(t)

	reads := []request{
		{"detail", http.MethodGet, path, nil},
		{"download", http.MethodGet, path + "/download", nil},
	}
	writes := []request{
		{"comment", http.MethodPost, path + "/comments", api.CreateSupportBundleCommentRequest{Content: "x"}},
		{"cancel", http.MethodPatch, path + "/status", setBundleStatus("canceled")},
		{"delete", http.MethodDelete, path, nil},
	}
	create := request{"create", http.MethodPost, apitest.SupportBundles, api.CreateSupportBundleRequest{Title: "x"}}
	all := append(append([]request{create}, reads...), writes...)

	g.Expect(statuses(f.vendor.Client, []request{create})).To(Equal(map[string]int{"create": http.StatusForbidden}),
		"only customers open bundles")
	g.Expect(statuses(colleague.Client, reads)).To(Equal(allWithStatus(reads, http.StatusOK)))
	g.Expect(statuses(colleague.Client, append([]request{create}, writes...))).
		To(Equal(allWithStatus(append([]request{create}, writes...), http.StatusForbidden)))
	g.Expect(statuses(reader.Client, reads)).To(Equal(allWithStatus(reads, http.StatusOK)))
	g.Expect(statuses(reader.Client, writes)).To(Equal(allWithStatus(writes, http.StatusForbidden)))
	g.Expect(statuses(otherCustomer.Client, append(reads, writes...))).
		To(Equal(allWithStatus(append(reads, writes...), http.StatusNotFound)))
	g.Expect(statuses(otherVendor.Client, append(reads, writes...))).
		To(Equal(allWithStatus(append(reads, writes...), http.StatusNotFound)))

	g.Expect(apitest.IDs(f.vendor.Get(apitest.SupportBundles), http.StatusOK)).
		To(ConsistOf(f.bundle.ID, otherBundle.ID))
	g.Expect(apitest.IDs(colleague.Get(apitest.SupportBundles), http.StatusOK)).To(Equal([]uuid.UUID{f.bundle.ID}))
	g.Expect(apitest.IDs(otherCustomer.Get(apitest.SupportBundles), http.StatusOK)).
		To(Equal([]uuid.UUID{otherBundle.ID}))
	g.Expect(apitest.IDs(otherVendor.Get(apitest.SupportBundles), http.StatusOK)).To(BeEmpty())
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM SupportBundleComment`)).To(BeZero())
	g.Expect(getSupportBundle(f.vendor.Client, f.bundle.ID).Status).
		To(Equal(string(types.SupportBundleStatusInitialized)))

	f.vendor.SetSubscriptionType(types.SubscriptionTypeCommunity)
	list := request{"list", http.MethodGet, apitest.SupportBundles, nil}
	configuration := request{"configuration", http.MethodGet, supportBundleConfiguration, nil}
	g.Expect(statuses(f.vendor.Client, []request{list, configuration})).
		To(Equal(allWithStatus([]request{list, configuration}, http.StatusForbidden)))
	g.Expect(statuses(f.customer.Client, append([]request{list}, all...))).
		To(Equal(allWithStatus(append([]request{list}, all...), http.StatusForbidden)))
}

func TestSupportBundleConfiguration(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	f := newSupportBundleFixture(h)
	reader := f.vendor.Invite(types.UserRoleReadOnly, nil)
	otherVendor := h.RegisterVendor()
	g := NewWithT(t)

	envVars := func(vars ...api.SupportBundleConfigurationEnvVar) api.CreateUpdateSupportBundleConfigurationRequest {
		return api.CreateUpdateSupportBundleConfigurationRequest{EnvVars: vars}
	}
	saved := apitest.Expect[[]api.SupportBundleConfigurationEnvVar](f.vendor.Put(supportBundleConfiguration,
		envVars(api.SupportBundleConfigurationEnvVar{Name: "APP_MODE"},
			api.SupportBundleConfigurationEnvVar{Name: "DB_PASSWORD", Redacted: true})), http.StatusOK)
	g.Expect(saved).To(ConsistOf(
		api.SupportBundleConfigurationEnvVar{Name: "APP_MODE"},
		api.SupportBundleConfigurationEnvVar{Name: "DB_PASSWORD", Redacted: true},
	))
	g.Expect(statuses(f.vendor.Client, []request{
		{"empty name", http.MethodPut, supportBundleConfiguration, envVars(api.SupportBundleConfigurationEnvVar{})},
		{
			"invalid name", http.MethodPut, supportBundleConfiguration,
			envVars(api.SupportBundleConfigurationEnvVar{Name: "1; rm -rf /"}),
		},
		{"duplicate name", http.MethodPut, supportBundleConfiguration, envVars(
			api.SupportBundleConfigurationEnvVar{Name: "APP_MODE"}, api.SupportBundleConfigurationEnvVar{Name: "app_mode"})},
		{"malformed body", http.MethodPut, supportBundleConfiguration, "{"},
	})).To(Equal(map[string]int{
		"empty name":     http.StatusBadRequest,
		"invalid name":   http.StatusBadRequest,
		"duplicate name": http.StatusBadRequest,
		"malformed body": http.StatusBadRequest,
	}))
	g.Expect(apitest.Expect[[]api.SupportBundleConfigurationEnvVar](f.vendor.Get(supportBundleConfiguration),
		http.StatusOK)).To(HaveLen(2), "a rejected save keeps the configuration")

	scriptBody := func(name, content string, enabled bool) api.CreateUpdateSupportBundleConfigurationScriptRequest {
		return api.CreateUpdateSupportBundleConfigurationScriptRequest{Name: name, Content: content, Enabled: enabled}
	}
	enabled := apitest.Expect[api.SupportBundleConfigurationScript](f.vendor.Post(supportBundleScripts,
		scriptBody("disk usage", "df -h", true)), http.StatusOK)
	disabled := apitest.Expect[api.SupportBundleConfigurationScript](f.vendor.Post(supportBundleScripts,
		scriptBody("processes", "ps aux", false)), http.StatusOK)
	enabledPath := apitest.Path(supportBundleScripts, enabled.ID)
	g.Expect(statuses(f.vendor.Client, []request{
		{"no name", http.MethodPost, supportBundleScripts, scriptBody("", "df", true)},
		{"name too long", http.MethodPost, supportBundleScripts, scriptBody(strings.Repeat("x", 65), "df", true)},
		{"name with a slash", http.MethodPost, supportBundleScripts, scriptBody("a/b", "df", true)},
		{"no content", http.MethodPost, supportBundleScripts, scriptBody("x", "", true)},
		{"content too large", http.MethodPost, supportBundleScripts, scriptBody("x", strings.Repeat("x", 64*1024+1), true)},
		{
			"description too long", http.MethodPost, supportBundleScripts,
			api.CreateUpdateSupportBundleConfigurationScriptRequest{
				Name: "x", Content: "df", Description: new(strings.Repeat("x", 257)),
			},
		},
		{"duplicate name", http.MethodPost, supportBundleScripts, scriptBody("Disk Usage", "du", true)},
		{"rename onto another", http.MethodPut, enabledPath, scriptBody("processes", "df -h", true)},
		{"update unknown", http.MethodPut, apitest.Path(supportBundleScripts, uuid.New()), scriptBody("x", "df", true)},
		{"update malformed id", http.MethodPut, supportBundleScripts + "/nope", scriptBody("x", "df", true)},
		{"delete unknown", http.MethodDelete, apitest.Path(supportBundleScripts, uuid.New()), nil},
	})).To(Equal(map[string]int{
		"no name":              http.StatusBadRequest,
		"name too long":        http.StatusBadRequest,
		"name with a slash":    http.StatusBadRequest,
		"no content":           http.StatusBadRequest,
		"content too large":    http.StatusBadRequest,
		"description too long": http.StatusBadRequest,
		"duplicate name":       http.StatusConflict,
		"rename onto another":  http.StatusConflict,
		"update unknown":       http.StatusNotFound,
		"update malformed id":  http.StatusNotFound,
		"delete unknown":       http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM SupportBundleConfigurationScript`)).To(Equal(2))

	script := string(h.Anonymous().Get(collectPath(f.bundle.CollectCommand)).ExpectStatus(http.StatusOK).Body)
	g.Expect(script).To(ContainSubstring("APP_MODE"))
	g.Expect(script).To(ContainSubstring("DB_PASSWORD"))
	g.Expect(script).To(ContainSubstring(base64.StdEncoding.EncodeToString([]byte("df -h"))))
	g.Expect(script).NotTo(ContainSubstring(base64.StdEncoding.EncodeToString([]byte("ps aux"))),
		"a disabled script is not collected")

	vendorOnly := []request{
		{"get env vars", http.MethodGet, supportBundleConfiguration, nil},
		{"save env vars", http.MethodPut, supportBundleConfiguration, envVars()},
		{"list scripts", http.MethodGet, supportBundleScripts, nil},
		{"create script", http.MethodPost, supportBundleScripts, scriptBody("new", "df", true)},
		{"update script", http.MethodPut, enabledPath, scriptBody("disk usage", "rm -rf /", true)},
		{"delete script", http.MethodDelete, enabledPath, nil},
	}
	g.Expect(statuses(f.customer.Client, vendorOnly)).To(Equal(allWithStatus(vendorOnly, http.StatusForbidden)))
	g.Expect(statuses(reader.Client, vendorOnly)).To(Equal(map[string]int{
		"get env vars":  http.StatusOK,
		"list scripts":  http.StatusOK,
		"save env vars": http.StatusForbidden,
		"create script": http.StatusForbidden,
		"update script": http.StatusForbidden,
		"delete script": http.StatusForbidden,
	}))
	g.Expect(statuses(otherVendor.Client, vendorOnly[4:])).To(Equal(allWithStatus(vendorOnly[4:], http.StatusNotFound)))
	g.Expect(apitest.Expect[[]api.SupportBundleConfigurationScript](otherVendor.Get(supportBundleScripts),
		http.StatusOK)).To(BeEmpty())
	g.Expect(apitest.QueryValue[string](h,
		`SELECT content FROM SupportBundleConfigurationScript WHERE id = $1`, enabled.ID)).To(Equal("df -h"))

	updated := apitest.Expect[api.SupportBundleConfigurationScript](f.vendor.Put(enabledPath,
		scriptBody("disk usage", "df -i", false)), http.StatusOK)
	g.Expect(updated.Content).To(Equal("df -i"))
	g.Expect(updated.Enabled).To(BeFalse())
	f.vendor.Delete(apitest.Path(supportBundleScripts, disabled.ID)).ExpectStatus(http.StatusNoContent)
	g.Expect(apitest.Expect[[]api.SupportBundleConfigurationScript](f.vendor.Get(supportBundleScripts),
		http.StatusOK)).To(HaveLen(1))
}
