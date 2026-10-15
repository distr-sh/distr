package apitest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

// alertConfiguration is the part of the alert configuration response the tests read. The endpoint
// still answers the internal type.
type alertConfiguration struct {
	ID                     uuid.UUID   `json:"id"`
	CustomerOrganizationID *uuid.UUID  `json:"customerOrganizationId"`
	Name                   string      `json:"name"`
	Enabled                bool        `json:"enabled"`
	StatusTriggerEnabled   bool        `json:"statusTriggerEnabled"`
	CpuTriggerThreshold    *int        `json:"cpuTriggerThresholdPercent"`
	DeploymentTargetIDs    []uuid.UUID `json:"deploymentTargetIds"`
	UserAccountIDs         []uuid.UUID `json:"userAccountIds"`
}

func alertBody(name string, targets, users []uuid.UUID) object {
	return object{
		"name":                 name,
		"enabled":              true,
		"statusTriggerEnabled": true,
		"deploymentTargetIds":  targets,
		"userAccountIds":       users,
	}
}

func createAlert(user *apitest.User, body object) alertConfiguration {
	return apitest.Expect[alertConfiguration](user.Post(apitest.AlertConfigurations, body), http.StatusOK)
}

func updateNotificationBody(
	name string, applications, users []uuid.UUID,
) api.CreateUpdateNotificationConfigurationRequest {
	return api.CreateUpdateNotificationConfigurationRequest{
		Name: name, Enabled: true, ApplicationIDs: applications, UserAccountIDs: users,
	}
}

func createUpdateNotification(
	user *apitest.User, body api.CreateUpdateNotificationConfigurationRequest,
) api.UpdateNotificationConfiguration {
	return apitest.Expect[api.UpdateNotificationConfiguration](user.Post(apitest.UpdateNotifications, body),
		http.StatusOK)
}

func withCustomer(path string, customer uuid.UUID) string {
	return path + "?customerOrganizationId=" + customer.String()
}

func notificationRecords(client *apitest.Client, path string) []api.NotificationRecord {
	return apitest.Expect[[]api.NotificationRecord](client.Get(path), http.StatusOK)
}

func recordTypes(records []api.NotificationRecord) []string {
	result := make([]string, len(records))
	for i, record := range records {
		result[i] = record.Type
	}
	return result
}

func subjects(mails []apitest.Mail) []string {
	result := make([]string, len(mails))
	for i, mail := range mails {
		result[i] = mail.Subject
	}
	return result
}

// mailsAfterSetup returns a function listing the mails sent from now on, which leaves the invitation
// mails of the fixture out.
func mailsAfterSetup(h *apitest.Harness) func() []apitest.Mail {
	setup := len(h.Mails())
	return func() []apitest.Mail {
		return h.Mails()[setup:]
	}
}

