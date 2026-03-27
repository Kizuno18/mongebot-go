// Package engine - Worker represents a single viewer goroutine with its resources.
package engine

import (
	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
)

// Worker wraps a platform.Viewer with its assigned resources.
type Worker struct {
	id       string
	viewer   platform.Viewer
	proxy    *proxy.Proxy
	proxyMgr *proxy.Manager
}

// Stop gracefully stops the worker and releases its resources.
func (w *Worker) Stop() {
	w.viewer.Stop()
	if w.proxyMgr != nil {
		w.proxyMgr.Release(w.proxy)
	}
}

// Status returns the viewer status.
func (w *Worker) Status() platform.ViewerStatus {
	return w.viewer.Status()
}

// Metrics returns the viewer metrics.
func (w *Worker) Metrics() *platform.ViewerMetrics {
	return w.viewer.Metrics()
}
