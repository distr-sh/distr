package main

import (
	"context"
	"os"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/controllerclient"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	logger := util.Require(zap.NewDevelopment())
	client := util.Require(controllerclient.NewFromEnv(logger))

	logger.Info("posting fake status", zap.Any("args", os.Args))

	deployment := api.ControllerDeployment{RevisionID: util.Require(uuid.Parse(os.Args[1]))}
	statusType := util.Require(types.ParseDeploymentStatusType(os.Args[2]))
	message := "test status"
	if len(os.Args) > 3 {
		message = os.Args[3]
	}
	util.Must(client.Status(context.Background(), deployment, statusType, message))
	logger.Info("status posted")
}
