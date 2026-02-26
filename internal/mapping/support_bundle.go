package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func SupportBundleConfigurationToAPI(
	config types.SupportBundleConfiguration,
	envVars []types.SupportBundleConfigurationEnvVar,
) api.SupportBundleConfiguration {
	return api.SupportBundleConfiguration{
		ID:        config.ID,
		CreatedAt: config.CreatedAt,
		EnvVars: List(envVars, func(ev types.SupportBundleConfigurationEnvVar) api.SupportBundleConfigurationEnvVar {
			return api.SupportBundleConfigurationEnvVar{
				ID:       &ev.ID,
				Name:     ev.Name,
				Redacted: ev.Redacted,
			}
		}),
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
