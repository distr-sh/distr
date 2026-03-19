package main

import (
	"context"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/avast/retry-go/v5"
	"github.com/distr-sh/distr/internal/types"
	"github.com/docker/compose/v5/pkg/api"
	mobyClient "github.com/moby/moby/client"
	"go.uber.org/zap"
)

var deleteImageRetrier = retry.New(
	retry.Attempts(3),
	retry.Delay(time.Second),
	retry.DelayType(retry.BackOffDelay),
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

		aggErr = errors.Join(
			aggErr,
			deleteImageRetrier.Do(func() error {
				result, err := apiClient.ImageRemove(ctx, image, mobyClient.ImageRemoveOptions{PruneChildren: true})
				if err != nil {
					logger.Warn("failed to delete old image", zap.Error(err))
				} else {
					logger.Info("deleted old image", zap.Any("result", result))
				}
				return err
			}),
		)
	}

	return
}
