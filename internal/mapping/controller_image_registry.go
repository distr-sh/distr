package mapping

import "github.com/distr-sh/distr/api"

func ControllerImageRegistryToAPI(own *string, defaultImageRegistry string) api.ControllerImageRegistry {
	return api.ControllerImageRegistry{ImageRegistry: own, DefaultImageRegistry: defaultImageRegistry}
}
