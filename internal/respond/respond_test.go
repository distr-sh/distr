package respond

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/getsentry/sentry-go"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

func serve(t *testing.T, ctx context.Context, resp Responder, err error) (*httptest.ResponseRecorder, []*sentry.Event) {
	t.Helper()
	var events []*sentry.Event
	client, clientErr := sentry.NewClient(sentry.ClientOptions{
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			events = append(events, event)
			return nil
		},
	})
	if clientErr != nil {
		t.Fatal(clientErr)
	}
	ctx = sentry.SetHubOnContext(ctx, sentry.NewHub(client, sentry.NewScope()))
	ctx = internalctx.WithLogger(ctx, zap.NewNop())

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	Handler(func(*http.Request) (Responder, error) { return resp, err }).ServeHTTP(rec, req)
	return rec, events
}

func TestHandlerErrors(t *testing.T) {
	t.Run("unexpected error is reported and its cause stays out of the body", func(t *testing.T) {
		g := NewWithT(t)
		rec, events := serve(t, t.Context(), JSON("ignored"), errors.New("connection to 10.0.0.1 refused"))
		g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		g.Expect(rec.Body.String()).To(Equal("Internal Server Error\n"))
		g.Expect(events).To(HaveLen(1))
	})

	t.Run("wrapped sentinel answers with its status but not its details", func(t *testing.T) {
		g := NewWithT(t)
		err := fmt.Errorf("%w: key of target 1234 does not match", apierrors.ErrUnauthorized)
		rec, events := serve(t, t.Context(), nil, err)
		g.Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		g.Expect(rec.Body.String()).To(Equal("Unauthorized\n"))
		g.Expect(events).To(BeEmpty())
	})

	t.Run("wrapped api error answers with its own message only", func(t *testing.T) {
		g := NewWithT(t)
		err := fmt.Errorf("create secret: %w", apierrors.NewConflict("secret with this key already exists"))
		rec, events := serve(t, t.Context(), nil, err)
		g.Expect(rec.Code).To(Equal(http.StatusConflict))
		g.Expect(rec.Body.String()).To(Equal("secret with this key already exists\n"))
		g.Expect(errors.Is(err, apierrors.ErrConflict)).To(BeTrue())
		g.Expect(events).To(BeEmpty())
	})

	t.Run("cancellation by the client is not reported", func(t *testing.T) {
		g := NewWithT(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		rec, events := serve(t, ctx, nil, fmt.Errorf("query: %w", context.Canceled))
		g.Expect(rec.Body.Len()).To(BeZero())
		g.Expect(events).To(BeEmpty())
	})

	t.Run("encoding failure is still answered with a 500", func(t *testing.T) {
		g := NewWithT(t)
		rec, events := serve(t, t.Context(), JSON(make(chan int)), nil)
		g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		g.Expect(rec.Header().Get("Content-Type")).NotTo(Equal("application/json"))
		g.Expect(events).To(HaveLen(1))
	})
}
