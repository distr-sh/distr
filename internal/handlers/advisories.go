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

const duplicateCveIDMessage = "another advisory in this organization already covers this CVE ID"

func AdvisoriesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Advisories"))

	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", getAdvisoriesHandler()).
			With(option.Description("List advisories")).
			With(option.Request(api.ListAdvisoriesRequest{})).
			With(option.Response(http.StatusOK, []api.Advisory{}))

		r.With(middleware.RequireVendorOrPartner).
			Get("/tags", getAdvisoryTagsHandler()).
			With(option.Description("List all advisory tags used in the organization")).
			With(option.Response(http.StatusOK, []string{}))

		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", createAdvisoryHandler()).
			With(option.Description("Create a new advisory")).
			With(option.Request(api.CreateAdvisoryRequest{})).
			With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

		r.Route("/{advisoryId}", func(r chiopenapi.Router) {
			r.Get("/", getAdvisoryDetailHandler()).
				With(option.Description("Get advisory detail")).
				With(option.Request(api.AdvisoryIDRequest{})).
				With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

			// Deliberately not served from the read-only database: the detail view
			// refetches impact right after a status or version change, and replica lag
			// would show the state from before the edit.
			r.With(middleware.RequireVendorOrPartner).
				Get("/impact", getAdvisoryImpactHandler()).
				With(option.Description("Get the customers affected by this advisory")).
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

					r.Patch("/status", updateAdvisoryStatusHandler()).
						With(option.Description("Update the status of an advisory")).
						With(option.Request(struct {
							api.AdvisoryIDRequest
							api.UpdateAdvisoryStatusRequest
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
			CustomerOrgID: a.CurrentCustomerOrgID(),
			Statuses:      parsed.Statuses,
			Severities:    parsed.Severities,
			Tags:          parsed.Tags,
		}

		advisories, err := db.GetAdvisories(ctx, *a.CurrentOrgID(), filter)
		if err != nil {
			log.Error("failed to get advisories", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(advisories, mapping.AdvisoryToAPI))
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

		request, err := JsonBody[api.CreateAdvisoryRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !validateAdvisoryVersionsInOrg(w, r, orgID, &request.CreateUpdateAdvisoryRequest) {
			return
		}

		severity, _ := types.ParseAdvisorySeverity(request.Severity)
		advisory := types.Advisory{
			OrganizationID:         orgID,
			CreatedByUserAccountID: &userID,
			Title:                  request.Title,
			Description:            request.Description,
			Status:                 request.Status,
			Severity:               severity,
			CveID:                  request.CveID,
		}

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.CreateAdvisory(ctx, &advisory); err != nil {
				return err
			}
			if err := applyAdvisoryAssociations(
				ctx, advisory.ID, request.CreateUpdateAdvisoryRequest,
			); err != nil {
				return err
			}
			return db.CreateAdvisoryEvent(
				ctx, advisory.ID, &userID, types.AdvisoryEventTypeCreated, nil)
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

		versionsBefore, ok := loadAdvisoryVersionMarkings(w, r, existing.ID)
		if !ok {
			return
		}

		existingReferences, err := db.GetAdvisoryReferences(ctx, existing.ID)
		if err != nil {
			log.Error("failed to get advisory references", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		referencesAdded, referencesRemoved := referenceChangeMessages(existingReferences, request.References)

		severity, _ := types.ParseAdvisorySeverity(request.Severity)
		advisory := types.Advisory{
			ID:             existing.ID,
			OrganizationID: orgID,
			Title:          request.Title,
			Description:    request.Description,
			Severity:       severity,
			CveID:          request.CveID,
		}

		tagsMessage := tagsChangeMessage(existing.Tags, request.Tags)
		detailsMessage := detailChangeMessage(existing.Advisory, request)

		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.UpdateAdvisory(ctx, &advisory); err != nil {
				return err
			}
			if err := applyAdvisoryAssociations(ctx, advisory.ID, request); err != nil {
				return err
			}
			// Only recorded when something actually changed, so that opening the form and
			// saving it unchanged does not leave a trace that suggests otherwise.
			if detailsMessage != nil {
				if err := db.CreateAdvisoryEvent(
					ctx, advisory.ID, &userID, types.AdvisoryEventTypeEdited, detailsMessage,
				); err != nil {
					return err
				}
			}
			if tagsMessage != nil {
				if err := db.CreateAdvisoryEvent(
					ctx, advisory.ID, &userID, types.AdvisoryEventTypeTagsChanged, tagsMessage,
				); err != nil {
					return err
				}
			}
			if referencesAdded != nil {
				if err := db.CreateAdvisoryEvent(
					ctx, advisory.ID, &userID, types.AdvisoryEventTypeReferenceAdded, referencesAdded,
				); err != nil {
					return err
				}
			}
			if referencesRemoved != nil {
				if err := db.CreateAdvisoryEvent(
					ctx, advisory.ID, &userID, types.AdvisoryEventTypeReferenceRemoved, referencesRemoved,
				); err != nil {
					return err
				}
			}
			// Read back rather than derived from the request, so that the names in the
			// message come from the database instead of being looked up separately.
			versionsAfter, err := advisoryVersionMarkings(ctx, advisory.ID)
			if err != nil {
				return err
			}
			if versionsMessage := versionChangeMessage(versionsBefore, versionsAfter); versionsMessage != nil {
				if err := db.CreateAdvisoryEvent(
					ctx, advisory.ID, &userID, types.AdvisoryEventTypeVersionsChanged, versionsMessage,
				); err != nil {
					return err
				}
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

func updateAdvisoryStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		existing := requireAdvisory(w, r)
		if existing == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)
		userID := a.CurrentUserID()

		request, err := JsonBody[api.UpdateAdvisoryStatusRequest](w, r)
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

		// Canceling means the advisory was never disclosed. Once published_at is set the
		// advisory has been visible to customers at some point (even after unpublishing back to
		// draft), so it can only be unpublished or resolved, never canceled.
		if status == types.AdvisoryStatusCanceled && existing.PublishedAt != nil {
			http.Error(w, "cannot cancel an advisory that has already been published",
				http.StatusBadRequest)
			return
		}

		message := fmt.Sprintf("changed status from %v to %v", existing.Status, status)
		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.UpdateAdvisoryStatus(
				ctx, existing.ID, existing.OrganizationID, status,
			); err != nil {
				return err
			}
			return db.CreateAdvisoryEvent(
				ctx, existing.ID, &userID, types.AdvisoryEventTypeStatusChanged, &message)
		})
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Error("failed to update advisory status", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondAdvisoryDetail(w, r, existing.ID)
	}
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

		deployments, err := db.GetAdvisoryImpactedDeployments(
			ctx, advisory.ID, advisory.OrganizationID, a.CurrentPartnerOrgID())
		if err != nil {
			log.Error("failed to get advisory impacted deployments", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pulls, err := db.GetAdvisoryImpactedPulls(
			ctx, advisory.ID, advisory.OrganizationID, a.CurrentPartnerOrgID())
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

// requireAdvisory parses the advisory ID from the path and loads it scoped to the
// current organization, applying the customer entitlement filter for customer users.
// Returns nil if an error response was already written.
func requireAdvisory(w http.ResponseWriter, r *http.Request) *types.AdvisoryWithDetails {
	id, err := uuid.Parse(r.PathValue("advisoryId"))
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	advisory, err := db.GetAdvisoryByID(ctx, id, *a.CurrentOrgID(), a.CurrentCustomerOrgID())
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

	advisory, err := db.GetAdvisoryByID(ctx, id, *a.CurrentOrgID(), a.CurrentCustomerOrgID())
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

// buildAdvisoryDetail loads the child collections of an advisory. The event
// timeline is vendor-internal and is therefore omitted for customer users.
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

	applicationVersions, err := db.GetAdvisoryApplicationVersions(ctx, advisory.ID)
	if err != nil {
		return fail("failed to get advisory application versions", err)
	}

	artifactVersions, err := db.GetAdvisoryArtifactVersions(ctx, advisory.ID)
	if err != nil {
		return fail("failed to get advisory artifact versions", err)
	}

	events := []types.AdvisoryEventWithUser{}
	if a.CurrentCustomerOrgID() == nil {
		events, err = db.GetAdvisoryEvents(ctx, advisory.ID)
		if err != nil {
			return fail("failed to get advisory events", err)
		}
	}

	return api.AdvisoryDetail{
		Advisory:            mapping.AdvisoryToAPI(advisory),
		Description:         advisory.Description,
		References:          mapping.List(references, mapping.AdvisoryReferenceToAPI),
		ApplicationVersions: mapping.List(applicationVersions, mapping.AdvisoryApplicationVersionToAPI),
		ArtifactVersions:    mapping.List(artifactVersions, mapping.AdvisoryArtifactVersionToAPI),
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
		FixedApplicationVersionIDs:    request.FixedApplicationVersionIDs,
		AffectedArtifactVersionIDs:    request.AffectedArtifactVersionIDs,
		FixedArtifactVersionIDs:       request.FixedArtifactVersionIDs,
	})
}

// validateAdvisoryVersionsInOrg rejects a request that references versions belonging to
// another organization. Returns false if an error response was already written.
func validateAdvisoryVersionsInOrg(
	w http.ResponseWriter, r *http.Request, orgID uuid.UUID, request *api.CreateUpdateAdvisoryRequest,
) bool {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	applicationVersionIDs := slices.Concat(
		request.AffectedApplicationVersionIDs, request.FixedApplicationVersionIDs)
	artifactVersionIDs := slices.Concat(
		request.AffectedArtifactVersionIDs, request.FixedArtifactVersionIDs)

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

// versionMarking is an application or artifact version together with the relation it has to
// the advisory, reduced to what a timeline message needs.
type versionMarking struct {
	id       uuid.UUID
	label    string
	relation types.AdvisoryVersionRelation
}

// advisoryVersionMarkings reads the currently marked application and artifact versions
// as a single list.
func advisoryVersionMarkings(ctx context.Context, advisoryID uuid.UUID) ([]versionMarking, error) {
	applicationVersions, err := db.GetAdvisoryApplicationVersions(ctx, advisoryID)
	if err != nil {
		return nil, err
	}
	artifactVersions, err := db.GetAdvisoryArtifactVersions(ctx, advisoryID)
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

func loadAdvisoryVersionMarkings(
	w http.ResponseWriter, r *http.Request, advisoryID uuid.UUID,
) ([]versionMarking, bool) {
	ctx := r.Context()
	markings, err := advisoryVersionMarkings(ctx, advisoryID)
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
func detailChangeMessage(before types.Advisory, after api.CreateUpdateAdvisoryRequest) *string {
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
	before []types.AdvisoryReference, after []api.AdvisoryReference,
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
