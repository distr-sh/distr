package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func VulnerabilitiesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Vulnerabilities"))

	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", getVulnerabilitiesHandler()).
			With(option.Description("List vulnerabilities")).
			With(option.Request(api.ListVulnerabilitiesRequest{})).
			With(option.Response(http.StatusOK, []api.Vulnerability{}))

		r.With(middleware.RequireVendorOrPartner).
			Get("/tags", getVulnerabilityTagsHandler()).
			With(option.Description("List all vulnerability tags used in the organization")).
			With(option.Response(http.StatusOK, []string{}))

		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", createVulnerabilityHandler()).
			With(option.Description("Create a new vulnerability")).
			With(option.Request(api.CreateVulnerabilityRequest{})).
			With(option.Response(http.StatusOK, api.VulnerabilityDetail{}))

		r.Route("/{vulnerabilityId}", func(r chiopenapi.Router) {
			r.Get("/", getVulnerabilityDetailHandler()).
				With(option.Description("Get vulnerability detail")).
				With(option.Request(api.VulnerabilityIDRequest{})).
				With(option.Response(http.StatusOK, api.VulnerabilityDetail{}))

			// Deliberately not served from the read-only database: the detail view
			// refetches impact right after a status or version change, and replica lag
			// would show the state from before the edit.
			r.With(middleware.RequireVendorOrPartner).
				Get("/impact", getVulnerabilityImpactHandler()).
				With(option.Description("Get the customers affected by this vulnerability")).
				With(option.Request(api.VulnerabilityIDRequest{})).
				With(option.Response(http.StatusOK, api.VulnerabilityImpact{}))

			r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Group(func(r chiopenapi.Router) {
					r.Put("/", updateVulnerabilityHandler()).
						With(option.Description("Update a vulnerability")).
						With(option.Request(struct {
							api.VulnerabilityIDRequest
							api.CreateUpdateVulnerabilityRequest
						}{})).
						With(option.Response(http.StatusOK, api.VulnerabilityDetail{}))

					r.Patch("/status", updateVulnerabilityStatusHandler()).
						With(option.Description("Update the status of a vulnerability")).
						With(option.Request(struct {
							api.VulnerabilityIDRequest
							api.UpdateVulnerabilityStatusRequest
						}{})).
						With(option.Response(http.StatusOK, api.VulnerabilityDetail{}))

					r.Delete("/", deleteVulnerabilityHandler()).
						With(option.Description("Delete a vulnerability")).
						With(option.Request(api.VulnerabilityIDRequest{})).
						With(option.Response(http.StatusNoContent, nil))

					r.Post("/comments", createVulnerabilityCommentHandler()).
						With(option.Description("Add a comment to the vulnerability timeline")).
						With(option.Request(struct {
							api.VulnerabilityIDRequest
							api.CreateVulnerabilityCommentRequest
						}{})).
						With(option.Response(http.StatusOK, api.VulnerabilityEvent{}))
				})
		})
	})
}

func getVulnerabilitiesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		values := r.URL.Query()
		query := api.ListVulnerabilitiesRequest{
			Status:   values["status"],
			Severity: values["severity"],
			Tag:      values["tag"],
		}
		parsed, err := query.Parse()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		filter := db.VulnerabilityFilter{
			CustomerOrgID: a.CurrentCustomerOrgID(),
			Statuses:      parsed.Statuses,
			Severities:    parsed.Severities,
			Tags:          parsed.Tags,
		}

		vulnerabilities, err := db.GetVulnerabilities(ctx, *a.CurrentOrgID(), filter)
		if err != nil {
			log.Error("failed to get vulnerabilities", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(vulnerabilities, mapping.VulnerabilityToAPI))
	}
}

func getVulnerabilityTagsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		tags, err := db.GetVulnerabilityTagNames(ctx, *a.CurrentOrgID())
		if err != nil {
			log.Error("failed to get vulnerability tags", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, tags)
	}
}

func getVulnerabilityDetailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vulnerability := requireVulnerability(w, r)
		if vulnerability == nil {
			return
		}

		detail, ok := buildVulnerabilityDetail(w, r, *vulnerability)
		if !ok {
			return
		}
		RespondJSON(w, detail)
	}
}

func createVulnerabilityHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)
		orgID := *a.CurrentOrgID()
		userID := a.CurrentUserID()

		request, err := JsonBody[api.CreateVulnerabilityRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !validateVulnerabilityVersionsInOrg(w, r, orgID, &request.CreateUpdateVulnerabilityRequest) {
			return
		}

		severity, _ := types.ParseVulnerabilitySeverity(request.Severity)
		vulnerability := types.Vulnerability{
			OrganizationID:         orgID,
			CreatedByUserAccountID: &userID,
			Title:                  request.Title,
			Description:            request.Description,
			Status:                 request.Status,
			Severity:               severity,
			CveID:                  request.CveID,
		}

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.CreateVulnerability(ctx, &vulnerability); err != nil {
				return err
			}
			if err := applyVulnerabilityAssociations(
				ctx, vulnerability.ID, request.CreateUpdateVulnerabilityRequest,
			); err != nil {
				return err
			}
			return db.CreateVulnerabilityEvent(
				ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeCreated, nil)
		})
		if err != nil {
			log.Error("failed to create vulnerability", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondVulnerabilityDetail(w, r, vulnerability.ID)
	}
}

func updateVulnerabilityHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		existing := requireVulnerability(w, r)
		if existing == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)
		orgID := *a.CurrentOrgID()
		userID := a.CurrentUserID()

		request, err := JsonBody[api.CreateUpdateVulnerabilityRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !validateVulnerabilityVersionsInOrg(w, r, orgID, &request) {
			return
		}

		versionsBefore, ok := loadVulnerabilityVersionMarkings(w, r, existing.ID)
		if !ok {
			return
		}

		existingReferences, err := db.GetVulnerabilityReferences(ctx, existing.ID)
		if err != nil {
			log.Error("failed to get vulnerability references", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		referencesAdded, referencesRemoved := referenceChangeMessages(existingReferences, request.References)

		severity, _ := types.ParseVulnerabilitySeverity(request.Severity)
		vulnerability := types.Vulnerability{
			ID:             existing.ID,
			OrganizationID: orgID,
			Title:          request.Title,
			Description:    request.Description,
			Severity:       severity,
			CveID:          request.CveID,
		}

		tagsMessage := tagsChangeMessage(existing.Tags, request.Tags)
		detailsMessage := detailChangeMessage(existing.Vulnerability, request)

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.UpdateVulnerability(ctx, &vulnerability); err != nil {
				return err
			}
			if err := applyVulnerabilityAssociations(ctx, vulnerability.ID, request); err != nil {
				return err
			}
			// Only recorded when something actually changed, so that opening the form and
			// saving it unchanged does not leave a trace that suggests otherwise.
			if detailsMessage != nil {
				if err := db.CreateVulnerabilityEvent(
					ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeEdited, detailsMessage,
				); err != nil {
					return err
				}
			}
			if tagsMessage != nil {
				if err := db.CreateVulnerabilityEvent(
					ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeTagsChanged, tagsMessage,
				); err != nil {
					return err
				}
			}
			if referencesAdded != nil {
				if err := db.CreateVulnerabilityEvent(
					ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeReferenceAdded, referencesAdded,
				); err != nil {
					return err
				}
			}
			if referencesRemoved != nil {
				if err := db.CreateVulnerabilityEvent(
					ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeReferenceRemoved, referencesRemoved,
				); err != nil {
					return err
				}
			}
			// Read back rather than derived from the request, so that the names in the
			// message come from the database instead of being looked up separately.
			versionsAfter, err := vulnerabilityVersionMarkings(ctx, vulnerability.ID)
			if err != nil {
				return err
			}
			if versionsMessage := versionChangeMessage(versionsBefore, versionsAfter); versionsMessage != nil {
				if err := db.CreateVulnerabilityEvent(
					ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeVersionsChanged, versionsMessage,
				); err != nil {
					return err
				}
			}
			return nil
		})
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Error("failed to update vulnerability", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondVulnerabilityDetail(w, r, vulnerability.ID)
	}
}

func updateVulnerabilityStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		existing := requireVulnerability(w, r)
		if existing == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)
		userID := a.CurrentUserID()

		request, err := JsonBody[api.UpdateVulnerabilityStatusRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		status := request.Status
		if !existing.Status.CanTransitionTo(status) {
			http.Error(w, fmt.Sprintf("cannot change status from %v to %v", existing.Status, status),
				http.StatusBadRequest)
			return
		}

		message := fmt.Sprintf("changed status from %v to %v", existing.Status, status)
		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.UpdateVulnerabilityStatus(
				ctx, existing.ID, existing.OrganizationID, status,
			); err != nil {
				return err
			}
			return db.CreateVulnerabilityEvent(
				ctx, existing.ID, &userID, types.VulnerabilityEventTypeStatusChanged, &message)
		})
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Error("failed to update vulnerability status", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondVulnerabilityDetail(w, r, existing.ID)
	}
}

func deleteVulnerabilityHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vulnerability := requireVulnerability(w, r)
		if vulnerability == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		err := db.DeleteVulnerability(ctx, vulnerability.ID, vulnerability.OrganizationID)
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if err != nil {
			log.Error("failed to delete vulnerability", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func createVulnerabilityCommentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vulnerability := requireVulnerability(w, r)
		if vulnerability == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.CreateVulnerabilityCommentRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		event, err := db.CreateVulnerabilityCommentEvent(
			ctx, vulnerability.ID, a.CurrentUserID(), request.Content)
		if err != nil {
			log.Error("failed to create vulnerability comment", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.VulnerabilityEventToAPI(*event))
	}
}

func getVulnerabilityImpactHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vulnerability := requireVulnerability(w, r)
		if vulnerability == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		deployments, err := db.GetVulnerabilityImpactedDeployments(
			ctx, vulnerability.ID, vulnerability.OrganizationID)
		if err != nil {
			log.Error("failed to get vulnerability impacted deployments", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pulls, err := db.GetVulnerabilityImpactedPulls(ctx, vulnerability.ID, vulnerability.OrganizationID)
		if err != nil {
			log.Error("failed to get vulnerability impacted pulls", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, api.VulnerabilityImpact{
			Deployments: mapping.List(deployments, mapping.VulnerabilityImpactedDeploymentToAPI),
			Pulls:       mapping.List(pulls, mapping.VulnerabilityImpactedPullToAPI),
		})
	}
}

// requireVulnerability parses the vulnerability ID from the path and loads it scoped to the
// current organization, applying the customer entitlement filter for customer users.
// Returns nil if an error response was already written.
func requireVulnerability(w http.ResponseWriter, r *http.Request) *types.VulnerabilityWithDetails {
	id, err := uuid.Parse(r.PathValue("vulnerabilityId"))
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	vulnerability, err := db.GetVulnerabilityByID(ctx, id, *a.CurrentOrgID(), a.CurrentCustomerOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
		return nil
	} else if err != nil {
		log.Error("failed to get vulnerability", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	return vulnerability
}

func respondVulnerabilityDetail(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	vulnerability, err := db.GetVulnerabilityByID(ctx, id, *a.CurrentOrgID(), a.CurrentCustomerOrgID())
	if err != nil {
		log.Error("failed to get vulnerability", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	detail, ok := buildVulnerabilityDetail(w, r, *vulnerability)
	if !ok {
		return
	}
	RespondJSON(w, detail)
}

// buildVulnerabilityDetail loads the child collections of a vulnerability. The event
// timeline is vendor-internal and is therefore omitted for customer users.
func buildVulnerabilityDetail(
	w http.ResponseWriter, r *http.Request, vulnerability types.VulnerabilityWithDetails,
) (api.VulnerabilityDetail, bool) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	fail := func(message string, err error) (api.VulnerabilityDetail, bool) {
		log.Error(message, zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return api.VulnerabilityDetail{}, false
	}

	references, err := db.GetVulnerabilityReferences(ctx, vulnerability.ID)
	if err != nil {
		return fail("failed to get vulnerability references", err)
	}

	applicationVersions, err := db.GetVulnerabilityApplicationVersions(ctx, vulnerability.ID)
	if err != nil {
		return fail("failed to get vulnerability application versions", err)
	}

	artifactVersions, err := db.GetVulnerabilityArtifactVersions(ctx, vulnerability.ID)
	if err != nil {
		return fail("failed to get vulnerability artifact versions", err)
	}

	events := []types.VulnerabilityEventWithUser{}
	if a.CurrentCustomerOrgID() == nil {
		events, err = db.GetVulnerabilityEvents(ctx, vulnerability.ID)
		if err != nil {
			return fail("failed to get vulnerability events", err)
		}
	}

	return api.VulnerabilityDetail{
		Vulnerability:       mapping.VulnerabilityToAPI(vulnerability),
		Description:         vulnerability.Description,
		References:          mapping.List(references, mapping.VulnerabilityReferenceToAPI),
		ApplicationVersions: mapping.List(applicationVersions, mapping.VulnerabilityApplicationVersionToAPI),
		ArtifactVersions:    mapping.List(artifactVersions, mapping.VulnerabilityArtifactVersionToAPI),
		Events:              mapping.List(events, mapping.VulnerabilityEventToAPI),
	}, true
}

func applyVulnerabilityAssociations(
	ctx context.Context, vulnerabilityID uuid.UUID, request api.CreateUpdateVulnerabilityRequest,
) error {
	if err := db.SetVulnerabilityTags(ctx, vulnerabilityID, request.Tags); err != nil {
		return err
	}
	references := make([]types.VulnerabilityReference, len(request.References))
	for i, reference := range request.References {
		references[i] = types.VulnerabilityReference{URL: reference.URL, Label: reference.Label}
	}
	if err := db.SetVulnerabilityReferences(ctx, vulnerabilityID, references); err != nil {
		return err
	}
	return db.SetVulnerabilityVersions(ctx, vulnerabilityID, db.VulnerabilityVersionSelection{
		AffectedApplicationVersionIDs: request.AffectedApplicationVersionIDs,
		FixedApplicationVersionIDs:    request.FixedApplicationVersionIDs,
		AffectedArtifactVersionIDs:    request.AffectedArtifactVersionIDs,
		FixedArtifactVersionIDs:       request.FixedArtifactVersionIDs,
	})
}

// validateVulnerabilityVersionsInOrg rejects a request that references versions belonging to
// another organization. Returns false if an error response was already written.
func validateVulnerabilityVersionsInOrg(
	w http.ResponseWriter, r *http.Request, orgID uuid.UUID, request *api.CreateUpdateVulnerabilityRequest,
) bool {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	applicationVersionIDs := slices.Concat(
		request.AffectedApplicationVersionIDs, request.FixedApplicationVersionIDs)
	artifactVersionIDs := slices.Concat(
		request.AffectedArtifactVersionIDs, request.FixedArtifactVersionIDs)

	count, err := db.CountVulnerabilityVersionsOutsideOrg(ctx, orgID, applicationVersionIDs, artifactVersionIDs)
	if err != nil {
		log.Error("failed to validate vulnerability versions", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if count > 0 {
		http.Error(w, "one or more versions do not exist in this organization", http.StatusBadRequest)
		return false
	}
	return true
}

// versionMarking is an application or artifact version together with the relation it has to
// the vulnerability, reduced to what a timeline message needs.
type versionMarking struct {
	id       uuid.UUID
	label    string
	relation types.VulnerabilityVersionRelation
}

// vulnerabilityVersionMarkings reads the currently marked application and artifact versions
// as a single list.
func vulnerabilityVersionMarkings(ctx context.Context, vulnerabilityID uuid.UUID) ([]versionMarking, error) {
	applicationVersions, err := db.GetVulnerabilityApplicationVersions(ctx, vulnerabilityID)
	if err != nil {
		return nil, err
	}
	artifactVersions, err := db.GetVulnerabilityArtifactVersions(ctx, vulnerabilityID)
	if err != nil {
		return nil, err
	}

	markings := make([]versionMarking, 0, len(applicationVersions)+len(artifactVersions))
	for _, version := range applicationVersions {
		markings = append(markings, versionMarking{
			id:       version.ApplicationVersionID,
			label:    version.ApplicationName + " " + version.ApplicationVersionName,
			relation: version.Relation,
		})
	}
	for _, version := range artifactVersions {
		markings = append(markings, versionMarking{
			id:       version.ArtifactVersionID,
			label:    version.ArtifactName + " " + version.ArtifactVersionName,
			relation: version.Relation,
		})
	}
	return markings, nil
}

func loadVulnerabilityVersionMarkings(
	w http.ResponseWriter, r *http.Request, vulnerabilityID uuid.UUID,
) ([]versionMarking, bool) {
	ctx := r.Context()
	markings, err := vulnerabilityVersionMarkings(ctx, vulnerabilityID)
	if err != nil {
		internalctx.GetLogger(ctx).Error("failed to get vulnerability versions", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	return markings, true
}

// maxVersionChangeMessageParts bounds the timeline message: marking a hundred versions at
// once must not write a message nobody can read.
const maxVersionChangeMessageParts = 10

// versionChangeMessage describes how the affected and fixed markings changed, or returns nil
// when they are the same. Versions are matched by id, so one that switches between affected
// and fixed reads as a change rather than as a removal plus an addition.
func versionChangeMessage(before, after []versionMarking) *string {
	beforeByID := make(map[uuid.UUID]versionMarking, len(before))
	for _, marking := range before {
		beforeByID[marking.id] = marking
	}
	afterByID := make(map[uuid.UUID]versionMarking, len(after))
	for _, marking := range after {
		afterByID[marking.id] = marking
	}

	var parts []string
	for _, marking := range after {
		previous, existed := beforeByID[marking.id]
		if !existed {
			parts = append(parts, fmt.Sprintf("marked %v as %v", marking.label, marking.relation))
		} else if previous.relation != marking.relation {
			parts = append(parts, fmt.Sprintf(
				"changed %v from %v to %v", marking.label, previous.relation, marking.relation))
		}
	}
	for _, marking := range before {
		if _, exists := afterByID[marking.id]; !exists {
			parts = append(parts, fmt.Sprintf("unmarked %v", marking.label))
		}
	}

	if len(parts) == 0 {
		return nil
	}
	if len(parts) > maxVersionChangeMessageParts {
		remaining := len(parts) - maxVersionChangeMessageParts
		parts = append(parts[:maxVersionChangeMessageParts], fmt.Sprintf("and %v more", remaining))
	}

	message := strings.Join(parts, "; ")
	return &message
}

// detailChangeMessage describes which of the editable detail fields changed, or returns nil
// when none did. The description is only reported as changed: a diff of free-form Markdown
// does not belong in a one-line timeline entry.
func detailChangeMessage(before types.Vulnerability, after api.CreateUpdateVulnerabilityRequest) *string {
	var parts []string

	if before.Title != after.Title {
		parts = append(parts, fmt.Sprintf("changed the title from %q to %q", before.Title, after.Title))
	}
	if string(before.Severity) != after.Severity {
		parts = append(parts, fmt.Sprintf("changed the severity from %v to %v", before.Severity, after.Severity))
	}
	if message := cveChangeMessage(before.CveID, after.CveID); message != "" {
		parts = append(parts, message)
	}
	if before.Description != after.Description {
		parts = append(parts, "updated the description")
	}

	if len(parts) == 0 {
		return nil
	}
	message := strings.Join(parts, "; ")
	return &message
}

func cveChangeMessage(before, after *string) string {
	switch {
	case before == nil && after == nil:
		return ""
	case before == nil:
		return fmt.Sprintf("set the CVE ID to %v", *after)
	case after == nil:
		return fmt.Sprintf("removed the CVE ID %v", *before)
	case *before != *after:
		return fmt.Sprintf("changed the CVE ID from %v to %v", *before, *after)
	default:
		return ""
	}
}

// referenceChangeMessages compares the reference list of an update request against the stored
// one and returns the message for the reference_added and the reference_removed event, or nil
// where no event should be recorded. References are identified by their URL, which is what
// makes a reference the same reference to a reader.
func referenceChangeMessages(
	before []types.VulnerabilityReference, after []api.VulnerabilityReference,
) (added, removed *string) {
	beforeURLs := make([]string, len(before))
	for i, reference := range before {
		beforeURLs[i] = reference.URL
	}
	afterURLs := make([]string, len(after))
	for i, reference := range after {
		afterURLs[i] = reference.URL
	}

	var addedURLs, removedURLs []string
	for _, url := range afterURLs {
		if !slices.Contains(beforeURLs, url) {
			addedURLs = append(addedURLs, url)
		}
	}
	for _, url := range beforeURLs {
		if !slices.Contains(afterURLs, url) {
			removedURLs = append(removedURLs, url)
		}
	}

	if len(addedURLs) > 0 {
		message := "added " + strings.Join(addedURLs, ", ")
		added = &message
	}
	if len(removedURLs) > 0 {
		message := "removed " + strings.Join(removedURLs, ", ")
		removed = &message
	}
	return added, removed
}

// tagsChangeMessage describes a tag change for the timeline, or returns nil when the tag set
// is unchanged.
func tagsChangeMessage(before, after []string) *string {
	var added, removed []string
	for _, tag := range after {
		if !slices.Contains(before, tag) {
			added = append(added, tag)
		}
	}
	for _, tag := range before {
		if !slices.Contains(after, tag) {
			removed = append(removed, tag)
		}
	}
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}

	var parts []string
	if len(added) > 0 {
		parts = append(parts, "added "+strings.Join(added, ", "))
	}
	if len(removed) > 0 {
		parts = append(parts, "removed "+strings.Join(removed, ", "))
	}
	message := strings.Join(parts, " and ")
	return &message
}
