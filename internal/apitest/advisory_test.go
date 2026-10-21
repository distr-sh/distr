package apitest_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

// vulnerabilitiesVendor registers a vendor on the Business feature the advisories need.
func vulnerabilitiesVendor(h *apitest.Harness) *apitest.User {
	vendor := h.RegisterVendor()
	vendor.GrantFeature(types.FeatureVulnerabilities)
	return vendor
}

func advisoryBody(title string) api.CreateUpdateAdvisoryRequest {
	return api.CreateUpdateAdvisoryRequest{Title: title, Severity: string(types.AdvisorySeverityHigh)}
}

func createAdvisory(user *apitest.User, body api.CreateUpdateAdvisoryRequest) uuid.UUID {
	return apitest.Expect[apitest.Entity](user.Post(apitest.Advisories, body), http.StatusOK).ID
}

func getAdvisory(client *apitest.Client, id uuid.UUID) api.AdvisoryDetail {
	return apitest.Expect[api.AdvisoryDetail](client.Get(apitest.Path(apitest.Advisories, id)), http.StatusOK)
}

func patchAdvisory(user *apitest.User, id uuid.UUID, body api.PatchAdvisoryRequest) api.AdvisoryDetail {
	return apitest.Expect[api.AdvisoryDetail](user.Patch(apitest.Path(apitest.Advisories, id), body), http.StatusOK)
}

func eventTypes(events []api.AdvisoryEvent) []types.AdvisoryEventType {
	result := make([]types.AdvisoryEventType, len(events))
	for i, event := range events {
		result[i] = event.Type
	}
	return result
}

func advisoryImpact(client *apitest.Client, id uuid.UUID) api.AdvisoryImpact {
	return apitest.Expect[api.AdvisoryImpact](client.Get(apitest.Path(apitest.Advisories, id)+"/impact"), http.StatusOK)
}

