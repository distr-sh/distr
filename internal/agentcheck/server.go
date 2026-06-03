package agentcheck

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	heartbeatTimestamp       *time.Time
	heartbeatHealthyDuration time.Duration
	mut                      sync.RWMutex
}

func NewServer(d time.Duration) *Server {
	return &Server{heartbeatHealthyDuration: d}
}

func (h *Server) Heartbeat() {
	h.HeartbeatAt(time.Now())
}

func (h *Server) HeartbeatAt(t time.Time) {
	h.mut.Lock()
	defer h.mut.Unlock()

	h.heartbeatTimestamp = &t
}

func (h *Server) IsStale() bool {
	return h.heartbeatTimestamp == nil || h.heartbeatTimestamp.Add(h.heartbeatHealthyDuration).Before(time.Now())
}

// ServeHTTP implements [http.Handler].
func (h *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mut.RLock()
	defer h.mut.RUnlock()

	if h.heartbeatTimestamp == nil {
		http.Error(w, "timestamp not available", http.StatusInternalServerError)
	} else if h.IsStale() {
		http.Error(w,
			fmt.Sprintf("timestamp %v is older than %v",
				h.heartbeatTimestamp.Format(time.RFC3339Nano),
				h.heartbeatHealthyDuration),
			http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
