package logstore

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestFakeQueryDeploymentLogRecords(t *testing.T) {
	g := NewWithT(t)
	fake := NewFake()
	base := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	otherOrgID := uuid.New()

	g.Expect(fake.SaveDeploymentLogRecords(t.Context(), testOrgID, []api.DeploymentLogRecord{
		{
			DeploymentID: testDeploymentID, DeploymentRevisionID: testRevisionID,
			Resource: "resource-a", Timestamp: base, Severity: "info", Body: "matching one",
		},
		{
			DeploymentID: testDeploymentID, DeploymentRevisionID: testRevisionID,
			Resource: "resource-a", Timestamp: base.Add(time.Second), Severity: "info", Body: "no thanks",
		},
		{
			DeploymentID: testDeploymentID, DeploymentRevisionID: testRevisionID,
			Resource: "resource-b", Timestamp: base, Severity: "info", Body: "matching other resource",
		},
	})).To(Succeed())
	g.Expect(fake.SaveDeploymentLogRecords(t.Context(), otherOrgID, []api.DeploymentLogRecord{
		{
			DeploymentID: testDeploymentID, DeploymentRevisionID: testRevisionID,
			Resource: "resource-a", Timestamp: base, Severity: "info", Body: "matching but other org",
		},
	})).To(Succeed())

	records, err := util.SeqCollect(fake.QueryDeploymentLogRecords(t.Context(), testOrgID, DeploymentLogQuery{
		DeploymentID: testDeploymentID,
		Resources:    []string{"resource-a"},
		Start:        base,
		End:          base.Add(time.Minute),
		Filter:       "matching",
		Limit:        10,
		Direction:    types.OrderDirectionDesc,
	}))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Body).To(Equal("matching one"))
	g.Expect(records[0].ID).To(Equal(DeploymentLogRecordID(records[0])))
}

func TestFakeQueryDeploymentLogRecordsInvalidFilter(t *testing.T) {
	g := NewWithT(t)
	fake := NewFake()
	_, err := util.SeqCollect(fake.QueryDeploymentLogRecords(t.Context(), testOrgID, DeploymentLogQuery{Filter: "["}))
	g.Expect(err).To(MatchError(apierrors.ErrBadRequest))
}

func TestFakeGetDeploymentLogRecordResources(t *testing.T) {
	g := NewWithT(t)
	fake := NewFake()
	base := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	oldRevisionID := uuid.New()

	g.Expect(fake.SaveDeploymentLogRecords(t.Context(), testOrgID, []api.DeploymentLogRecord{
		{
			DeploymentID: testDeploymentID, DeploymentRevisionID: testRevisionID,
			Resource: "active-resource", Timestamp: base, Severity: "info", Body: "x",
		},
		{
			DeploymentID: testDeploymentID, DeploymentRevisionID: oldRevisionID,
			Resource: "archived-resource", Timestamp: base, Severity: "info", Body: "x",
		},
	})).To(Succeed())

	active, archived, err := fake.GetDeploymentLogRecordResources(t.Context(), testOrgID, DeploymentLogResourcesQuery{
		DeploymentID:      testDeploymentID,
		LatestRevisionIDs: []uuid.UUID{testRevisionID},
		Start:             base.Add(-time.Hour),
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(active).To(Equal([]string{"active-resource"}))
	g.Expect(archived).To(Equal([]string{"archived-resource"}))
}

func TestFakeQueryDeploymentTargetLogRecords(t *testing.T) {
	g := NewWithT(t)
	fake := NewFake()
	base := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	g.Expect(fake.SaveDeploymentTargetLogRecords(t.Context(), testOrgID, testDeploymentTargetID,
		[]api.DeploymentTargetLogRecordRequest{
			{Timestamp: base, Severity: "info", Body: "older"},
			{Timestamp: base.Add(time.Second), Severity: "info", Body: "newer"},
		})).To(Succeed())

	var bodies []string
	seq := fake.QueryDeploymentTargetLogRecords(t.Context(), testOrgID, DeploymentTargetLogQuery{
		DeploymentTargetID: testDeploymentTargetID,
		Start:              base,
		Limit:              10,
		Direction:          types.OrderDirectionDesc,
	})
	for record, err := range seq {
		g.Expect(err).NotTo(HaveOccurred())
		bodies = append(bodies, record.Body)
	}
	g.Expect(bodies).To(Equal([]string{"newer", "older"}))
}
