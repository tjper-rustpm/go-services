// Package healthz provides and API enabling the support of service health
// checks. This is typically used when running in Kubernetes environment to
// manage and signal health status.
package healthz

import (
	"net/http"
	"sync"
)

// NewHTTP creates an HTTP instance.
func NewHTTP() *HTTP {
	return &HTTP{
		mutex:   new(sync.RWMutex),
		healthy: false,
	}
}

// HTTP provides an HTTP handler to correctly handle HTTP-based health checks.
type HTTP struct {
	mutex *sync.RWMutex
	// healthy indicates if the HTTP health check should report healthy to
	// clients.
	healthy bool
}

// ServeHTTP implements the http.Handler interface.
func (h *HTTP) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.mutex.RLock()
	status := h.healthy
	h.mutex.RUnlock()

	if status {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusServiceUnavailable)
}

// IsHealthy indicates if the HTTP instance is indicating it is healthy during
// health checks. See Healthy() and Sick() to mutate the health of the HTTP
// instance.
func (h *HTTP) IsHealthy() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.healthy
}

// Healthy mutates the HTTP instance to communicate a status of "healthy" during
// health checks.
func (h *HTTP) Healthy() {
	h.mutex.Lock()
	h.healthy = true
	h.mutex.Unlock()
}

// Sick mutates the HTTP instance to communicate a status of "sick" during
// health checks.
func (h *HTTP) Sick() {
	h.mutex.Lock()
	h.healthy = false
	h.mutex.Unlock()
}
