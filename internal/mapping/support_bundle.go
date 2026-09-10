package mapping

import (
	"github.com/distr-sh/distr/api"
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

func SupportBundleConfigurationScriptToAPI(
	script types.SupportBundleConfigurationScript,
) api.SupportBundleConfigurationScript {
	return api.SupportBundleConfigurationScript{
		ID:          script.ID,
		CreatedAt:   script.CreatedAt,
		Name:        script.Name,
		Description: script.Description,
		Content:     script.Content,
		Enabled:     script.Enabled,
	}
}

func SupportBundleConfigurationScriptToInternal(
	request api.CreateUpdateSupportBundleConfigurationScriptRequest,
	orgID uuid.UUID,
) types.SupportBundleConfigurationScript {
	return types.SupportBundleConfigurationScript{
		OrganizationID: orgID,
		Name:           request.Name,
		Description:    request.Description,
		Content:        request.Content,
		Enabled:        request.Enabled,
	}
}

func SupportBundleToAPI(bundle types.SupportBundleWithDetails) api.SupportBundle {
	return api.SupportBundle{
		ID:                       bundle.ID,
		CreatedAt:                bundle.CreatedAt,
		CustomerOrganizationID:   bundle.CustomerOrganizationID,
		CustomerOrganizationName: bundle.CustomerOrganizationName,
		CreatedByUserAccountID:   bundle.CreatedByUserAccountID,
		CreatedByUserName:        bundle.CreatedByUserName,
		CreatedByImageURL:        CreateImageURL(bundle.CreatedByImageID),
		Title:                    bundle.Title,
		Description:              bundle.Description,
		Status:                   string(bundle.Status),
		ResourceCount:            bundle.ResourceCount,
		CommentCount:             bundle.CommentCount,
		LastCommentAt:            bundle.LastCommentAt,
		StatusChangedByUserName:  bundle.StatusChangedByUserName,
		StatusChangedByImageURL:  CreateImageURL(bundle.StatusChangedByImageID),
		StatusChangedAt:          bundle.StatusChangedAt,
	}
}

func SupportBundleResourceToAPI(resource types.SupportBundleResource) api.SupportBundleResource {
	return api.SupportBundleResource{
		ID:        resource.ID,
		CreatedAt: resource.CreatedAt,
		Name:      resource.Name,
		Content:   string(resource.Content),
	}
}

func SupportBundleResourceToSummaryAPI(resource types.SupportBundleResource) api.SupportBundleResourceSummary {
	return api.SupportBundleResourceSummary{
		ID:        resource.ID,
		CreatedAt: resource.CreatedAt,
		Name:      resource.Name,
	}
}

func SupportBundleCommentToAPI(comment types.SupportBundleCommentWithUser) api.SupportBundleComment {
	return api.SupportBundleComment{
		ID:            comment.ID,
		CreatedAt:     comment.CreatedAt,
		UserAccountID: comment.UserAccountID,
		UserName:      comment.UserName,
		UserImageURL:  CreateImageURL(comment.UserImageID),
		Content:       comment.Content,
	}
}
