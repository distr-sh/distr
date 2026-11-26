package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/supportbundle/resourcetype"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func SupportBundleConfigurationEnvVarsToAPI(
	envVars []types.SupportBundleConfigurationEnvVar,
) []api.SupportBundleConfigurationEnvVar {
	return List(envVars, func(ev types.SupportBundleConfigurationEnvVar) api.SupportBundleConfigurationEnvVar {
		return api.SupportBundleConfigurationEnvVar{
			Name:     ev.Name,
			Redacted: ev.Redacted,
		}
	})
}

func SupportBundleKubernetesConfigurationToAPI(
	config types.SupportBundleKubernetesConfiguration,
) api.SupportBundleKubernetesConfiguration {
	return api.SupportBundleKubernetesConfiguration{
		IncludeConfigMapData: config.IncludeConfigMapData,
		IncludeSecretData:    config.IncludeSecretData,
	}
}

func SupportBundleKubernetesConfigurationToInternal(
	request api.SupportBundleKubernetesConfiguration,
	orgID uuid.UUID,
) types.SupportBundleKubernetesConfiguration {
	return types.SupportBundleKubernetesConfiguration{
		OrganizationID:       orgID,
		IncludeConfigMapData: request.IncludeConfigMapData,
		IncludeSecretData:    request.IncludeSecretData,
	}
}

func SupportBundleConfigurationScriptToAPI(
	script types.SupportBundleConfigurationScript,
) api.SupportBundleConfigurationScript {
	return api.SupportBundleConfigurationScript{
		ID:             script.ID,
		CreatedAt:      script.CreatedAt,
		DeploymentType: script.DeploymentType,
		Name:           script.Name,
		Description:    script.Description,
		Content:        script.Content,
		Enabled:        script.Enabled,
	}
}

func SupportBundleConfigurationScriptToInternal(
	request api.CreateUpdateSupportBundleConfigurationScriptRequest,
	orgID uuid.UUID,
) types.SupportBundleConfigurationScript {
	return types.SupportBundleConfigurationScript{
		OrganizationID: orgID,
		DeploymentType: request.DeploymentType,
		Name:           request.Name,
		Description:    request.Description,
		Content:        request.Content,
		Enabled:        request.Enabled,
	}
}

// SupportBundleToAPI returns a mapper that converts a SupportBundleWithDetails to its API representation. The
// identity of the users who created the bundle and changed its status is only included when the viewer is
// allowed to see it, by the same rule as deploymentRevisionCreatorToAPI, while their organization and whether they
// are deleted always are. A deleted user has no organization and ranks like a vendor, so only a vendor learns who
// they were.
func SupportBundleToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.SupportBundleWithDetails) api.SupportBundle {
	viewerRank := organizationKindRank(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(bundle types.SupportBundleWithDetails) api.SupportBundle {
		result := api.SupportBundle{
			ID:                                    bundle.ID,
			CreatedAt:                             bundle.CreatedAt,
			CustomerOrganizationID:                bundle.CustomerOrganizationID,
			CustomerOrganizationName:              bundle.CustomerOrganizationName,
			CreatedByCustomerOrganizationID:       bundle.CreatedByCustomerOrganizationID,
			CreatedByPartnerOrganizationID:        bundle.CreatedByPartnerOrganizationID,
			CreatedByDeleted:                      bundle.CreatedByDeleted,
			DeploymentType:                        bundle.DeploymentType,
			Title:                                 bundle.Title,
			Description:                           bundle.Description,
			Status:                                string(bundle.Status),
			ResourceCount:                         bundle.ResourceCount,
			UploadedAt:                            bundle.UploadedAt,
			CommentCount:                          bundle.CommentCount,
			LastCommentAt:                         bundle.LastCommentAt,
			StatusChangedByCustomerOrganizationID: bundle.StatusChangedByCustomerOrganizationID,
			StatusChangedByPartnerOrganizationID:  bundle.StatusChangedByPartnerOrganizationID,
			StatusChangedByDeleted:                bundle.StatusChangedByDeleted,
			StatusChangedAt:                       bundle.StatusChangedAt,
		}
		if organizationKindRank(
			bundle.CreatedByCustomerOrganizationID, bundle.CreatedByPartnerOrganizationID,
		) >= viewerRank {
			result.CreatedByUserAccountID = bundle.CreatedByUserAccountID
			result.CreatedByUserName = bundle.CreatedByUserName
			result.CreatedByImageURL = CreateImageURL(bundle.CreatedByImageID)
		}
		if organizationKindRank(
			bundle.StatusChangedByCustomerOrganizationID, bundle.StatusChangedByPartnerOrganizationID,
		) >= viewerRank {
			result.StatusChangedByUserName = bundle.StatusChangedByUserName
			result.StatusChangedByImageURL = CreateImageURL(bundle.StatusChangedByImageID)
		}
		return result
	}
}

func SupportBundleResourceToAPI(resource types.SupportBundleResource) api.SupportBundleResource {
	return api.SupportBundleResource{
		SupportBundleResourceSummary: SupportBundleResourceSummaryToAPI(resource.SupportBundleResourceSummary),
		Content:                      string(resource.Content),
	}
}

func SupportBundleResourceSummaryToAPI(resource types.SupportBundleResourceSummary) api.SupportBundleResourceSummary {
	return api.SupportBundleResourceSummary{
		ID:          resource.ID,
		CreatedAt:   resource.CreatedAt,
		Name:        resource.Name,
		ContentType: resourcetype.ParseContentType(resource.ContentType),
		Size:        resource.Size,
	}
}

// SupportBundleCommentToAPI returns a mapper that converts a SupportBundleCommentWithUser to its API
// representation, including the identity of the author only when the viewer is allowed to see it, like
// SupportBundleToAPI.
func SupportBundleCommentToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.SupportBundleCommentWithUser) api.SupportBundleComment {
	viewerRank := organizationKindRank(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(comment types.SupportBundleCommentWithUser) api.SupportBundleComment {
		result := api.SupportBundleComment{
			ID:                         comment.ID,
			CreatedAt:                  comment.CreatedAt,
			UserCustomerOrganizationID: comment.UserCustomerOrganizationID,
			UserPartnerOrganizationID:  comment.UserPartnerOrganizationID,
			UserDeleted:                comment.UserDeleted,
			Content:                    comment.Content,
		}
		if organizationKindRank(comment.UserCustomerOrganizationID, comment.UserPartnerOrganizationID) >= viewerRank {
			result.UserAccountID = comment.UserAccountID
			result.UserName = comment.UserName
			result.UserImageURL = CreateImageURL(comment.UserImageID)
		}
		return result
	}
}