func TestAlertConfigurationLifecycle(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	colleague := vendor.Invite(types.UserRoleReadOnly, nil)
	target := vendor.CreateDeploymentTarget("target", nil)
	otherTarget := vendor.CreateDeploymentTarget("other", nil)
	customerOrg := vendor.CreateCustomerOrganization("Acme")
	customerTarget := vendor.CreateDeploymentTarget("customer", &customerOrg)
	customer := vendor.Invite(types.UserRoleReadOnly, &customerOrg)
	otherVendor := h.RegisterVendor()
	foreignTarget := otherVendor.CreateDeploymentTarget("foreign", nil)
	g := NewWithT(t)

	created := createAlert(vendor, alertBody("status",
		[]uuid.UUID{target, customerTarget, foreignTarget, uuid.New()},
		[]uuid.UUID{vendor.ID, customer.ID, otherVendor.ID, uuid.New()}))
	g.Expect(created.CustomerOrganizationID).To(BeNil())
	g.Expect(created.DeploymentTargetIDs).To(ConsistOf(target, customerTarget),
		"a vendor links any target of the organization, and ids of other organizations are dropped")
	g.Expect(created.UserAccountIDs).To(ConsistOf(vendor.ID, customer.ID),
		"a vendor reaches every member of the organization, and other organizations' users are dropped")
	g.Expect(apitest.QueryValue[int](h,
		`SELECT count(*) FROM AlertConfiguration_DeploymentTarget WHERE alert_configuration_id = $1`, created.ID)).
		To(Equal(2))

	list := apitest.Expect[[]alertConfiguration](vendor.Get(apitest.AlertConfigurations), http.StatusOK)
	g.Expect(list).To(HaveLen(1))
	g.Expect(list[0].ID).To(Equal(created.ID))

	path := apitest.Path(apitest.AlertConfigurations, created.ID)
	body := alertBody("renamed", []uuid.UUID{otherTarget}, []uuid.UUID{colleague.ID})
	body["cpuTriggerThresholdPercent"] = 80
	body["enabled"] = false
	updated := apitest.Expect[alertConfiguration](vendor.Put(path, body), http.StatusOK)
	g.Expect(updated.Name).To(Equal("renamed"))
	g.Expect(updated.Enabled).To(BeFalse())
	g.Expect(updated.CpuTriggerThreshold).To(HaveValue(Equal(80)))
	g.Expect(updated.DeploymentTargetIDs).To(Equal([]uuid.UUID{otherTarget}), "links are replaced, not added")
	g.Expect(updated.UserAccountIDs).To(Equal([]uuid.UUID{colleague.ID}))

	g.Expect(statuses(vendor.Client, []request{
		{"get malformed id", http.MethodPut, apitest.AlertConfigurations + "/nope", body},
		{"update unknown", http.MethodPut, apitest.Path(apitest.AlertConfigurations, uuid.New()), body},
		{"update malformed body", http.MethodPut, path, "{"},
		{"delete malformed id", http.MethodDelete, apitest.AlertConfigurations + "/nope", nil},
		{"delete unknown", http.MethodDelete, apitest.Path(apitest.AlertConfigurations, uuid.New()), nil},
		{"list with malformed customer", http.MethodGet, apitest.AlertConfigurations + "?customerOrganizationId=x", nil},
		{"list with unknown customer", http.MethodGet, withCustomer(apitest.AlertConfigurations, uuid.New()), nil},
		{"create for unknown customer", http.MethodPost, apitest.AlertConfigurations, func() object {
			body := alertBody("x", nil, nil)
			body["customerOrganizationId"] = uuid.New()
			return body
		}()},
	})).To(Equal(map[string]int{
		"get malformed id":             http.StatusBadRequest,
		"update unknown":               http.StatusNotFound,
		"update malformed body":        http.StatusBadRequest,
		"delete malformed id":          http.StatusBadRequest,
		"delete unknown":               http.StatusNotFound,
		"list with malformed customer": http.StatusBadRequest,
		"list with unknown customer":   http.StatusBadRequest,
		"create for unknown customer":  http.StatusBadRequest,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AlertConfiguration`)).To(Equal(1))

	vendor.Delete(path).ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AlertConfiguration`)).To(BeZero())
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AlertConfiguration_Organization_UserAccount`)).
		To(BeZero())
}

func TestAlertConfigurationRejectsThresholdsOutOfRange(t *testing.T) {
	knownBug(t, "the alert configuration endpoints validate nothing, so a threshold of 150 percent or "+
		"-1 percent is stored and never triggers or always triggers")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	g := NewWithT(t)

	over := alertBody("over", nil, []uuid.UUID{vendor.ID})
	over["cpuTriggerThresholdPercent"] = 150
	negative := alertBody("negative", nil, []uuid.UUID{vendor.ID})
	negative["memoryTriggerThresholdPercent"] = -1
	g.Expect(statuses(vendor.Client, []request{
		{"over 100", http.MethodPost, apitest.AlertConfigurations, over},
		{"negative", http.MethodPost, apitest.AlertConfigurations, negative},
	})).To(Equal(map[string]int{
		"over 100": http.StatusBadRequest,
		"negative": http.StatusBadRequest,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AlertConfiguration`)).To(BeZero())
}

