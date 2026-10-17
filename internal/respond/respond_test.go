package respond

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/getsentry/sentry-go"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

func serve(t *testing.T, ctx context.Context, resp Responder, err error) (*httptest.ResponseRecorder, []*sentry.Event) {
	t.Helper()
	rec := httptest.NewRecorder()
	return rec, serveTo(t, ctx, rec, resp, err)
}

func serveTo(t *testing.T, ctx context.Context, w http.ResponseWriter, resp Responder, err error) []*sentry.Event {
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

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	Handler(func(*http.Request) (Responder, error) { return resp, err }).ServeHTTP(w, req)
	return events
}

type disconnectedWriter struct {
	*httptest.ResponseRecorder
}

func (disconnectedWriter) Write([]byte) (int, error) {
	return 0, errors.New("broken pipe")
}

func TestHandlerErrors(t *testing.T) {
	t.Run("unhandled sentinel is a reported 500 and its cause stays out of the body", func(t *testing.T) {
		g := NewWithT(t)
		err := fmt.Errorf("get target at 10.0.0.1: %w", distrerr.ErrNotFound)
		rec, events := serve(t, t.Context(), JSON("ignored"), err)
		g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		g.Expect(rec.Body.String()).To(Equal("Internal Server Error\n"))
		g.Expect(events).To(HaveLen(1))
	})

	t.Run("cancellation by the client is not reported", func(t *testing.T) {
		g := NewWithT(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		rec, events := serve(t, ctx, nil, fmt.Errorf("query: %w", context.Canceled))
		g.Expect(rec.Code).To(Equal(statusClientClosedRequest))
		g.Expect(rec.Body.Len()).To(BeZero())
		g.Expect(events).To(BeEmpty())
	})

	t.Run("writer answers in its place until the first write", func(t *testing.T) {
		g := NewWithT(t)
		rec, _ := serve(t, t.Context(), Writer("text/csv", func(io.Writer) (Responder, error) {
			return BadRequest("invalid filter"), nil
		}), nil)
		g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
		g.Expect(rec.Header().Get("Content-Type")).NotTo(Equal("text/csv"))

		rec, events := serve(t, t.Context(), Writer("text/csv", func(io.Writer) (Responder, error) {
			return nil, errors.New("query failed")
		}), nil)
		g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		g.Expect(events).To(HaveLen(1))
	})

	t.Run("writer keeps what it wrote when failing afterwards", func(t *testing.T) {
		g := NewWithT(t)
		rec, events := serve(t, t.Context(), Writer("text/csv", func(w io.Writer) (Responder, error) {
			_, _ = io.WriteString(w, "a,b\n")
			return BadRequest("ignored"), errors.New("query failed")
		}), nil)
		g.Expect(rec.Code).To(Equal(http.StatusOK))
		g.Expect(rec.Header().Get("Content-Type")).To(Equal("text/csv"))
		g.Expect(rec.Body.String()).To(Equal("a,b\n"))
		g.Expect(events).To(HaveLen(1))
	})

	t.Run("writer does not report a client that is gone", func(t *testing.T) {
		g := NewWithT(t)
		events := serveTo(t, t.Context(), disconnectedWriter{httptest.NewRecorder()},
			Writer("text/csv", func(w io.Writer) (Responder, error) {
				if _, err := io.WriteString(w, "a,b\n"); err != nil {
					return nil, fmt.Errorf("failed to write export: %w", err)
				}
				return nil, nil
			}), nil)
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
