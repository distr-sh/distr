package svc

import (
	"go.uber.org/zap"
)

func (r *Registry) GetLogger() *zap.Logger {
	return r.logger
}
