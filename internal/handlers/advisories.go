package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

const duplicateCveIDMessage = "another advisory in this organization already covers this CVE ID"

func advisoryScope(a authinfo.AuthInfoWithUserAndOrganization) db.AdvisoryScope {
	return db.AdvisoryScope{
		PartnerOrgID:  a.CurrentPartnerOrgID(),
		CustomerOrgID: a.CurrentCustomerOrgID(),
	}
}

func advisoryFilter(
	a authinfo.AuthInfoWithUserAndOrganization, parsed api.ParsedListAdvisoriesRequest,
) db.AdvisoryFilter {
	return db.AdvisoryFilter{
		Scope:      advisoryScope(a),
		Statuses:   parsed.Statuses,
		Severities: parsed.Severities,
		Tags:       parsed.Tags,
	}
}

func AdvisoriesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Advisories"))

	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", respond.Handler(getAdvisoriesHandler)).
			With(option.Description("List advisories")).
			With(option.Request(api.ListAdvisoriesRequest{})).
			With(option.Response(http.StatusOK, []api.Advisory{}))

		r.With(middleware.RequireVendor).
			Get("/tags", respond.Handler(getAdvisoryTagsHandler)).
			With(option.Description("List all advisory tags used in the organization")).
			With(option.Response(http.StatusOK, []string{}))

		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", respond.Handler(createAdvisoryHandler)).
			With(option.Description("Create a new advisory")).
			With(option.Request(api.CreateUpdateAdvisoryRequest{})).
			With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

		r.Route("/{advisoryId}", func(r chiopenapi.Router) {
			r.Get("/", respond.Handler(getAdvisoryDetailHandler)).
				With(option.Description("Get advisory detail")).
				With(option.Request(api.AdvisoryIDRequest{})).
				With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

			// Deliberately not served from the read-only database: the detail view refetches
			// impact right after an edit, and replica lag would show the state from before it.
			r.Get("/impact", respond.Handler(getAdvisoryImpactHandler)).
				With(option.Description("Get the deployments and downloads affected by this advisory")).
				With(option.Request(api.AdvisoryIDRequest{})).
				With(option.Response(http.StatusOK, api.AdvisoryImpact{}))

			r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Group(func(r chiopenapi.Router) {
					r.Put("/", respond.Handler(updateAdvisoryHandler)).
						With(option.Description("Update an advisory")).
						With(option.Request(struct {
							api.AdvisoryIDRequest
							api.CreateUpdateAdvisoryRequest
						}{})).
						With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

					r.Patch("/", respond.Handler(patchAdvisoryHandler)).
						With(option.Description("Change the status or severity of an advisory")).
						With(option.Request(struct {
							api.AdvisoryIDRequest
							api.PatchAdvisoryRequest
						}{})).
						With(option.Response(http.StatusOK, api.AdvisoryDetail{}))

					r.Post("/comments", respond.Handler(createAdvisoryCommentHandler)).
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

var advisoryNotFound = respond.NotFound("advisory not found")

func getAdvisoriesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	values := r.URL.Query()
	query := api.ListAdvisoriesRequest{
		Status:   values["status"],
		Severity: values["severity"],
		Tag:      values["tag"],
	}
	if parsed, err := query.Parse(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if advisories, err := db.GetAdvisories(ctx, *a.CurrentOrgID(), advisoryFilter(a, parsed)); err != nil {
		return nil, fmt.Errorf("failed to get advisories: %w", err)
	} else {
		return respond.JSON(mapping.List(
			advisories,
			mapping.AdvisoryToAPI(a.CurrentCustomerOrgID(), a.CurrentPartnerOrgID()),
		)), nil
	}
}

func getAdvisoryTagsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if tags, err := db.GetAdvisoryTagNames(ctx, *a.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get advisory tags: %w", err)
	} else {
		return respond.JSON(tags), nil
	}
}

func getAdvisoryDetailHandler(r *http.Request) (respond.Responder, error) {
	if advisory, resp, err := requireAdvisory(r); resp != nil || err != nil {
		return resp, err
	} else {
		return advisoryDetailResponse(r, *advisory)
	}
}

func createAdvisoryHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)
	orgID := *a.CurrentOrgID()
	userID := a.CurrentUserID()

	if request, resp := decodeJSON[api.CreateUpdateAdvisoryRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if resp, err := validateAdvisoryVersionsInOrg(r, orgID, &request); resp != nil || err != nil {
		return resp, err
	} else {
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

		err := db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.CreateAdvisory(ctx, &advisory); err != nil {
				return err
			}
			if err := applyAdvisoryAssociations(ctx, advisory.ID, request); err != nil {
				return err
			}
			return recordAdvisoryPublication(ctx, &advisory, userID)
		})
		if errors.Is(err, distrerr.ErrConflict) {
			return respond.Conflict(duplicateCveIDMessage), nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to create advisory: %w", err)
		} else {
			return respondAdvisoryDetail(r, advisory.ID)
		}
	}
}

