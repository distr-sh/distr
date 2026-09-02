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
	"github.com/distr-sh/distr/internal/authn/authinfo"
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

const duplicateCveIDMessage = "another advisory in this organization already covers this CVE ID"

func advisoryScope(a authinfo.AuthInfoWithUserAndOrganization) db.AdvisoryScope {
	return db.AdvisoryScope{
		PartnerOrgID:  a.CurrentPartnerOrgID(),
		CustomerOrgID: a.CurrentCustomerOrgID(),
	}
}

func AdvisoriesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Advisories"))

	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", getAdvisoriesHandler()).
			With(option.Description("List advisories")).
			With(option.Request(api.ListAdvisoriesRequest{})).
			With(option.Response(http.StatusOK, []api.Advisory{}))

		r.With(middleware.RequireVendor).
			Get("/tags", getAdvisoryTagsHandler()).
			With(option.Description("List all advisory tags used in the organization")).
			With(option.Response(http.StatusOK, []string{}))

		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", createAdvisoryHandler()).
			With(option.Description("Create a new advisory")).
			With(option.Request(api.CreateUpdateAdvisoryRequest{})).
			With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

		r.Route("/{advisoryId}", func(r chiopenapi.Router) {
			r.Get("/", getAdvisoryDetailHandler()).
				With(option.Description("Get advisory detail")).
				With(option.Request(api.AdvisoryIDRequest{})).
				With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

			// Deliberately not served from the read-only database: the detail view refetches
			// impact right after an edit, and replica lag would show the state from before it.
			r.Get("/impact", getAdvisoryImpactHandler()).
				With(option.Description("Get the deployments and downloads affected by this advisory")).
				With(option.Request(api.AdvisoryIDRequest{})).
				With(option.Response(http.StatusOK, api.AdvisoryImpact{}))

			r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Group(func(r chiopenapi.Router) {
					r.Put("/", updateAdvisoryHandler()).
						With(option.Description("Update an advisory")).
						With(option.Request(struct {
							api.AdvisoryIDRequest
							api.CreateUpdateAdvisoryRequest
						}{})).
						With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

					r.Patch("/", patchAdvisoryHandler()).
						With(option.Description("Change the status or severity of an advisory")).
						With(option.Request(struct {
							api.AdvisoryIDRequest
							api.PatchAdvisoryRequest
						}{})).
						With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

					r.Post("/comments", createAdvisoryCommentHandler()).
						With(option.Description("Add a comment to the advisory timeline")).
						With(option.Request(struct {
							api.AdvisoryIDRequest
							api.CreateAdvisoryCommentRequest
						}{})).
						With(option.Response(http.StatusOK, api.AdvisoryEvent{}))
				})
		})
	})
}

func getAdvisoriesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		values := r.URL.Query()
		query := api.ListAdvisoriesRequest{
			Status:   values["status"],
			Severity: values["severity"],
			Tag:      values["tag"],
		}
		parsed, err := query.Parse()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		filter := db.AdvisoryFilter{
			Scope:      advisoryScope(a),
			Statuses:   parsed.Statuses,
			Severities: parsed.Severities,
			Tags:       parsed.Tags,
		}

		advisories, err := db.GetAdvisories(ctx, *a.CurrentOrgID(), filter)
		if err != nil {
			log.Error("failed to get advisories", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(
			advisories,
			mapping.AdvisoryToAPI(a.CurrentCustomerOrgID(), a.CurrentPartnerOrgID()),
		))
	}
}

func getAdvisoryTagsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		tags, err := db.GetAdvisoryTagNames(ctx, *a.CurrentOrgID())
		if err != nil {
			log.Error("failed to get advisory tags", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, tags)
	}
}

func getAdvisoryDetailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		advisory := requireAdvisory(w, r)
		if advisory == nil {
			return
		}

		detail, ok := buildAdvisoryDetail(w, r, *advisory)
		if !ok {
			return
		}
		RespondJSON(w, detail)
	}
}

func createAdvisoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)
		orgID := *a.CurrentOrgID()
		userID := a.CurrentUserID()

		request, err := JsonBody[api.CreateUpdateAdvisoryRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !validateAdvisoryVersionsInOrg(w, r, orgID, &request) {
			return
		}

		status := request.Status
		if status == "" {
			status = types.AdvisoryStatusTriage
		}
		severity, _ := types.ParseAdvisorySeverity(request.Severity)
		advisory := types.Advisory{
			OrganizationID:         orgID,
			CreatedByUserAccountID: &userID,
			Title:                  request.Title,
			Description:            request.Description,
			Status:                 status,
			Severity:               severity,
			CveID:                  request.CveID,
		}

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.CreateAdvisory(ctx, &advisory); err != nil {
				return err
			}
			if err := applyAdvisoryAssociations(ctx, advisory.ID, request); err != nil {
				return err
			}
			return recordAdvisoryPublication(ctx, &advisory, userID)
		})
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, duplicateCveIDMessage, http.StatusConflict)
			return
		} else if err != nil {
			log.Error("failed to create advisory", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondAdvisoryDetail(w, r, advisory.ID)
	}
}

func updateAdvisoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		existing := requireAdvisory(w, r)
		if existing == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)
		orgID := *a.CurrentOrgID()
		userID := a.CurrentUserID()

		request, err := JsonBody[api.CreateUpdateAdvisoryRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !validateAdvisoryVersionsInOrg(w, r, orgID, &request) {
			return
		}

		// An advisory that has not been disclosed yet records nothing, so the change detection
		// feeding the timeline message is only worth running once it has been.
		wasPublished := existing.PublishedAt != nil
		var changes []string
		var versionsBefore []versionMarking
		if wasPublished {
			var ok bool
			if versionsBefore, ok = loadAdvisoryVersionMarkings(w, r, existing.ID); !ok {
				return
			}

			existingReferences, err := db.GetAdvisoryReferences(ctx, existing.ID)
			if err != nil {
				log.Error("failed to get advisory references", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			changes = slices.Concat(
				detailChangeParts(existing.Advisory, request),
				tagChangeParts(existing.Tags, request.Tags),
				referenceChangeParts(existingReferences, request.References),
			)
		}

		severity, _ := types.ParseAdvisorySeverity(request.Severity)
		advisory := types.Advisory{
			ID:             existing.ID,
			OrganizationID: orgID,
			Title:          request.Title,
			Description:    request.Description,
			Status:         existing.Status,
			Severity:       severity,
			CveID:          request.CveID,
		}
		if request.Status != "" {
			advisory.Status = request.Status
		}

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.UpdateAdvisory(ctx, &advisory); err != nil {
				return err
			}
			if err := applyAdvisoryAssociations(ctx, advisory.ID, request); err != nil {
				return err
			}
			if !wasPublished {
				return recordAdvisoryPublication(ctx, &advisory, userID)
			}
			// Read back rather than derived from the request, so that the names in the message
			// come from the database instead of being looked up separately.
			versionsAfter, err := advisoryVersionMarkings(ctx, advisory.ID, advisory.OrganizationID)
			if err != nil {
				return err
			}
			// One entry for the whole save, so that an edit reads as the single action it was,
			// and none at all when the save changed nothing.
			parts := slices.Concat(changes, versionChangeParts(versionsBefore, versionsAfter))
			if len(parts) > 0 {
				message := strings.Join(parts, "; ")
				if err := db.CreateAdvisoryEvent(
					ctx, advisory.ID, &userID, types.AdvisoryEventTypeEdited, &message,
				); err != nil {
					return err
				}
			}
			if advisory.Status != existing.Status {
				return createAdvisoryStatusEvent(ctx, advisory.ID, userID, existing.Status, advisory.Status)
			}
			return nil
		})
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, duplicateCveIDMessage, http.StatusConflict)
			return
		} else if err != nil {
			log.Error("failed to update advisory", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondAdvisoryDetail(w, r, advisory.ID)
	}
}

func patchAdvisoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		existing := requireAdvisory(w, r)
		if existing == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)
		userID := a.CurrentUserID()

		request, err := JsonBody[api.PatchAdvisoryRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		wasPublished := existing.PublishedAt != nil
		statusChanged := request.Status != nil && *request.Status != existing.Status
		severityChanged := request.Severity != nil && *request.Severity != existing.Severity

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			patched, err := db.PatchAdvisory(
				ctx, existing.ID, existing.OrganizationID, request.Status, request.Severity,
			)
			if err != nil {
				return err
			}
			if !wasPublished {
				return recordAdvisoryPublication(ctx, patched, userID)
			}
			if severityChanged {
				message := fmt.Sprintf("changed the severity from %v to %v", existing.Severity, *request.Severity)
				if err := db.CreateAdvisoryEvent(
					ctx, existing.ID, &userID, types.AdvisoryEventTypeEdited, &message,
				); err != nil {
					return err
				}
			}
			if statusChanged {
				return createAdvisoryStatusEvent(ctx, existing.ID, userID, existing.Status, *request.Status)
			}
			return nil
		})
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Error("failed to patch advisory", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondAdvisoryDetail(w, r, existing.ID)
	}
}

func createAdvisoryStatusEvent(
	ctx context.Context, advisoryID, userID uuid.UUID, before, after types.AdvisoryStatus,
) error {
	message := fmt.Sprintf("changed status from %v to %v", before, after)
	return db.CreateAdvisoryEvent(ctx, advisoryID, &userID, types.AdvisoryEventTypeStatusChanged, &message)
}

