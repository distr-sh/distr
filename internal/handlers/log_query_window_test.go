package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestResolveLogQueryStart(t *testing.T) {
	t.Run("zero after defaults to the window start", func(t *testing.T) {
		g := NewWithT(t)
		before := subscription.GetLogQueryWindowStart(types.SubscriptionTypeCommunity)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, time.Time{})
		after := subscription.GetLogQueryWindowStart(types.SubscriptionTypeCommunity)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(And(
			BeTemporally(">=", before),
			BeTemporally("<=", after),
		))
	})

	t.Run("after within the window is returned unchanged", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-time.Hour)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, requested)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(requested))
	})

	t.Run("after slightly older than the exact window is allowed (start-of-day slack)", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowCommunity - time.Minute)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, requested)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(requested))
	})

	t.Run("after older than the window including the slack is rejected", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowCommunity - 25*time.Hour)
		_, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, requested)
		g.Expect(err).To(MatchError(apierrors.ErrBadRequest))
	})

	t.Run("pro subscriptions get the larger default window", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowCommunity - time.Minute)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypePro, requested)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(requested))
	})

	t.Run("business subscriptions get the 30-day window", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowDefault - time.Minute)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeBusiness, requested)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(requested))

		tooOld := time.Now().Add(-subscription.LogQueryWindowBusiness - 25*time.Hour)
		_, err = resolveLogQueryStart(types.SubscriptionTypeBusiness, tooOld)
		g.Expect(err).To(MatchError(apierrors.ErrBadRequest))
	})
}

func logQueryRequest(params url.Values) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/?"+params.Encode(), nil)
}

func TestParseLogQueryRange(t *testing.T) {
	t.Run("defaults to the whole query window until now", func(t *testing.T) {
		g := NewWithT(t)
		result, err := parseLogQueryRange(logQueryRequest(nil), types.SubscriptionTypeCommunity)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result.StartExplicit).To(BeFalse())
		g.Expect(result.Start).To(BeTemporally("~", subscription.GetLogQueryWindowStart(
			types.SubscriptionTypeCommunity), time.Minute))
		g.Expect(result.End).To(BeTemporally("~", time.Now(), time.Minute))
		g.Expect(result.Filter).To(BeEmpty())
		g.Expect(result.IsEmpty()).To(BeFalse())
	})

	t.Run("uses the requested range and filter", func(t *testing.T) {
		g := NewWithT(t)
		after := time.Now().Add(-2 * time.Hour).Truncate(time.Second).UTC()
		before := time.Now().Add(-time.Hour).Truncate(time.Second).UTC()
		result, err := parseLogQueryRange(logQueryRequest(url.Values{
			"after":  {after.Format(time.RFC3339Nano)},
			"before": {before.Format(time.RFC3339Nano)},
			"filter": {"some.*error"},
		}), types.SubscriptionTypeCommunity)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result.StartExplicit).To(BeTrue())
		g.Expect(result.Start).To(BeTemporally("==", after))
		g.Expect(result.End).To(BeTemporally("==", before))
		g.Expect(result.Filter).To(Equal("some.*error"))
	})

	t.Run("a before cursor older than the start yields an empty range", func(t *testing.T) {
		g := NewWithT(t)
		result, err := parseLogQueryRange(logQueryRequest(url.Values{
			"before": {time.Now().Add(-subscription.LogQueryWindowCommunity - time.Hour).Format(time.RFC3339Nano)},
		}), types.SubscriptionTypeCommunity)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result.IsEmpty()).To(BeTrue())
	})

	t.Run("rejects an unparseable timestamp", func(t *testing.T) {
		g := NewWithT(t)
		_, err := parseLogQueryRange(logQueryRequest(url.Values{"after": {"yesterday"}}),
			types.SubscriptionTypeCommunity)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("rejects an invalid filter regex", func(t *testing.T) {
		g := NewWithT(t)
		_, err := parseLogQueryRange(logQueryRequest(url.Values{"filter": {"["}}), types.SubscriptionTypeCommunity)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("rejects a start outside the query window", func(t *testing.T) {
		g := NewWithT(t)
		tooOld := time.Now().Add(-subscription.LogQueryWindowCommunity - 25*time.Hour)
		_, err := parseLogQueryRange(logQueryRequest(url.Values{"after": {tooOld.Format(time.RFC3339Nano)}}),
			types.SubscriptionTypeCommunity)
		g.Expect(err).To(MatchError(apierrors.ErrBadRequest))
	})
}