func TestAlertConfigurationScoping(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	reader := vendor.Invite(types.UserRoleReadOnly, nil)
	vendorTarget := vendor.CreateDeploymentTarget("vendor", nil)
	customerOrg := vendor.CreateCustomerOrganization("Acme", types.CustomerOrganizationFeatureAlerts)
	customer := vendor.Invite(types.UserRoleAdmin, &customerOrg)
	customerReader := vendor.Invite(types.UserRoleReadOnly, &customerOrg)
	customerTarget := customer.CreateDeploymentTarget("customer", nil)
	otherOrg := vendor.CreateCustomerOrganization("Other", types.CustomerOrganizationFeatureAlerts)
	other := vendor.Invite(types.UserRoleAdmin, &otherOrg)
	otherTarget := vendor.CreateDeploymentTarget("other", &otherOrg)
	withoutFeature := vendor.Invite(types.UserRoleAdmin, new(vendor.CreateCustomerOrganization("Plain")))
	g := NewWithT(t)

	vendorConfig := createAlert(vendor, alertBody("vendor", []uuid.UUID{vendorTarget}, []uuid.UUID{vendor.ID}))
	customerConfig := createAlert(customer, alertBody("customer",
		[]uuid.UUID{customerTarget, vendorTarget, otherTarget},
		[]uuid.UUID{customer.ID, customerReader.ID, vendor.ID, other.ID}))
	g.Expect(customerConfig.CustomerOrganizationID).To(HaveValue(Equal(customerOrg)))
	g.Expect(customerConfig.DeploymentTargetIDs).To(Equal([]uuid.UUID{customerTarget}),
		"a customer links its own targets only")
	g.Expect(customerConfig.UserAccountIDs).To(ConsistOf(customer.ID, customerReader.ID),
		"a customer reaches its own members only")

	forVendor := createAlert(vendor, func() object {
		body := alertBody("by vendor for customer", []uuid.UUID{customerTarget, vendorTarget}, []uuid.UUID{customer.ID})
		body["customerOrganizationId"] = customerOrg
		return body
	}())
	g.Expect(forVendor.CustomerOrganizationID).To(HaveValue(Equal(customerOrg)))
	g.Expect(forVendor.DeploymentTargetIDs).To(Equal([]uuid.UUID{customerTarget}),
		"a configuration a vendor manages for a customer is limited like the customer's own")

	ids := func(client *apitest.Client, path string) []uuid.UUID {
		return apitest.IDs(client.Get(path), http.StatusOK)
	}
	g.Expect(ids(vendor.Client, apitest.AlertConfigurations)).To(Equal([]uuid.UUID{vendorConfig.ID}),
		"the vendor's list holds the vendor-level configurations only")
	g.Expect(ids(vendor.Client, withCustomer(apitest.AlertConfigurations, customerOrg))).
		To(ConsistOf(customerConfig.ID, forVendor.ID))
	g.Expect(ids(customer.Client, apitest.AlertConfigurations)).To(ConsistOf(customerConfig.ID, forVendor.ID))
	g.Expect(ids(customer.Client, withCustomer(apitest.AlertConfigurations, customerOrg))).
		To(ConsistOf(customerConfig.ID, forVendor.ID))
	g.Expect(ids(other.Client, apitest.AlertConfigurations)).To(BeEmpty())

	customerPath := apitest.Path(apitest.AlertConfigurations, customerConfig.ID)
	vendorPath := apitest.Path(apitest.AlertConfigurations, vendorConfig.ID)
	body := alertBody("changed", []uuid.UUID{customerTarget}, []uuid.UUID{customer.ID})
	g.Expect(statuses(customer.Client, []request{
		{"list of another customer", http.MethodGet, withCustomer(apitest.AlertConfigurations, otherOrg), nil},
		{"create for another customer", http.MethodPost, apitest.AlertConfigurations, func() object {
			body := alertBody("x", nil, []uuid.UUID{customer.ID})
			body["customerOrganizationId"] = otherOrg
			return body
		}()},
		{"update vendor's", http.MethodPut, vendorPath, body},
		{"delete vendor's", http.MethodDelete, vendorPath, nil},
		{"delete another customer's scope", http.MethodDelete, withCustomer(customerPath, otherOrg), nil},
	})).To(Equal(map[string]int{
		"list of another customer":        http.StatusBadRequest,
		"create for another customer":     http.StatusBadRequest,
		"update vendor's":                 http.StatusNotFound,
		"delete vendor's":                 http.StatusNotFound,
		"delete another customer's scope": http.StatusBadRequest,
	}))
	g.Expect(statuses(other.Client, []request{
		{"update", http.MethodPut, customerPath, body},
		{"delete", http.MethodDelete, customerPath, nil},
	})).To(Equal(map[string]int{"update": http.StatusNotFound, "delete": http.StatusNotFound}))
	g.Expect(statuses(vendor.Client, []request{
		{"update customer's without scope", http.MethodPut, customerPath, body},
		{"delete customer's without scope", http.MethodDelete, customerPath, nil},
	})).To(Equal(map[string]int{
		"update customer's without scope": http.StatusNotFound,
		"delete customer's without scope": http.StatusNotFound,
	}), "a vendor addresses a customer's configuration through the customer scope")

	writes := []request{
		{"create", http.MethodPost, apitest.AlertConfigurations, alertBody("x", nil, nil)},
		{"update", http.MethodPut, customerPath, body},
		{"delete", http.MethodDelete, customerPath, nil},
	}
	g.Expect(statuses(customerReader.Client, writes)).To(Equal(allWithStatus(writes, http.StatusForbidden)))
	g.Expect(statuses(reader.Client, writes)).To(Equal(allWithStatus(writes, http.StatusForbidden)))
	reader.Get(apitest.AlertConfigurations).ExpectStatus(http.StatusOK)
	all := append([]request{{"list", http.MethodGet, apitest.AlertConfigurations, nil}}, writes...)
	g.Expect(statuses(withoutFeature.Client, all)).To(Equal(allWithStatus(all, http.StatusForbidden)),
		"the alerts feature of the customer organization gates the endpoints")
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AlertConfiguration`)).To(Equal(3))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM AlertConfiguration WHERE id = $1`, customerConfig.ID)).
		To(Equal("customer"))

	vendor.Delete(withCustomer(customerPath, customerOrg)).ExpectStatus(http.StatusOK)
	customer.Delete(apitest.Path(apitest.AlertConfigurations, forVendor.ID)).ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AlertConfiguration`)).To(Equal(1))

	vendor.SetSubscriptionType(types.SubscriptionTypeCommunity)
	g.Expect(statuses(vendor.Client, all)).To(Equal(allWithStatus(all, http.StatusForbidden)))
	g.Expect(statuses(customer.Client, all)).To(Equal(allWithStatus(all, http.StatusForbidden)))
}

func TestUpdateNotificationConfiguration(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	reader := vendor.Invite(types.UserRoleReadOnly, nil)
	app := vendor.CreateApplication("app")
	otherApp := vendor.CreateApplication("other")
	customerOrg := vendor.CreateCustomerOrganization("Acme", types.CustomerOrganizationFeatureUpdateNotifications)
	customer := vendor.Invite(types.UserRoleAdmin, &customerOrg)
	otherOrg := vendor.CreateCustomerOrganization("Other", types.CustomerOrganizationFeatureUpdateNotifications)
	other := vendor.Invite(types.UserRoleAdmin, &otherOrg)
	withoutFeature := vendor.Invite(types.UserRoleAdmin, new(vendor.CreateCustomerOrganization("Plain")))
	otherVendor := h.RegisterVendor()
	foreignApp := otherVendor.CreateApplication("foreign")
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		{
			"neither applications nor artifacts", http.MethodPost, apitest.UpdateNotifications,
			updateNotificationBody("x", nil, []uuid.UUID{vendor.ID}),
		},
		{
			"no recipients", http.MethodPost, apitest.UpdateNotifications,
			updateNotificationBody("x", []uuid.UUID{app}, nil),
		},
		{"malformed body", http.MethodPost, apitest.UpdateNotifications, "{"},
		{
			"update malformed id", http.MethodPut, apitest.UpdateNotifications + "/nope",
			updateNotificationBody("x", []uuid.UUID{app}, []uuid.UUID{vendor.ID}),
		},
		{
			"update unknown", http.MethodPut, apitest.Path(apitest.UpdateNotifications, uuid.New()),
			updateNotificationBody("x", []uuid.UUID{app}, []uuid.UUID{vendor.ID}),
		},
		{"delete malformed id", http.MethodDelete, apitest.UpdateNotifications + "/nope", nil},
		{"delete unknown", http.MethodDelete, apitest.Path(apitest.UpdateNotifications, uuid.New()), nil},
	})).To(Equal(map[string]int{
		"neither applications nor artifacts": http.StatusBadRequest,
		"no recipients":                      http.StatusBadRequest,
		"malformed body":                     http.StatusBadRequest,
		"update malformed id":                http.StatusBadRequest,
		"update unknown":                     http.StatusNotFound,
		"delete malformed id":                http.StatusBadRequest,
		"delete unknown":                     http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM UpdateNotificationConfiguration`)).To(BeZero())

	vendorConfig := createUpdateNotification(vendor, updateNotificationBody("vendor",
		[]uuid.UUID{app, foreignApp, uuid.New()}, []uuid.UUID{vendor.ID, customer.ID, otherVendor.ID}))
	g.Expect(vendorConfig.Applications).To(HaveLen(1))
	g.Expect(vendorConfig.Applications[0].ID).To(Equal(app))
	g.Expect(vendorConfig.Recipients).To(HaveLen(2))
	g.Expect(vendorConfig.Recipients).To(
		ContainElement(HaveField("CustomerOrganizationID", HaveValue(Equal(customerOrg)))),
		"a recipient carries the membership that decides what they may see")

	customerConfig := createUpdateNotification(customer, updateNotificationBody("customer",
		[]uuid.UUID{app, otherApp}, []uuid.UUID{customer.ID, vendor.ID, other.ID}))
	g.Expect(customerConfig.Applications).To(BeEmpty(),
		"a customer links only applications it is entitled to, and has no entitlement yet")
	g.Expect(customerConfig.Recipients).To(HaveLen(1))
	g.Expect(customerConfig.Recipients[0].ID).To(Equal(customer.ID))

	createEntitlement(vendor, app, customerOrg)
	customerPath := apitest.Path(apitest.UpdateNotifications, customerConfig.ID)
	updated := apitest.Expect[api.UpdateNotificationConfiguration](customer.Put(customerPath,
		updateNotificationBody("customer", []uuid.UUID{app, otherApp}, []uuid.UUID{customer.ID})), http.StatusOK)
	g.Expect(updated.Applications).To(HaveLen(1))
	g.Expect(updated.Applications[0].ID).To(Equal(app))

	ids := func(client *apitest.Client, path string) []uuid.UUID {
		return apitest.IDs(client.Get(path), http.StatusOK)
	}
	g.Expect(ids(vendor.Client, apitest.UpdateNotifications)).To(Equal([]uuid.UUID{vendorConfig.ID}))
	g.Expect(ids(vendor.Client, withCustomer(apitest.UpdateNotifications, customerOrg))).
		To(Equal([]uuid.UUID{customerConfig.ID}))
	g.Expect(ids(customer.Client, apitest.UpdateNotifications)).To(Equal([]uuid.UUID{customerConfig.ID}))
	g.Expect(ids(other.Client, apitest.UpdateNotifications)).To(BeEmpty())
	g.Expect(ids(otherVendor.Client, apitest.UpdateNotifications)).To(BeEmpty())

	body := updateNotificationBody("changed", []uuid.UUID{app}, []uuid.UUID{customer.ID})
	writes := []request{
		{"create", http.MethodPost, apitest.UpdateNotifications, body},
		{"update", http.MethodPut, customerPath, body},
		{"delete", http.MethodDelete, customerPath, nil},
	}
	g.Expect(statuses(other.Client, writes[1:])).To(Equal(allWithStatus(writes[1:], http.StatusNotFound)))
	g.Expect(statuses(reader.Client, writes)).To(Equal(allWithStatus(writes, http.StatusForbidden)))
	all := append([]request{{"list", http.MethodGet, apitest.UpdateNotifications, nil}}, writes...)
	g.Expect(statuses(withoutFeature.Client, all)).To(Equal(allWithStatus(all, http.StatusForbidden)))
	g.Expect(statuses(vendor.Client, writes[1:])).To(Equal(allWithStatus(writes[1:], http.StatusNotFound)),
		"a vendor addresses a customer's configuration through the customer scope")
	g.Expect(apitest.QueryValue[string](h,
		`SELECT name FROM UpdateNotificationConfiguration WHERE id = $1`, customerConfig.ID)).To(Equal("customer"))

	vendor.Delete(withCustomer(customerPath, customerOrg)).ExpectStatus(http.StatusOK)
	vendor.Delete(apitest.Path(apitest.UpdateNotifications, vendorConfig.ID)).ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM UpdateNotificationConfiguration`)).To(BeZero())
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM UpdateNotificationConfiguration_Application`)).
		To(BeZero())
}

