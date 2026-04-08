package handlers

import (
	"testing"

	"github.com/distr-sh/distr/api"
	. "github.com/onsi/gomega"
)

func TestSanitizeLogRecords(t *testing.T) {
	t.Run("nil pointer is a no-op", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(func() { sanitizeLogRecords(nil) }).NotTo(Panic())
	})

	t.Run("empty slice remains empty", func(t *testing.T) {
		g := NewWithT(t)
		records := []api.DeploymentLogRecord{}
		sanitizeLogRecords(&records)
		g.Expect(records).To(BeEmpty())
	})

	t.Run("records without null bytes are kept as-is", func(t *testing.T) {
		g := NewWithT(t)
		records := []api.DeploymentLogRecord{
			{Body: "hello world"},
			{Body: "another log line"},
		}
		sanitizeLogRecords(&records)
		g.Expect(records).To(HaveLen(2))
		g.Expect(records[0].Body).To(Equal("hello world"))
		g.Expect(records[1].Body).To(Equal("another log line"))
	})

	t.Run("null bytes are removed from body", func(t *testing.T) {
		g := NewWithT(t)
		records := []api.DeploymentLogRecord{
			{Body: "hel\x00lo"},
			{Body: "\x00leading"},
			{Body: "trailing\x00"},
		}
		sanitizeLogRecords(&records)
		g.Expect(records).To(HaveLen(3))
		g.Expect(records[0].Body).To(Equal("hello"))
		g.Expect(records[1].Body).To(Equal("leading"))
		g.Expect(records[2].Body).To(Equal("trailing"))
	})

	t.Run("records that become empty after null byte removal are filtered out", func(t *testing.T) {
		g := NewWithT(t)
		records := []api.DeploymentLogRecord{
			{Body: "keep me"},
			{Body: "\x00\x00\x00"},
			{Body: "keep me too"},
		}
		sanitizeLogRecords(&records)
		g.Expect(records).To(HaveLen(2))
		g.Expect(records[0].Body).To(Equal("keep me"))
		g.Expect(records[1].Body).To(Equal("keep me too"))
	})

	t.Run("records with empty body are filtered out", func(t *testing.T) {
		g := NewWithT(t)
		records := []api.DeploymentLogRecord{
			{Body: ""},
			{Body: "non-empty"},
		}
		sanitizeLogRecords(&records)
		g.Expect(records).To(HaveLen(1))
		g.Expect(records[0].Body).To(Equal("non-empty"))
	})
}