// recordAdvisoryPublication opens the timeline at the disclosure, because an advisory under
// triage is rewritten until it is right and none of that is worth reading afterwards. Callers
// reach this only while the advisory had not been published before, so one that is still being
// drafted records nothing at all. Comments are the exception and are always kept.
func recordAdvisoryPublication(ctx context.Context, advisory *types.Advisory, userID uuid.UUID) error {
	if advisory.PublishedAt == nil {
		return nil
	}
	return db.CreateAdvisoryEvent(ctx, advisory.ID, &userID, types.AdvisoryEventTypePublished, nil)
}

func createAdvisoryCommentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		advisory := requireAdvisory(w, r)
		if advisory == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.CreateAdvisoryCommentRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		event, err := db.CreateAdvisoryCommentEvent(
			ctx, advisory.ID, a.CurrentUserID(), request.Content)
		if err != nil {
			log.Error("failed to create advisory comment", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.AdvisoryEventToAPI(*event))
	}
}

func getAdvisoryImpactHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		advisory := requireAdvisory(w, r)
		if advisory == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		// Customers reach this endpoint too, to see whether their own deployments still run an
		// affected version. The scope keeps every other customer's rows out of the response.
		scope := advisoryScope(a)

		deployments, err := db.GetAdvisoryImpactedDeployments(ctx, advisory.ID, advisory.OrganizationID, scope)
		if err != nil {
			log.Error("failed to get advisory impacted deployments", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pulls, err := db.GetAdvisoryImpactedPulls(ctx, advisory.ID, advisory.OrganizationID, scope)
		if err != nil {
			log.Error("failed to get advisory impacted pulls", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, api.AdvisoryImpact{
			Deployments: mapping.List(deployments, mapping.AdvisoryImpactedDeploymentToAPI),
			Pulls:       mapping.List(pulls, mapping.AdvisoryImpactedPullToAPI),
		})
	}
}

// requireAdvisory returns nil if an error response was already written.
func requireAdvisory(w http.ResponseWriter, r *http.Request) *types.AdvisoryWithDetails {
	id, err := uuid.Parse(r.PathValue("advisoryId"))
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	advisory, err := db.GetAdvisoryByID(ctx, id, *a.CurrentOrgID(), advisoryScope(a))
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
		return nil
	} else if err != nil {
		log.Error("failed to get advisory", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	return advisory
}

func respondAdvisoryDetail(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	advisory, err := db.GetAdvisoryByID(ctx, id, *a.CurrentOrgID(), advisoryScope(a))
	if err != nil {
		log.Error("failed to get advisory", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	detail, ok := buildAdvisoryDetail(w, r, *advisory)
	if !ok {
		return
	}
	RespondJSON(w, detail)
}

// buildAdvisoryDetail omits the event timeline for customer and partner users, since it is
// vendor-internal.
func buildAdvisoryDetail(
	w http.ResponseWriter, r *http.Request, advisory types.AdvisoryWithDetails,
) (api.AdvisoryDetail, bool) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	fail := func(message string, err error) (api.AdvisoryDetail, bool) {
		log.Error(message, zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return api.AdvisoryDetail{}, false
	}

	references, err := db.GetAdvisoryReferences(ctx, advisory.ID)
	if err != nil {
		return fail("failed to get advisory references", err)
	}

	versions, err := db.GetAdvisoryVersions(ctx, []uuid.UUID{advisory.ID}, *a.CurrentOrgID(), advisoryScope(a))
	if err != nil {
		return fail("failed to get advisory versions", err)
	}

	events := []types.AdvisoryEventWithUser{}
	if a.CurrentCustomerOrgID() == nil && a.CurrentPartnerOrgID() == nil {
		events, err = db.GetAdvisoryEvents(ctx, advisory.ID)
		if err != nil {
			return fail("failed to get advisory events", err)
		}
	}

	return api.AdvisoryDetail{
		Advisory:            mapping.AdvisoryToAPI(a.CurrentCustomerOrgID(), a.CurrentPartnerOrgID())(advisory),
		Description:         advisory.Description,
		References:          mapping.List(references, mapping.AdvisoryReferenceToAPI),
		ApplicationVersions: mapping.List(versions.ApplicationVersions, mapping.AdvisoryApplicationVersionToAPI),
		ArtifactVersions:    mapping.List(versions.ArtifactVersions, mapping.AdvisoryArtifactVersionToAPI),
		Events:              mapping.List(events, mapping.AdvisoryEventToAPI),
	}, true
}

func applyAdvisoryAssociations(
	ctx context.Context, advisoryID uuid.UUID, request api.CreateUpdateAdvisoryRequest,
) error {
	if err := db.SetAdvisoryTags(ctx, advisoryID, request.Tags); err != nil {
		return err
	}
	references := make([]types.AdvisoryReference, len(request.References))
	for i, reference := range request.References {
		references[i] = types.AdvisoryReference{URL: reference.URL, Label: reference.Label}
	}
	if err := db.SetAdvisoryReferences(ctx, advisoryID, references); err != nil {
		return err
	}
	return db.SetAdvisoryVersions(ctx, advisoryID, db.AdvisoryVersionSelection{
		AffectedApplicationVersionIDs: request.AffectedApplicationVersionIDs,
		PatchedApplicationVersionIDs:  request.PatchedApplicationVersionIDs,
		AffectedArtifactVersionIDs:    request.AffectedArtifactVersionIDs,
		PatchedArtifactVersionIDs:     request.PatchedArtifactVersionIDs,
	})
}

// validateAdvisoryVersionsInOrg returns false if an error response was already written.
func validateAdvisoryVersionsInOrg(
	w http.ResponseWriter, r *http.Request, orgID uuid.UUID, request *api.CreateUpdateAdvisoryRequest,
) bool {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	applicationVersionIDs := slices.Concat(
		request.AffectedApplicationVersionIDs, request.PatchedApplicationVersionIDs)
	artifactVersionIDs := slices.Concat(
		request.AffectedArtifactVersionIDs, request.PatchedArtifactVersionIDs)

	count, err := db.CountAdvisoryVersionsOutsideOrg(ctx, orgID, applicationVersionIDs, artifactVersionIDs)
	if err != nil {
		log.Error("failed to validate advisory versions", zap.Error(err))
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

type versionMarking struct {
	id       uuid.UUID
	label    string
	relation types.AdvisoryVersionRelation
}

// advisoryVersionMarkings queries with an empty scope on purpose: the markings feed the
// timeline messages describing an edit, which only vendors make and only vendors read, so they
// must show every marking.
func advisoryVersionMarkings(
	ctx context.Context, advisoryID, orgID uuid.UUID,
) ([]versionMarking, error) {
	versions, err := db.GetAdvisoryVersions(ctx, []uuid.UUID{advisoryID}, orgID, db.AdvisoryScope{})
	if err != nil {
		return nil, err
	}
	applicationVersions := versions.ApplicationVersions
	artifactVersions := versions.ArtifactVersions

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

func loadAdvisoryVersionMarkings(
	w http.ResponseWriter, r *http.Request, advisoryID uuid.UUID,
) ([]versionMarking, bool) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)
	markings, err := advisoryVersionMarkings(ctx, advisoryID, *a.CurrentOrgID())
	if err != nil {
		internalctx.GetLogger(ctx).Error("failed to get advisory versions", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	return markings, true
}

// maxVersionChangeMessageParts bounds the timeline message: marking a hundred versions at
// once must not write a message nobody can read.
const maxVersionChangeMessageParts = 10

// versionChangeParts matches versions by id, so that one switching between affected and
// patched reads as a change rather than as a removal plus an addition.
func versionChangeParts(before, after []versionMarking) []string {
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

	if len(parts) > maxVersionChangeMessageParts {
		remaining := len(parts) - maxVersionChangeMessageParts
		parts = append(parts[:maxVersionChangeMessageParts], fmt.Sprintf("and %v more", remaining))
	}
	return parts
}

// detailChangeParts reports the description as changed without saying how, since a diff of
// free-form Markdown does not belong in a one-line timeline entry.
func detailChangeParts(before types.Advisory, after api.CreateUpdateAdvisoryRequest) []string {
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

	return parts
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

// referenceChangeParts identifies references by their URL, which is what makes a reference the
// same reference to a reader.
func referenceChangeParts(before []types.AdvisoryReference, after []api.AdvisoryReference) []string {
	beforeURLs := make([]string, len(before))
	for i, reference := range before {
		beforeURLs[i] = reference.URL
	}
	afterURLs := make([]string, len(after))
	for i, reference := range after {
		afterURLs[i] = reference.URL
	}
	return changeParts("references", beforeURLs, afterURLs)
}

func tagChangeParts(before, after []string) []string {
	return changeParts("tags", before, after)
}

func changeParts(noun string, before, after []string) []string {
	var parts []string
	if added := missing(after, before); len(added) > 0 {
		parts = append(parts, fmt.Sprintf("added the %v %v", noun, strings.Join(added, ", ")))
	}
	if removed := missing(before, after); len(removed) > 0 {
		parts = append(parts, fmt.Sprintf("removed the %v %v", noun, strings.Join(removed, ", ")))
	}
	return parts
}

func missing(values, from []string) []string {
	var result []string
	for _, value := range values {
		if !slices.Contains(from, value) {
			result = append(result, value)
		}
	}
	return result
}
