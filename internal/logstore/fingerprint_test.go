package logstore

import (
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestDeploymentLogRecordID(t *testing.T) {
	base := DeploymentLogRecord{
		DeploymentID:         uuid.MustParse("98be36e4-aa8a-4596-a5e8-8da0e0974105"),
		DeploymentRevisionID: uuid.MustParse("addb2eac-c1e5-4580-a36e-42c011327dd5"),
		Resource:             "some-resource",
		Timestamp:            time.Date(2026, 7, 16, 10, 0, 0, 123456789, time.UTC),
		Severity:             "info",
		Body:                 "hello world",
	}

	t.Run("is deterministic", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(DeploymentLogRecordID(base)).To(Equal(DeploymentLogRecordID(base)))
	})

	t.Run("changes when severity changes", func(t *testing.T) {
		g := NewWithT(t)
		other := base
		other.Severity = "error"
		g.Expect(DeploymentLogRecordID(other)).NotTo(Equal(DeploymentLogRecordID(base)))
	})

	t.Run("changes when timestamp changes", func(t *testing.T) {
		g := NewWithT(t)
		other := base
		other.Timestamp = base.Timestamp.Add(time.Nanosecond)
		g.Expect(DeploymentLogRecordID(other)).NotTo(Equal(DeploymentLogRecordID(base)))
	})

	t.Run("changes when body changes", func(t *testing.T) {
		g := NewWithT(t)
		other := base
		other.Body = "goodbye world"
		g.Expect(DeploymentLogRecordID(other)).NotTo(Equal(DeploymentLogRecordID(base)))
	})

	t.Run("changes when resource changes", func(t *testing.T) {
		g := NewWithT(t)
		other := base
		other.Resource = "other-resource"
		g.Expect(DeploymentLogRecordID(other)).NotTo(Equal(DeploymentLogRecordID(base)))
	})
}

func TestDeploymentTargetLogRecordID(t *testing.T) {
	base := DeploymentTargetLogRecord{
		DeploymentTargetID: uuid.MustParse("bd3ff37e-9dc2-4de6-9668-3f0e4c112233"),
		Timestamp:          time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
		Severity:           "info",
		Body:               "agent says hi",
	}

	t.Run("is deterministic", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(DeploymentTargetLogRecordID(base)).To(Equal(DeploymentTargetLogRecordID(base)))
	})

	t.Run("differs from deployment log record fingerprint domain", func(t *testing.T) {
		g := NewWithT(t)
		record := DeploymentLogRecord{
			Timestamp: base.Timestamp,
			Severity:  base.Severity,
			Body:      base.Body,
		}
		g.Expect(DeploymentLogRecordID(record)).NotTo(Equal(DeploymentTargetLogRecordID(base)))
	})
}