func updateAdvisoryHandler(r *http.Request) (respond.Responder, error) {
	existing, resp, err := requireAdvisory(r)
	if resp != nil || err != nil {
		return resp, err
	}

	ctx := r.Context()
	a := auth.Authentication.Require(ctx)
	orgID := *a.CurrentOrgID()
	userID := a.CurrentUserID()

	request, resp := decodeJSON[api.CreateUpdateAdvisoryRequest](r)
	if resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	}

	if resp, err := validateAdvisoryVersionsInOrg(r, orgID, &request); resp != nil || err != nil {
		return resp, err
	}

	// An advisory that has not been disclosed yet records nothing, so the change detection
	// feeding the timeline message is only worth running once it has been.
	wasPublished := existing.PublishedAt != nil
	var changes []string
	var versionsBefore []versionMarking
	if wasPublished {
		if versionsBefore, err = loadAdvisoryVersionMarkings(r, existing.ID); err != nil {
			return nil, err
		}

		existingReferences, err := db.GetAdvisoryReferences(ctx, existing.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get advisory references: %w", err)
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
	if errors.Is(err, distrerr.ErrNotFound) {
		return advisoryNotFound, nil
	} else if errors.Is(err, distrerr.ErrConflict) {
		return respond.Conflict(duplicateCveIDMessage), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to update advisory: %w", err)
	}

	return respondAdvisoryDetail(r, advisory.ID)
}

func patchAdvisoryHandler(r *http.Request) (respond.Responder, error) {
	existing, resp, err := requireAdvisory(r)
	if resp != nil || err != nil {
		return resp, err
	}

	ctx := r.Context()
	a := auth.Authentication.Require(ctx)
	userID := a.CurrentUserID()

	request, resp := decodeJSON[api.PatchAdvisoryRequest](r)
	if resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
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
	if errors.Is(err, distrerr.ErrNotFound) {
		return advisoryNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to patch advisory: %w", err)
	}

	return respondAdvisoryDetail(r, existing.ID)
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

func createAdvisoryCommentHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if advisory, resp, err := requireAdvisory(r); resp != nil || err != nil {
		return resp, err
	} else if request, resp := decodeJSON[api.CreateAdvisoryCommentRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if event, err := db.CreateAdvisoryCommentEvent(
		ctx, advisory.ID, a.CurrentUserID(), request.Content,
	); err != nil {
		return nil, fmt.Errorf("failed to create advisory comment: %w", err)
	} else {
		return respond.JSON(mapping.AdvisoryEventToAPI(*event)), nil
	}
}

func getAdvisoryImpactHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	// Customers reach this endpoint too, to see whether their own deployments still run an
	// affected version. The scope keeps every other customer's rows out of the response.
	scope := advisoryScope(a)

	if advisory, resp, err := requireAdvisory(r); resp != nil || err != nil {
		return resp, err
	} else if deployments, err := db.GetAdvisoryImpactedDeployments(
		ctx, advisory.ID, advisory.OrganizationID, scope,
	); err != nil {
		return nil, fmt.Errorf("failed to get advisory impacted deployments: %w", err)
	} else if pulls, err := db.GetAdvisoryImpactedPulls(
		ctx, advisory.ID, advisory.OrganizationID, scope,
	); err != nil {
		return nil, fmt.Errorf("failed to get advisory impacted pulls: %w", err)
	} else {
		return respond.JSON(api.AdvisoryImpact{
			Deployments: mapping.List(deployments, mapping.AdvisoryImpactedDeploymentToAPI),
			Pulls:       mapping.List(pulls, mapping.AdvisoryImpactedPullToAPI),
		}), nil
	}
}

func requireAdvisory(r *http.Request) (*types.AdvisoryWithDetails, respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("advisoryId")); err != nil {
		return nil, advisoryNotFound, nil
	} else if advisory, err := db.GetAdvisoryByID(
		ctx, id, *a.CurrentOrgID(), advisoryScope(a),
	); errors.Is(err, distrerr.ErrNotFound) {
		return nil, advisoryNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get advisory: %w", err)
	} else {
		return advisory, nil, nil
	}
}

