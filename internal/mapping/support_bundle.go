package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
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
		Content:   resource.Content,
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