func TestDeploymentStatusAlerts(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	customerOrg := vendor.CreateCustomerOrganization("Acme", types.CustomerOrganizationFeatureAlerts)
	customer := vendor.Invite(types.UserRoleAdmin, &customerOrg)
	vendorTarget := deployDockerTarget(vendor, version, "")
	customerTarget := vendor.CreateDeploymentTarget("customer", &customerOrg)
	_, customerRevision := vendor.Deploy(api.DeploymentRequest{
		DeploymentTargetID: customerTarget, ApplicationVersionID: version,
	})
	createAlert(vendor, alertBody("vendor", []uuid.UUID{vendorTarget.target}, []uuid.UUID{vendor.ID}))
	createAlert(customer, alertBody("customer", []uuid.UUID{customerTarget}, []uuid.UUID{customer.ID}))
	disabled := alertBody("disabled", []uuid.UUID{vendorTarget.target}, []uuid.UUID{vendor.ID})
	disabled["statusTriggerEnabled"] = false
	createAlert(vendor, disabled)
	mails := mailsAfterSetup(h)
	g := NewWithT(t)

	report := func(target uuid.UUID, revision uuid.UUID, statusType types.DeploymentStatusType) {
		controller := h.LoginController(vendor.RequestAccess(target))
		controller.Get(controllerResources).ExpectStatus(http.StatusOK)
		controller.Post(controllerStatus, api.ControllerDeploymentStatus{
			RevisionID: revision, Type: statusType, Message: string(statusType),
		}).ExpectStatus(http.StatusOK)
	}

	report(vendorTarget.target, vendorTarget.revision, types.DeploymentStatusTypeHealthy)
	report(vendorTarget.target, vendorTarget.revision, types.DeploymentStatusTypeError)
	g.Eventually(mails).WithTimeout(10 * time.Second).Should(HaveLen(1))
	mail := mails()[0]
	g.Expect(mail.To).To(Equal([]string{vendor.Email}))
	g.Expect(mail.Subject).To(HavePrefix("[Error] "))
	g.Expect(mail.Subject).To(ContainSubstring("app@1.0.0 (target)"))
	g.Consistently(mails).WithTimeout(time.Second).Should(HaveLen(1),
		"a configuration without the status trigger sends nothing")

	report(vendorTarget.target, vendorTarget.revision, types.DeploymentStatusTypeError)
	report(vendorTarget.target, vendorTarget.revision, types.DeploymentStatusTypeHealthy)
	g.Eventually(mails).WithTimeout(10 * time.Second).Should(HaveLen(2))
	g.Expect(subjects(mails())[1]).To(HavePrefix("[Recovered] "), "a repeated error is not announced again")
	g.Expect(mails()[1].To).To(Equal([]string{vendor.Email}))

	report(customerTarget, customerRevision, types.DeploymentStatusTypeError)
	g.Eventually(mails).WithTimeout(10 * time.Second).Should(HaveLen(3))
	g.Expect(mails()[2].To).To(Equal([]string{customer.Email}))
	g.Expect(mails()[2].Subject).To(ContainSubstring("(Acme customer)"))

	g.Expect(recordTypes(notificationRecords(vendor.Client, apitest.NotificationRecords))).
		To(Equal([]string{"resolved", "alert"}), "the vendor sees the records of the vendor-level configurations")
	g.Expect(recordTypes(notificationRecords(customer.Client, apitest.NotificationRecords))).To(Equal([]string{"alert"}))
	g.Expect(recordTypes(notificationRecords(vendor.Client, withCustomer(apitest.NotificationRecords, customerOrg)))).
		To(Equal([]string{"alert"}))
	record := notificationRecords(customer.Client, apitest.NotificationRecords)[0]
	g.Expect(record.DeploymentTargetID).To(HaveValue(Equal(customerTarget)))
	g.Expect(record.DeploymentStatusMessage).To(HaveValue(Equal("error")))
	g.Expect(record.DeliveryError).To(BeEmpty())
	g.Expect(statuses(customer.Client, []request{
		{"records of another scope", http.MethodGet, withCustomer(apitest.NotificationRecords, uuid.New()), nil},
	})).To(Equal(map[string]int{"records of another scope": http.StatusBadRequest}))
}

