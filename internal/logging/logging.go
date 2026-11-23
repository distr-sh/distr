package logging

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/distr-sh/distr/internal/buildconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New() *zap.Logger {
	if buildconfig.IsRelease() {
		config := zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		return zap.Must(config.Build())
	} else {
		return zap.Must(zap.NewDevelopment())
	}
}

func Sync(logger *zap.Logger) error {
	// some devices like stdout and stderr can not be synced by the OS
	if err := logger.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
		return fmt.Errorf("logger sync failed: %w", err)
	}
	return nil
}
