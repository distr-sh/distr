package agentlogging

import (
	"bytes"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRedirectForwardsLogrusToZap(t *testing.T) {
	g := NewWithT(t)
	core, logs := observer.New(zapcore.DebugLevel)

	var logrusOutput bytes.Buffer
	logrus.StandardLogger().SetOutput(&logrusOutput)
	Redirect(zap.New(core))

	logrus.WithField("project", "foo").Warnf("Found orphan containers (%s) for this project", "bar")

	entries := logs.All()
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Level).To(Equal(zapcore.WarnLevel))
	g.Expect(entries[0].Message).To(Equal("Found orphan containers (bar) for this project"))
	g.Expect(entries[0].ContextMap()).To(HaveKeyWithValue("project", "foo"))

	// logrus must not write the entry itself as well, since the zap console core already writes the
	// forwarded one to the same stderr.
	g.Expect(logrusOutput.Len()).To(BeZero())
}
