package main

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/distr-sh/distr/internal/types"
	"github.com/docker/compose/v5/pkg/api"
	mobyClient "github.com/moby/moby/client"
	"go.uber.org/zap"
)

func GetDeploymentImages(ctx context.Context, deployment AgentDeployment) ([]string, error) {
	switch deployment.DockerType {
	case types.DockerTypeCompose:
		return getDeploymentImagesCompose(ctx, deployment)
	default:
		return nil, nil
	}
}

func getDeploymentImagesCompose(ctx context.Context, deployment AgentDeployment) ([]string, error) {
	summaries, err := composeService.Ps(ctx, deployment.ProjectName, api.PsOptions{All: true})
	if err != nil {
		return nil, err
	}

	images := make(map[string]struct{}, len(summaries))
	for _, summary := range summaries {
		images[summary.Image] = struct{}{}
	}

	return slices.Collect(maps.Keys(images)), nil
}

func DeleteImages(ctx context.Context, images []string) (aggErr error) {
	apiClient := dockerCli.Client()

	for _, image := range images {
		logger := logger.With(zap.String("image", image))
		logger.Debug("trying to delete old image")

		result, err := apiClient.ImageRemove(ctx, image, mobyClient.ImageRemoveOptions{PruneChildren: true})
		if err != nil {
			logger.Warn("failed to delete old image", zap.Error(err))
			aggErr = errors.Join(aggErr, err)
		} else {
			logger.Info("deleted old image", zap.Any("result", result))
		}
	}

	return
}
