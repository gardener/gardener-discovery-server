// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func init() {
	prometheus.MustRegister(requestLatency, requestTotal, requestInFlight)
	metrics.Registry.MustRegister(requestLatency, requestTotal, requestInFlight)
}

const (
	subsystemName = "gardener_discovery_server"
)

var (
	requestLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:      "path_latency_seconds",
		Subsystem: subsystemName,
		Help:      "Histogram of the latency of processing HTTP requests",
	},
		[]string{"path"},
	)

	requestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "path_requests_total",
		Subsystem: subsystemName,
		Help:      "Total number of HTTP requests by path and code.",
	},
		[]string{"path", "code"},
	)

	requestInFlight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "path_requests_in_flight",
		Subsystem: subsystemName,
		Help:      "Number of currently server HTTP requests.",
	},
		[]string{"path"},
	)
)

// InstrumentHandler instruments the http handler with request generic metrics.
func InstrumentHandler(path string, handler http.Handler) http.Handler {
	var (
		label    = prometheus.Labels{"path": path}
		latency  = requestLatency.MustCurryWith(label)
		total    = requestTotal.MustCurryWith(label)
		inFlight = requestInFlight.With(label)
	)

	return promhttp.InstrumentHandlerDuration(
		latency,
		promhttp.InstrumentHandlerCounter(
			total,
			promhttp.InstrumentHandlerInFlight(inFlight, handler),
		),
	)
}
