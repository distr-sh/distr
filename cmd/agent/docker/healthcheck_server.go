package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type helathcheckServer struct {
	heartbeatTimestamp       *time.Time
	heartbeatHealthyDuration time.Duration
	mut                      sync.RWMutex
}

func NewHealthcheckServer(d time.Duration) *helathcheckServer {
	return &helathcheckServer{heartbeatHealthyDuration: d}
}

func (h *helathcheckServer) HeartbeatNow() {
	h.HeartbeatAt(time.Now())
}

func (h *helathcheckServer) HeartbeatAt(t time.Time) {
	h.mut.Lock()
	defer h.mut.Unlock()

	h.heartbeatTimestamp = &t
}

func (h *helathcheckServer) IsStale() bool {
	return h.heartbeatTimestamp == nil || h.heartbeatTimestamp.Add(h.heartbeatHealthyDuration).Before(time.Now())
}

// ServeHTTP implements [http.Handler].
func (h *helathcheckServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mut.RLock()
	defer h.mut.RUnlock()

	if h.heartbeatTimestamp == nil {
		http.Error(w, "timestamp not available", http.StatusInternalServerError)
	} else if h.IsStale() {
		http.Error(w,
			fmt.Sprintf("timestamp %v is older than %v", h.heartbeatTimestamp, h.heartbeatHealthyDuration),
			http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
