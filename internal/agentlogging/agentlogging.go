// Package agentlogging routes what the agent's dependencies log into the agent's zap logger, so that
// the deployment target log core forwards it to Distr like the agent's own messages. What a
// dependency logs past zap never leaves the agent's container.
package agentlogging

import (
	"io"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Redirect sends what logrus and the standard library logger receive to logger. The docker CLI,
// compose and compose-go log through logrus, which keeps its own level (info unless the docker CLI
// is initialized with a log level), so logger's level only decides what is kept from there on.
func Redirect(logger *zap.Logger) {
	std := logrus.StandardLogger()
	// logrus writes every entry to its output in addition to firing the hooks, and that output is
	// os.Stderr, where the zap console core writes the forwarded entry as well.
	std.SetOutput(io.Discard)
	std.AddHook(logrusHook{logger: forwardingLogger(logger)})
	zap.RedirectStdLog(logger)
}

// forwardingLogger strips the annotations zap adds about where an entry was logged. For a forwarded
// entry they describe this package and logrus rather than the code the message is about, and a
// development logger's stacktrace on every warning is long enough to bury the warning itself.
func forwardingLogger(logger *zap.Logger) *zap.Logger {
	return logger.WithOptions(zap.WithCaller(false), zap.AddStacktrace(zapcore.FatalLevel))
}

type logrusHook struct {
	logger *zap.Logger
}

// Levels implements [logrus.Hook].
func (h logrusHook) Levels() []logrus.Level { return logrus.AllLevels }

// Fire implements [logrus.Hook].
func (h logrusHook) Fire(entry *logrus.Entry) error {
	if ce := h.logger.Check(zapLevel(entry.Level), entry.Message); ce != nil {
		ce.Time = entry.Time
		fields := make([]zap.Field, 0, len(entry.Data))
		for key, value := range entry.Data {
			fields = append(fields, zap.Any(key, value))
		}
		ce.Write(fields...)
	}
	return nil
}

func zapLevel(level logrus.Level) zapcore.Level {
	switch level {
	// logrus ends a fatal entry by exiting and a panic entry by panicking itself. Mapping them to
	// the zap levels of the same name would do that from inside the hook, before logrus gets there.
	case logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel:
		return zapcore.ErrorLevel
	case logrus.WarnLevel:
		return zapcore.WarnLevel
	case logrus.InfoLevel:
		return zapcore.InfoLevel
	default:
		return zapcore.DebugLevel
	}
}

var _ logrus.Hook = logrusHook{}
