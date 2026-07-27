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
		type ListVulnerabilitiesRequest struct {
			Status   string `query:"status"`
			Severity string `query:"severity"`
			Tag      string `query:"tag"`
		}

		r.Get("/", getVulnerabilitiesHandler()).
			With(option.Description("List vulnerabilities")).
			With(option.Request(ListVulnerabilitiesRequest{})).
			With(option.Response(http.StatusOK, []api.Vulnerability{}))

		r.With(middleware.RequireVendorOrPartner).
			Get("/tags", getVulnerabilityTagsHandler()).
			With(option.Description("List all vulnerability tags used in the organization")).
			With(option.Response(http.StatusOK, []string{}))

		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", createVulnerabilityHandler()).
			With(option.Description("Create a new vulnerability")).
			With(option.Request(api.CreateUpdateVulnerabilityRequest{})).
			With(option.Response(http.StatusOK, api.VulnerabilityDetail{}))

		r.Route("/{vulnerabilityId}", func(r chiopenapi.Router) {
			type VulnerabilityIDRequest struct {
				VulnerabilityID uuid.UUID `path:"vulnerabilityId"`
			}

			r.Get("/", getVulnerabilityDetailHandler()).
				With(option.Description("Get vulnerability detail")).
				With(option.Request(VulnerabilityIDRequest{})).
				With(option.Response(http.StatusOK, api.VulnerabilityDetail{}))

			r.With(middleware.RequireVendorOrPartner, middleware.UseReadonlyDB).
				Get("/impact", getVulnerabilityImpactHandler()).
				With(option.Description("Get the customers affected by this vulnerability")).
				With(option.Request(VulnerabilityIDRequest{})).
				With(option.Response(http.StatusOK, api.VulnerabilityImpact{}))

			r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Group(func(r chiopenapi.Router) {
					r.Put("/", updateVulnerabilityHandler()).
						With(option.Description("Update a vulnerability")).
						With(option.Request(struct {
							VulnerabilityIDRequest
							api.CreateUpdateVulnerabilityRequest
						}{})).
						With(option.Response(http.StatusOK, api.VulnerabilityDetail{}))

					r.Patch("/status", updateVulnerabilityStatusHandler()).
						With(option.Description("Update the status of a vulnerability")).
						With(option.Request(struct {
							VulnerabilityIDRequest
							api.UpdateVulnerabilityStatusRequest
						}{})).
						With(option.Response(http.StatusOK, api.VulnerabilityDetail{}))

					r.Delete("/", deleteVulnerabilityHandler()).
						With(option.Description("Delete a vulnerability")).
						With(option.Request(VulnerabilityIDRequest{}))

					r.Post("/comments", createVulnerabilityCommentHandler()).
						With(option.Description("Add a comment to the vulnerability timeline")).
						With(option.Request(struct {
							VulnerabilityIDRequest
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

		filter := db.VulnerabilityFilter{CustomerOrgID: a.CurrentCustomerOrgID()}
		if value := r.URL.Query().Get("status"); value != "" {
			status, err := types.ParseVulnerabilityStatus(value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			filter.Status = &status
		}
		if value := r.URL.Query().Get("severity"); value != "" {
			severity, err := types.ParseVulnerabilitySeverity(value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			filter.Severity = &severity
		}
		if value := r.URL.Query().Get("tag"); value != "" {
			filter.Tag = &value
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

		severity, _ := types.ParseVulnerabilitySeverity(request.Severity)
		vulnerability := types.Vulnerability{
			OrganizationID:         orgID,
			CreatedByUserAccountID: &userID,
			Title:                  request.Title,
			Description:            request.Description,
			Severity:               severity,
			CveID:                  request.CveID,
		}

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.CreateVulnerability(ctx, &vulnerability); err != nil {
				return err
			}
			if err := applyVulnerabilityAssociations(ctx, vulnerability.ID, request); err != nil {
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

		versionsChanged, ok := vulnerabilityVersionsChanged(w, r, existing.ID, request)
		if !ok {
			return
		}

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

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.UpdateVulnerability(ctx, &vulnerability); err != nil {
				return err
			}
			if err := applyVulnerabilityAssociations(ctx, vulnerability.ID, request); err != nil {
				return err
			}
			if err := db.CreateVulnerabilityEvent(
				ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeEdited, nil,
			); err != nil {
				return err
			}
			if tagsMessage != nil {
				if err := db.CreateVulnerabilityEvent(
					ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeTagsChanged, tagsMessage,
				); err != nil {
					return err
				}
			}
			if versionsChanged {
				if err := db.CreateVulnerabilityEvent(
					ctx, vulnerability.ID, &userID, types.VulnerabilityEventTypeVersionsChanged, nil,
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
		}

		status, err := types.ParseVulnerabilityStatus(request.Status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

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

func vulnerabilityVersionsChanged(
	w http.ResponseWriter, r *http.Request, vulnerabilityID uuid.UUID, request api.CreateUpdateVulnerabilityRequest,
) (bool, bool) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	applicationVersions, err := db.GetVulnerabilityApplicationVersions(ctx, vulnerabilityID)
	if err != nil {
		log.Error("failed to get vulnerability application versions", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false, false
	}
	artifactVersions, err := db.GetVulnerabilityArtifactVersions(ctx, vulnerabilityID)
	if err != nil {
		log.Error("failed to get vulnerability artifact versions", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false, false
	}

	var currentAffectedApps, currentFixedApps []uuid.UUID
	for _, version := range applicationVersions {
		if version.Relation == types.VulnerabilityVersionRelationAffected {
			currentAffectedApps = append(currentAffectedApps, version.ApplicationVersionID)
		} else {
			currentFixedApps = append(currentFixedApps, version.ApplicationVersionID)
		}
	}
	var currentAffectedArtifacts, currentFixedArtifacts []uuid.UUID
	for _, version := range artifactVersions {
		if version.Relation == types.VulnerabilityVersionRelationAffected {
			currentAffectedArtifacts = append(currentAffectedArtifacts, version.ArtifactVersionID)
		} else {
			currentFixedArtifacts = append(currentFixedArtifacts, version.ArtifactVersionID)
		}
	}

	changed := !sameIDs(currentAffectedApps, request.AffectedApplicationVersionIDs) ||
		!sameIDs(currentFixedApps, request.FixedApplicationVersionIDs) ||
		!sameIDs(currentAffectedArtifacts, request.AffectedArtifactVersionIDs) ||
		!sameIDs(currentFixedArtifacts, request.FixedArtifactVersionIDs)
	return changed, true
}

func sameIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := slices.Clone(a)
	sortedB := slices.Clone(b)
	slices.SortFunc(sortedA, func(x, y uuid.UUID) int { return strings.Compare(x.String(), y.String()) })
	slices.SortFunc(sortedB, func(x, y uuid.UUID) int { return strings.Compare(x.String(), y.String()) })
	return slices.Equal(sortedA, sortedB)
}

// tagsChangeMessage describes a tag change for the timeline, or returns nil when the tag
// set is unchanged.
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