func TestUpdateAvailableNotifications(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	first := vendor.CreateApplicationVersion(app, "1.0.0")
	idle := vendor.CreateApplication("idle")
	vendor.CreateApplicationVersion(idle, "1.0.0")
	customerOrg := vendor.CreateCustomerOrganization("Acme", types.CustomerOrganizationFeatureUpdateNotifications)
	customer := vendor.Invite(types.UserRoleAdmin, &customerOrg)
	otherOrg := vendor.CreateCustomerOrganization("Other", types.CustomerOrganizationFeatureUpdateNotifications)
	other := vendor.Invite(types.UserRoleAdmin, &otherOrg)
	vendorTarget := vendor.CreateDeploymentTarget("headquarters", nil)
	customerTarget := vendor.CreateDeploymentTarget("customer", &customerOrg)
	vendor.Deploy(api.DeploymentRequest{DeploymentTargetID: vendorTarget, ApplicationVersionID: first})
	vendor.Deploy(api.DeploymentRequest{DeploymentTargetID: customerTarget, ApplicationVersionID: first})
	createUpdateNotification(vendor, updateNotificationBody("releases", []uuid.UUID{app, idle},
		[]uuid.UUID{vendor.ID, customer.ID, other.ID}))
	mails := mailsAfterSetup(h)
	g := NewWithT(t)

	vendor.CreateApplicationVersion(idle, "2.0.0")
	vendor.CreateApplicationVersion(app, "1.1.0")
	g.Eventually(mails).WithTimeout(10 * time.Second).Should(HaveLen(2))
	g.Consistently(mails).WithTimeout(time.Second).Should(HaveLen(2),
		"a customer without a deployment behind and an application without deployments get no mail")
	g.Expect(subjects(mails())).To(HaveEach(HaveSuffix("Update available: app 1.1.0")))
	recipients := [][]string{mails()[0].To, mails()[1].To}
	g.Expect(recipients).To(ConsistOf([]string{vendor.Email}, []string{customer.Email}))
	for _, mail := range mails() {
		if mail.To[0] == customer.Email {
			g.Expect(mail.HTMLBody).NotTo(ContainSubstring("headquarters"),
				"a customer is told about its own deployments only")
		} else {
			g.Expect(mail.HTMLBody).To(ContainSubstring("headquarters"))
		}
	}

	vendorRecords := notificationRecords(vendor.Client, apitest.NotificationRecords)
	g.Expect(vendorRecords).To(HaveLen(1))
	g.Expect(vendorRecords[0].Type).To(Equal("update_available"))
	g.Expect(vendorRecords[0].ApplicationVersionName).To(HaveValue(Equal("1.1.0")))
	g.Expect(vendorRecords[0].Deployments).To(HaveLen(2))
	customerRecords := notificationRecords(customer.Client, apitest.NotificationRecords)
	g.Expect(customerRecords).To(HaveLen(1))
	g.Expect(customerRecords[0].Deployments).To(HaveLen(1))
	g.Expect(customerRecords[0].Deployments[0].DeploymentTargetName).To(Equal("customer"))
	g.Expect(notificationRecords(other.Client, apitest.NotificationRecords)).To(BeEmpty())
}