func respondAdvisoryDetail(r *http.Request, id uuid.UUID) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if advisory, err := db.GetAdvisoryByID(ctx, id, *a.CurrentOrgID(), advisoryScope(a)); err != nil {
		return nil, fmt.Errorf("failed to get advisory: %w", err)
	} else {
		return advisoryDetailResponse(r, *advisory)
	}
}

// advisoryDetailResponse omits the event timeline for customer and partner users, since it is
// vendor-internal.
func advisoryDetailResponse(r *http.Request, advisory types.AdvisoryWithDetails) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if references, err := db.GetAdvisoryReferences(ctx, advisory.ID); err != nil {
		return nil, fmt.Errorf("failed to get advisory references: %w", err)
	} else if versions, err := db.GetAdvisoryVersions(
		ctx, []uuid.UUID{advisory.ID}, *a.CurrentOrgID(), advisoryScope(a),
	); err != nil {
		return nil, fmt.Errorf("failed to get advisory versions: %w", err)
	} else {
		events := []types.AdvisoryEventWithUser{}
		if a.CurrentCustomerOrgID() == nil && a.CurrentPartnerOrgID() == nil {
			if events, err = db.GetAdvisoryEvents(ctx, advisory.ID); err != nil {
				return nil, fmt.Errorf("failed to get advisory events: %w", err)
			}
		}

		return respond.JSON(api.AdvisoryDetail{
			Advisory:    mapping.AdvisoryToAPI(a.CurrentCustomerOrgID(), a.CurrentPartnerOrgID())(advisory),
			Description: advisory.Description,
			References:  mapping.List(references, mapping.AdvisoryReferenceToAPI),
			ApplicationVersions: mapping.List(
				versions.ApplicationVersions,
				mapping.AdvisoryApplicationVersionToAPI(a.CurrentCustomerOrgID(), a.CurrentPartnerOrgID()),
			),
			ArtifactVersions: mapping.List(versions.ArtifactVersions, mapping.AdvisoryArtifactVersionToAPI),
			Events:           mapping.List(events, mapping.AdvisoryEventToAPI),
		}), nil
	}
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

func validateAdvisoryVersionsInOrg(
	r *http.Request, orgID uuid.UUID, request *api.CreateUpdateAdvisoryRequest,
) (respond.Responder, error) {
	ctx := r.Context()

	applicationVersionIDs := slices.Concat(
		request.AffectedApplicationVersionIDs, request.PatchedApplicationVersionIDs)
	artifactVersionIDs := slices.Concat(
		request.AffectedArtifactVersionIDs, request.PatchedArtifactVersionIDs)

	if count, err := db.CountAdvisoryVersionsOutsideOrg(
		ctx, orgID, applicationVersionIDs, artifactVersionIDs,
	); err != nil {
		return nil, fmt.Errorf("failed to validate advisory versions: %w", err)
	} else if count > 0 {
		return respond.BadRequest("one or more versions do not exist in this organization"), nil
	} else {
		return nil, nil
	}
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
			id:       version.ApplicationVersion.ID,
			label:    version.ApplicationName + " " + version.ApplicationVersion.Name,
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

func loadAdvisoryVersionMarkings(r *http.Request, advisoryID uuid.UUID) ([]versionMarking, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)
	markings, err := advisoryVersionMarkings(ctx, advisoryID, *a.CurrentOrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to get advisory versions: %w", err)
	}
	return markings, nil
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