func TestAdvisoryLifecycle(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := vulnerabilitiesVendor(h)
	application := vendor.CreateApplication("app")
	affected := vendor.CreateApplicationVersion(application, "1.0.0")
	patched := vendor.CreateApplicationVersion(application, "1.0.1")
	g := NewWithT(t)

	body := advisoryBody("RCE in the login form")
	body.Description = "  indented markdown"
	body.CveID = new("cve-2026-0001")
	body.Tags = []string{"login", "rce", "login"}
	body.References = []api.AdvisoryReference{{URL: "https://example.com/advisory", Label: new("")}}
	body.AffectedApplicationVersionIDs = []uuid.UUID{affected, affected}
	body.PatchedApplicationVersionIDs = []uuid.UUID{patched}
	id := createAdvisory(vendor, body)

	detail := getAdvisory(vendor.Client, id)
	g.Expect(detail.Status).To(Equal(types.AdvisoryStatusTriage), "the status defaults to triage")
	g.Expect(detail.Severity).To(Equal(types.AdvisorySeverityHigh))
	g.Expect(detail.CveID).To(HaveValue(Equal("cve-2026-0001")))
	g.Expect(detail.Description).To(Equal("  indented markdown"), "the description keeps its leading whitespace")
	g.Expect(detail.Tags).To(Equal([]string{"login", "rce"}), "tags are deduplicated and sorted")
	g.Expect(detail.References).To(HaveLen(1))
	g.Expect(detail.References[0].Label).To(BeNil(), "an empty label is dropped")
	g.Expect(detail.AffectedVersionCount).To(BeEquivalentTo(1), "a version listed twice counts once")
	g.Expect(detail.PatchedVersionCount).To(BeEquivalentTo(1))
	g.Expect(detail.ReferenceCount).To(BeEquivalentTo(1))
	g.Expect(detail.PublishedAt).To(BeNil())
	g.Expect(detail.Affected).To(BeNil(), "vendors see the status instead of the affected flag")
	g.Expect(detail.Events).To(BeEmpty(), "nothing is recorded before the disclosure")
	g.Expect(apitest.Expect[[]string](vendor.Get(apitest.Advisories+"/tags"), http.StatusOK)).
		To(Equal([]string{"login", "rce"}))

	vendor.Post(apitest.Advisories, func() api.CreateUpdateAdvisoryRequest {
		duplicate := advisoryBody("duplicate")
		duplicate.CveID = new("CVE-2026-0001")
		return duplicate
	}()).ExpectStatus(http.StatusConflict)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Advisory`)).To(Equal(1),
		"a CVE ID is unique per organization regardless of its case")

	updated := body
	updated.Title = "RCE in the login form (updated)"
	updated.Tags = []string{"rce"}
	updated.Status = types.AdvisoryStatusDraft
	detail = apitest.Expect[api.AdvisoryDetail](vendor.Put(apitest.Path(apitest.Advisories, id), updated),
		http.StatusOK)
	g.Expect(detail.Status).To(Equal(types.AdvisoryStatusDraft))
	g.Expect(detail.Tags).To(Equal([]string{"rce"}))
	g.Expect(detail.Events).To(BeEmpty(), "edits of an undisclosed advisory leave no trace")
	updated.Status = ""
	detail = apitest.Expect[api.AdvisoryDetail](vendor.Put(apitest.Path(apitest.Advisories, id), updated),
		http.StatusOK)
	g.Expect(detail.Status).To(Equal(types.AdvisoryStatusDraft), "a PUT without status keeps the status")

	detail = patchAdvisory(vendor, id, api.PatchAdvisoryRequest{Status: new(types.AdvisoryStatusPublished)})
	g.Expect(detail.PublishedAt).NotTo(BeNil())
	g.Expect(eventTypes(detail.Events)).To(Equal([]types.AdvisoryEventType{types.AdvisoryEventTypePublished}))
	publishedAt := *detail.PublishedAt

	detail = patchAdvisory(vendor, id, api.PatchAdvisoryRequest{Severity: new(types.AdvisorySeverityCritical)})
	g.Expect(detail.Severity).To(Equal(types.AdvisorySeverityCritical))
	g.Expect(detail.Status).To(Equal(types.AdvisoryStatusPublished), "a patch of the severity keeps the status")
	g.Expect(eventTypes(detail.Events)).To(Equal([]types.AdvisoryEventType{
		types.AdvisoryEventTypePublished, types.AdvisoryEventTypeEdited,
	}))
	g.Expect(detail.Events[1].Message).To(HaveValue(Equal("changed the severity from high to critical")))

	updated.Title = "RCE in the login form"
	updated.Severity = string(types.AdvisorySeverityCritical)
	updated.References = nil
	updated.PatchedApplicationVersionIDs = nil
	detail = apitest.Expect[api.AdvisoryDetail](vendor.Put(apitest.Path(apitest.Advisories, id), updated),
		http.StatusOK)
	g.Expect(detail.Events).To(HaveLen(3), "one edit is one timeline entry")
	g.Expect(detail.Events[2].Type).To(Equal(types.AdvisoryEventTypeEdited))
	message := *detail.Events[2].Message
	g.Expect(message).
		To(ContainSubstring(`changed the title from "RCE in the login form (updated)" to "RCE in the login form"`))
	g.Expect(message).To(ContainSubstring("removed the references https://example.com/advisory"))
	g.Expect(message).To(ContainSubstring("unmarked app 1.0.1"))
	g.Expect(detail.PatchedVersionCount).To(BeZero())
	detail = apitest.Expect[api.AdvisoryDetail](vendor.Put(apitest.Path(apitest.Advisories, id), updated),
		http.StatusOK)
	g.Expect(detail.Events).To(HaveLen(3), "a save without changes records nothing")

	comment := apitest.Expect[api.AdvisoryEvent](vendor.Post(apitest.Path(apitest.Advisories, id)+"/comments",
		api.CreateAdvisoryCommentRequest{Content: "fix is being rolled out"}), http.StatusOK)
	g.Expect(comment.Type).To(Equal(types.AdvisoryEventTypeComment))
	g.Expect(comment.UserName).To(HaveValue(Equal("Vendor")))

	detail = patchAdvisory(vendor, id, api.PatchAdvisoryRequest{Status: new(types.AdvisoryStatusResolved)})
	g.Expect(detail.ResolvedAt).NotTo(BeNil())
	g.Expect(detail.PublishedAt).To(HaveValue(Equal(publishedAt)), "resolving keeps the disclosure date")
	g.Expect(eventTypes(detail.Events)).To(Equal([]types.AdvisoryEventType{
		types.AdvisoryEventTypePublished, types.AdvisoryEventTypeEdited, types.AdvisoryEventTypeEdited,
		types.AdvisoryEventTypeComment, types.AdvisoryEventTypeStatusChanged,
	}))
	g.Expect(detail.Events[4].Message).To(HaveValue(Equal("changed status from published to resolved")))

	detail = patchAdvisory(vendor, id, api.PatchAdvisoryRequest{Status: new(types.AdvisoryStatusPublished)})
	g.Expect(detail.ResolvedAt).To(BeNil(), "reopening clears the resolution date")
	g.Expect(detail.PublishedAt).To(HaveValue(Equal(publishedAt)), "republishing keeps the original disclosure date")

	other := createAdvisory(vendor, func() api.CreateUpdateAdvisoryRequest {
		body := advisoryBody("low")
		body.Severity = string(types.AdvisorySeverityLow)
		body.Tags = []string{"ui"}
		return body
	}())
	list := func(query string) []uuid.UUID {
		return apitest.IDs(vendor.Get(apitest.Advisories+query), http.StatusOK)
	}
	g.Expect(list("")).To(ConsistOf(id, other))
	g.Expect(list("?status=published")).To(Equal([]uuid.UUID{id}))
	g.Expect(list("?status=triage&status=draft")).To(Equal([]uuid.UUID{other}))
	g.Expect(list("?severity=low")).To(Equal([]uuid.UUID{other}))
	g.Expect(list("?tag=rce&tag=")).To(Equal([]uuid.UUID{id}))
	g.Expect(list("?tag=login")).To(BeEmpty(), "a removed tag matches nothing")
	g.Expect(list("?status=published&tag=ui")).To(BeEmpty(), "filters are combined")
	g.Expect(apitest.Expect[[]string](vendor.Get(apitest.Advisories+"/tags"), http.StatusOK)).
		To(Equal([]string{"rce", "ui"}))
}

func TestAdvisoryCreatedAsPublishedIsStampedAsPublished(t *testing.T) {
	knownBug(t, "the INSERT does not maintain published_at, so an advisory created in the published "+
		"status has no disclosure date and no publication event")
	t.Parallel()
	h := apitest.New(t)
	vendor := vulnerabilitiesVendor(h)
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	g := NewWithT(t)

	body := advisoryBody("published on creation")
	body.Status = types.AdvisoryStatusPublished
	body.AffectedApplicationVersionIDs = []uuid.UUID{version}
	detail := apitest.Expect[api.AdvisoryDetail](vendor.Post(apitest.Advisories, body), http.StatusOK)
	g.Expect(detail.PublishedAt).NotTo(BeNil())
	g.Expect(eventTypes(detail.Events)).To(Equal([]types.AdvisoryEventType{types.AdvisoryEventTypePublished}))
}

func TestAdvisoryErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := vulnerabilitiesVendor(h)
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	otherVendor := h.RegisterVendor()
	foreignVersion := otherVendor.CreateApplicationVersion(otherVendor.CreateApplication("app"), "1.0.0")
	existing := createAdvisory(vendor, advisoryBody("existing"))
	existingPath := apitest.Path(apitest.Advisories, existing)
	unknownPath := apitest.Path(apitest.Advisories, uuid.New())
	g := NewWithT(t)

	tags := make([]string, 21)
	for i := range tags {
		tags[i] = "tag-" + string(rune('a'+i))
	}
	with := func(name string, change func(body *api.CreateUpdateAdvisoryRequest)) request {
		body := advisoryBody("valid")
		change(&body)
		return request{name, http.MethodPost, apitest.Advisories, body}
	}
	g.Expect(statuses(vendor.Client, []request{
		with("no title", func(b *api.CreateUpdateAdvisoryRequest) { b.Title = "" }),
		with("title too long", func(b *api.CreateUpdateAdvisoryRequest) { b.Title = strings.Repeat("x", 201) }),
		with("no severity", func(b *api.CreateUpdateAdvisoryRequest) { b.Severity = "" }),
		with("unknown severity", func(b *api.CreateUpdateAdvisoryRequest) { b.Severity = "urgent" }),
		with("unknown status", func(b *api.CreateUpdateAdvisoryRequest) { b.Status = "open" }),
		with("too many tags", func(b *api.CreateUpdateAdvisoryRequest) { b.Tags = tags }),
		with("empty tag", func(b *api.CreateUpdateAdvisoryRequest) { b.Tags = []string{""} }),
		with("tag too long", func(b *api.CreateUpdateAdvisoryRequest) { b.Tags = []string{strings.Repeat("x", 51)} }),
		with("reference without url", func(b *api.CreateUpdateAdvisoryRequest) {
			b.References = []api.AdvisoryReference{{Label: new("docs")}}
		}),
		with("reference with another scheme", func(b *api.CreateUpdateAdvisoryRequest) {
			b.References = []api.AdvisoryReference{{URL: "ftp://example.com"}}
		}),
		with("reference without host", func(b *api.CreateUpdateAdvisoryRequest) {
			b.References = []api.AdvisoryReference{{URL: "https:///path"}}
		}),
		with("version affected and patched", func(b *api.CreateUpdateAdvisoryRequest) {
			b.AffectedApplicationVersionIDs = []uuid.UUID{version}
			b.PatchedApplicationVersionIDs = []uuid.UUID{version}
		}),
		with("unknown version", func(b *api.CreateUpdateAdvisoryRequest) {
			b.AffectedApplicationVersionIDs = []uuid.UUID{uuid.New()}
		}),
		with("unknown artifact version", func(b *api.CreateUpdateAdvisoryRequest) {
			b.PatchedArtifactVersionIDs = []uuid.UUID{uuid.New()}
		}),
		with("version of another organization", func(b *api.CreateUpdateAdvisoryRequest) {
			b.AffectedApplicationVersionIDs = []uuid.UUID{foreignVersion}
		}),
		{"malformed body", http.MethodPost, apitest.Advisories, "{"},
		{"get malformed id", http.MethodGet, apitest.Advisories + "/nope", nil},
		{"get unknown", http.MethodGet, unknownPath, nil},
		{"impact of unknown", http.MethodGet, unknownPath + "/impact", nil},
		{"list with unknown status", http.MethodGet, apitest.Advisories + "?status=open", nil},
		{"list with unknown severity", http.MethodGet, apitest.Advisories + "?severity=urgent", nil},
		{"put unknown", http.MethodPut, unknownPath, advisoryBody("x")},
		{"put without title", http.MethodPut, existingPath, advisoryBody("")},
		{"put version of another organization", http.MethodPut, existingPath, func() api.CreateUpdateAdvisoryRequest {
			body := advisoryBody("existing")
			body.PatchedApplicationVersionIDs = []uuid.UUID{foreignVersion}
			return body
		}()},
		{"patch without fields", http.MethodPatch, existingPath, api.PatchAdvisoryRequest{}},
		{"patch unknown status", http.MethodPatch, existingPath, object{"status": "open"}},
		{"patch unknown severity", http.MethodPatch, existingPath, object{"severity": "urgent"}},
		{"patch unknown", http.MethodPatch, unknownPath, api.PatchAdvisoryRequest{Status: new(types.AdvisoryStatusDraft)}},
		{"patch malformed body", http.MethodPatch, existingPath, "{"},
		{"empty comment", http.MethodPost, existingPath + "/comments", api.CreateAdvisoryCommentRequest{}},
		{"blank comment", http.MethodPost, existingPath + "/comments", api.CreateAdvisoryCommentRequest{Content: "  "}},
		{"comment on unknown", http.MethodPost, unknownPath + "/comments", api.CreateAdvisoryCommentRequest{Content: "x"}},
	})).To(Equal(map[string]int{
		"no title":                            http.StatusBadRequest,
		"title too long":                      http.StatusBadRequest,
		"no severity":                         http.StatusBadRequest,
		"unknown severity":                    http.StatusBadRequest,
		"unknown status":                      http.StatusBadRequest,
		"too many tags":                       http.StatusBadRequest,
		"empty tag":                           http.StatusBadRequest,
		"tag too long":                        http.StatusBadRequest,
		"reference without url":               http.StatusBadRequest,
		"reference with another scheme":       http.StatusBadRequest,
		"reference without host":              http.StatusBadRequest,
		"version affected and patched":        http.StatusBadRequest,
		"unknown version":                     http.StatusBadRequest,
		"unknown artifact version":            http.StatusBadRequest,
		"version of another organization":     http.StatusBadRequest,
		"malformed body":                      http.StatusBadRequest,
		"get malformed id":                    http.StatusNotFound,
		"get unknown":                         http.StatusNotFound,
		"impact of unknown":                   http.StatusNotFound,
		"list with unknown status":            http.StatusBadRequest,
		"list with unknown severity":          http.StatusBadRequest,
		"put unknown":                         http.StatusNotFound,
		"put without title":                   http.StatusBadRequest,
		"put version of another organization": http.StatusBadRequest,
		"patch without fields":                http.StatusBadRequest,
		"patch unknown status":                http.StatusBadRequest,
		"patch unknown severity":              http.StatusBadRequest,
		"patch unknown":                       http.StatusNotFound,
		"patch malformed body":                http.StatusBadRequest,
		"empty comment":                       http.StatusBadRequest,
		"blank comment":                       http.StatusBadRequest,
		"comment on unknown":                  http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Advisory`)).To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AdvisoryApplicationVersion`)).To(BeZero())
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AdvisoryEvent`)).To(BeZero())
	detail := getAdvisory(vendor.Client, existing)
	g.Expect(detail.Title).To(Equal("existing"))
	g.Expect(detail.Status).To(Equal(types.AdvisoryStatusTriage))
}

func TestAdvisoryAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := vulnerabilitiesVendor(h)
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	body := advisoryBody("published")
	body.Status = types.AdvisoryStatusPublished
	body.AffectedApplicationVersionIDs = []uuid.UUID{version}
	published := createAdvisory(vendor, body)
	publishedPath := apitest.Path(apitest.Advisories, published)
	reader := vendor.Invite(types.UserRoleReadOnly, nil)
	customer := vendor.Invite(types.UserRoleAdmin, new(vendor.CreateCustomerOrganization("Customer")))
	otherVendor := vulnerabilitiesVendor(h)
	withoutFeature := h.RegisterVendor()
	g := NewWithT(t)

	reads := []request{
		{"list", http.MethodGet, apitest.Advisories, nil},
		{"detail", http.MethodGet, publishedPath, nil},
		{"impact", http.MethodGet, publishedPath + "/impact", nil},
	}
	vendorOnly := []request{{"tags", http.MethodGet, apitest.Advisories + "/tags", nil}}
	writes := []request{
		{"create", http.MethodPost, apitest.Advisories, advisoryBody("new")},
		{"update", http.MethodPut, publishedPath, body},
		{"patch", http.MethodPatch, publishedPath, api.PatchAdvisoryRequest{Severity: new(types.AdvisorySeverityLow)}},
		{"comment", http.MethodPost, publishedPath + "/comments", api.CreateAdvisoryCommentRequest{Content: "x"}},
	}
	all := append(append(append([]request{}, reads...), vendorOnly...), writes...)

	g.Expect(statuses(withoutFeature.Client, all)).To(Equal(allWithStatus(all, http.StatusForbidden)))
	g.Expect(statuses(reader.Client, append(append([]request{}, reads...), vendorOnly...))).
		To(Equal(allWithStatus(append(append([]request{}, reads...), vendorOnly...), http.StatusOK)))
	g.Expect(statuses(reader.Client, writes)).To(Equal(allWithStatus(writes, http.StatusForbidden)))
	g.Expect(statuses(customer.Client, reads)).To(Equal(allWithStatus(reads, http.StatusOK)),
		"a published advisory with affected versions is visible to a customer without entitlements")
	g.Expect(statuses(customer.Client, append(append([]request{}, vendorOnly...), writes...))).
		To(Equal(allWithStatus(append(append([]request{}, vendorOnly...), writes...), http.StatusForbidden)))
	g.Expect(statuses(otherVendor.Client, all)).To(Equal(map[string]int{
		"list":    http.StatusOK,
		"tags":    http.StatusOK,
		"create":  http.StatusOK,
		"detail":  http.StatusNotFound,
		"impact":  http.StatusNotFound,
		"update":  http.StatusNotFound,
		"patch":   http.StatusNotFound,
		"comment": http.StatusNotFound,
	}))
	g.Expect(apitest.IDs(otherVendor.Get(apitest.Advisories), http.StatusOK)).To(HaveLen(1),
		"an organization lists only its own advisories")
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Advisory WHERE organization_id = $1`,
		vendor.OrganizationID)).To(Equal(1))
	detail := getAdvisory(vendor.Client, published)
	g.Expect(detail.Severity).To(Equal(types.AdvisorySeverityHigh))

	customerDetail := getAdvisory(customer.Client, published)
	g.Expect(customerDetail.Events).To(BeEmpty(), "the timeline is vendor-internal")
	g.Expect(customerDetail.Affected).To(HaveValue(BeFalse()))
	g.Expect(customerDetail.ResolvedAt).To(BeNil())
	g.Expect(customerDetail.CreatedByUserName).To(BeNil())
}

func TestAdvisoryCustomerVisibility(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := vulnerabilitiesVendor(h)
	application := vendor.CreateApplication("app")
	affected := vendor.CreateApplicationVersion(application, "1.0.0")
	patched := vendor.CreateApplicationVersion(application, "1.0.1")
	otherApplication := vendor.CreateApplication("other")
	otherAffected := vendor.CreateApplicationVersion(otherApplication, "1.0.0")
	entitledOrg := vendor.CreateCustomerOrganization("Entitled")
	entitled := vendor.Invite(types.UserRoleReadOnly, &entitledOrg)
	deployedOrg := vendor.CreateCustomerOrganization("Deployed")
	deployed := vendor.Invite(types.UserRoleReadOnly, &deployedOrg)
	deployedTarget := vendor.CreateDeploymentTarget("deployed", &deployedOrg)
	unrelatedOrg := vendor.CreateCustomerOrganization("Unrelated")
	unrelated := vendor.Invite(types.UserRoleReadOnly, &unrelatedOrg)
	g := NewWithT(t)

	deployment, _ := vendor.Deploy(api.DeploymentRequest{
		DeploymentTargetID: deployedTarget, ApplicationVersionID: affected,
	})

	body := advisoryBody("app advisory")
	body.AffectedApplicationVersionIDs = []uuid.UUID{affected}
	body.PatchedApplicationVersionIDs = []uuid.UUID{patched}
	advisory := createAdvisory(vendor, body)
	withoutVersions := createAdvisory(vendor, func() api.CreateUpdateAdvisoryRequest {
		body := advisoryBody("published without affected versions")
		body.Status = types.AdvisoryStatusPublished
		body.PatchedApplicationVersionIDs = []uuid.UUID{patched}
		return body
	}())
	otherAdvisory := createAdvisory(vendor, func() api.CreateUpdateAdvisoryRequest {
		body := advisoryBody("other app advisory")
		body.Status = types.AdvisoryStatusPublished
		body.AffectedApplicationVersionIDs = []uuid.UUID{otherAffected}
		return body
	}())
	visibleTo := func(user *apitest.User) []uuid.UUID {
		return apitest.IDs(user.Get(apitest.Advisories), http.StatusOK)
	}
	advisoryPath := apitest.Path(apitest.Advisories, advisory)

	g.Expect(visibleTo(deployed)).To(Equal([]uuid.UUID{otherAdvisory}),
		"an advisory under triage is hidden and one without affected versions is not disclosed")
	deployed.Get(advisoryPath).ExpectStatus(http.StatusNotFound)
	deployed.Get(advisoryPath + "/impact").ExpectStatus(http.StatusNotFound)
	g.Expect(visibleTo(vendor)).To(ConsistOf(advisory, withoutVersions, otherAdvisory))

	patchAdvisory(vendor, advisory, api.PatchAdvisoryRequest{Status: new(types.AdvisoryStatusPublished)})
	g.Expect(visibleTo(unrelated)).To(ConsistOf(advisory, otherAdvisory),
		"without entitlements, every customer sees every disclosed advisory")
	g.Expect(getAdvisory(unrelated.Client, advisory).ApplicationVersions).To(HaveLen(2))

	createEntitlement(vendor, application, entitledOrg)
	g.Expect(visibleTo(unrelated)).To(BeEmpty(),
		"the first application entitlement gates every application advisory")
	g.Expect(visibleTo(entitled)).To(Equal([]uuid.UUID{advisory}),
		"an entitlement discloses the advisories of the application it covers")
	g.Expect(visibleTo(deployed)).To(Equal([]uuid.UUID{advisory}),
		"a customer who deployed an affected version sees the advisory without an entitlement")

	deployedDetail := getAdvisory(deployed.Client, advisory)
	g.Expect(deployedDetail.Affected).To(HaveValue(BeTrue()))
	g.Expect(deployedDetail.ApplicationVersions).To(HaveLen(1), "only the deployed version is disclosed")
	g.Expect(deployedDetail.ApplicationVersions[0].ApplicationVersion.ID).To(Equal(affected))
	g.Expect(deployedDetail.PatchedVersionCount).To(BeZero(), "the counts match the disclosed versions")
	g.Expect(deployedDetail.AffectedVersionCount).To(BeEquivalentTo(1))
	entitledDetail := getAdvisory(entitled.Client, advisory)
	g.Expect(entitledDetail.Affected).To(HaveValue(BeFalse()))
	g.Expect(entitledDetail.ApplicationVersions).To(HaveLen(2))
	g.Expect(entitledDetail.PatchedVersionCount).To(BeEquivalentTo(1))

	impact := advisoryImpact(vendor.Client, advisory)
	g.Expect(impact.Deployments).To(HaveLen(1))
	g.Expect(impact.Deployments[0].DeploymentID).To(Equal(deployment))
	g.Expect(impact.Deployments[0].CustomerOrganizationID).To(HaveValue(Equal(deployedOrg)))
	g.Expect(impact.Deployments[0].State).To(Equal(types.AdvisoryImpactStateAffected))
	g.Expect(advisoryImpact(deployed.Client, advisory).Deployments).To(HaveLen(1))
	g.Expect(advisoryImpact(entitled.Client, advisory).Deployments).To(BeEmpty(),
		"a customer sees the impact on its own deployments only")

	vendor.Deploy(api.DeploymentRequest{
		DeploymentID: &deployment, DeploymentTargetID: deployedTarget, ApplicationVersionID: patched,
	})
	g.Expect(visibleTo(deployed)).To(Equal([]uuid.UUID{advisory}), "an upgraded customer keeps seeing the advisory")
	deployedDetail = getAdvisory(deployed.Client, advisory)
	g.Expect(deployedDetail.Affected).To(HaveValue(BeFalse()), "but is no longer affected")
	g.Expect(deployedDetail.ApplicationVersions).To(HaveLen(2), "and now knows the patched version too")
	impact = advisoryImpact(vendor.Client, advisory)
	g.Expect(impact.Deployments).To(HaveLen(1))
	g.Expect(impact.Deployments[0].State).To(Equal(types.AdvisoryImpactStatePatched))
	g.Expect(impact.Deployments[0].CurrentApplicationVersionID).To(Equal(patched))
	g.Expect(impact.Deployments[0].ApplicationVersionID).To(Equal(affected), "the exposure window stays visible")

	patchAdvisory(vendor, advisory, api.PatchAdvisoryRequest{Status: new(types.AdvisoryStatusCanceled)})
	g.Expect(visibleTo(deployed)).To(BeEmpty(), "a canceled advisory is withdrawn")
	patchAdvisory(vendor, advisory, api.PatchAdvisoryRequest{Status: new(types.AdvisoryStatusResolved)})
	g.Expect(visibleTo(deployed)).To(Equal([]uuid.UUID{advisory}), "a resolved one stays visible")
}

func TestAdvisoryArtifactVersions(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	vendor.GrantFeature(types.FeatureVulnerabilities)
	artifact := vendor.push(t, "app", "1.0.0", "first")
	vendor.push(t, "app", "1.0.1", "second")
	artifactDetail := getArtifact(vendor.Client, artifact)
	versions := artifactDetail.tags()
	vendor.push(t, "app", "latest", "second")
	customerOrg := vendor.CreateCustomerOrganization("Customer")
	customer := vendor.Invite(types.UserRoleReadOnly, &customerOrg)
	customerKey := customer.CreateAccessToken()
	otherOrg := vendor.CreateCustomerOrganization("Other")
	other := vendor.Invite(types.UserRoleReadOnly, &otherOrg)
	g := NewWithT(t)

	g.Expect(h.PullImage(customerKey, "vendor/app", "1.0.0")).To(Succeed())
	g.Expect(h.PullImage(customerKey, "vendor/app", "1.0.0")).To(Succeed())

	body := advisoryBody("artifact advisory")
	body.Status = types.AdvisoryStatusPublished
	body.AffectedArtifactVersionIDs = []uuid.UUID{versions["1.0.0"]}
	body.PatchedArtifactVersionIDs = []uuid.UUID{versions["1.0.1"]}
	advisory := createAdvisory(vendor.User, body)
	advisoryPath := apitest.Path(apitest.Advisories, advisory)

	detail := getAdvisory(vendor.Client, advisory)
	g.Expect(detail.ArtifactVersions).To(HaveLen(2))
	g.Expect(detail.ArtifactVersions[1].ArtifactVersionName).To(Equal(artifactDetail.digest("1.0.1")),
		"the artifacts page marks a version by its digest")
	g.Expect(detail.ArtifactVersions[1].ArtifactVersionTags).To(Equal([]string{"1.0.1", "latest"}),
		"the tags sharing the content travel with a marked version")

	impact := advisoryImpact(vendor.Client, advisory)
	g.Expect(impact.Deployments).To(BeEmpty())
	g.Expect(impact.Pulls).To(HaveLen(1))
	g.Expect(impact.Pulls[0].CustomerOrganizationID).To(HaveValue(Equal(customerOrg)))
	g.Expect(impact.Pulls[0].ArtifactVersionName).To(Equal("1.0.0"),
		"a pull of a tag counts for the digest the advisory marks")
	g.Expect(impact.Pulls[0].PullCount).To(BeEquivalentTo(2))

	g.Expect(apitest.IDs(other.Get(apitest.Advisories), http.StatusOK)).To(Equal([]uuid.UUID{advisory}),
		"without artifact entitlements, every customer sees the advisory")
	g.Expect(getAdvisory(other.Client, advisory).Affected).To(HaveValue(BeFalse()))
	g.Expect(advisoryImpact(other.Client, advisory).Pulls).To(BeEmpty())
	g.Expect(getAdvisory(customer.Client, advisory).Affected).To(HaveValue(BeTrue()),
		"a customer who pulled an affected version is affected")
	g.Expect(advisoryImpact(customer.Client, advisory).Pulls).To(HaveLen(1))

	g.Expect(h.PullImage(customerKey, "vendor/app", "latest")).To(Succeed())
	g.Expect(getAdvisory(customer.Client, advisory).Affected).To(HaveValue(BeFalse()),
		"pulling another tag of the patched content clears the artifact")

	vendor.Post(apitest.ArtifactEntitlements, artifactEntitlementBody("other", &otherOrg,
		artifactSelection(artifact, versions["1.0.1"]))).ExpectStatus(http.StatusOK)
	g.Expect(apitest.IDs(other.Get(apitest.Advisories), http.StatusOK)).To(BeEmpty(),
		"an entitlement to the patched version alone discloses nothing")
	other.Get(advisoryPath).ExpectStatus(http.StatusNotFound)
	g.Expect(apitest.IDs(customer.Get(apitest.Advisories), http.StatusOK)).To(BeEmpty(),
		"unlike a deployment, an earlier pull grants no visibility once artifacts are gated")

	vendor.Post(apitest.ArtifactEntitlements, artifactEntitlementBody("customer", &customerOrg,
		artifactSelection(artifact, versions["1.0.0"]))).ExpectStatus(http.StatusOK)
	g.Expect(apitest.IDs(customer.Get(apitest.Advisories), http.StatusOK)).To(Equal([]uuid.UUID{advisory}))
	customerDetail := getAdvisory(customer.Client, advisory)
	g.Expect(customerDetail.ArtifactVersions).To(HaveLen(2), "the pulled patched version stays disclosed")
	g.Expect(customerDetail.Events).To(BeEmpty())
}
